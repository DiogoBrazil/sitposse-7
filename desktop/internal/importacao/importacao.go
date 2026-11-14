// Package importacao é o núcleo único de recepção de lotes (Wi-Fi agora; USB depois):
// recebe o manifesto, recebe as fotos em staging, confere tamanho e SHA-256, publica
// os arquivos ANTES dos registros e aplica tudo numa transação que grava o recibo.
//
// Referências: docs/synchronization/protocolo.md §5–6, integridade.md,
// desktop/docs/importacao-e-consolidacao.md.
package importacao

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"sitposse7/desktop/internal/banco"
	"sitposse7/desktop/internal/contratos"
)

// Erros com significado HTTP (o syncserver traduz).
var (
	ErrInvalido       = errors.New("requisição inválida")
	ErrVersao         = errors.New("versão incompatível")
	ErrProibido       = errors.New("operação não permitida para este aparelho")
	ErrNaoEncontrado  = errors.New("não encontrado")
	ErrConflitoLote   = errors.New("já existe um lote com este UUID e conteúdo diferente")
	ErrOffset         = errors.New("offset diferente do recebido até agora")
	ErrFotosFaltantes = errors.New("ainda faltam fotos para importar o lote")
)

// MaxBloco é o maior bloco aceito em PUT /v1/fotos (protocolo.md §3).
const MaxBloco = 1 << 20

// Dispositivo é o aparelho autenticado pela credencial.
type Dispositivo struct {
	InstalacaoUUID string
	EquipeUUID     string
	OperacaoUUID   string
}

// Importador implementa a recepção. Seguro para uso concorrente.
type Importador struct {
	B          *banco.Banco
	PastaDados string
	CentralID  string
	Agora      func() time.Time
	mu         sync.Mutex // serializa escrita de fotos e importação
}

func (im *Importador) agora() time.Time {
	if im.Agora != nil {
		return im.Agora()
	}
	return time.Now()
}

func (im *Importador) dirStaging() string { return filepath.Join(im.PastaDados, "staging") }

func (im *Importador) caminhoParte(foto string) string {
	return filepath.Join(im.dirStaging(), foto+".part")
}

// CaminhoRelativoFoto é onde a foto publicada fica dentro da pasta de dados.
func CaminhoRelativoFoto(foto string) string {
	return filepath.ToSlash(filepath.Join("fotos", foto[:2], foto+".jpg"))
}

func (im *Importador) caminhoFinal(foto string) string {
	return filepath.Join(im.PastaDados, filepath.FromSlash(CaminhoRelativoFoto(foto)))
}

// ---------- PUT /v1/lotes/{lote} ----------

// ReceberLote grava o manifesto (idempotente) e informa as fotos que faltam.
func (im *Importador) ReceberLote(ctx context.Context, d Dispositivo, loteURL string, bruto []byte) (*contratos.EstadoLote, error) {
	var l contratos.Lote
	dec := json.NewDecoder(bytes.NewReader(bruto))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&l); err != nil {
		return nil, fmt.Errorf("%w: JSON do lote: %v", ErrInvalido, err)
	}
	if l.ProtocoloVersao != contratos.ProtocoloVersao || l.SchemaVersao != contratos.SchemaVersao {
		return nil, fmt.Errorf("%w: protocolo %d/esquema %d (esta Central aceita %d/%d)", ErrVersao,
			l.ProtocoloVersao, l.SchemaVersao, contratos.ProtocoloVersao, contratos.SchemaVersao)
	}
	if err := contratos.ValidarEnvelope(&l); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalido, err)
	}
	if l.LoteUUID != loteURL {
		return nil, fmt.Errorf("%w: lote_uuid diferente do endereço", ErrInvalido)
	}
	if l.InstalacaoUUID != d.InstalacaoUUID || l.Equipe.UUID != d.EquipeUUID {
		return nil, fmt.Errorf("%w: o lote é de outra instalação ou equipe", ErrProibido)
	}
	soma := sha256.Sum256(bruto)
	hashManifesto := hex.EncodeToString(soma[:])

	im.mu.Lock()
	defer im.mu.Unlock()
	tx, err := im.B.Escrita.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var inst, status, hashExistente string
	err = tx.QueryRowContext(ctx, `SELECT instalacao_uuid, status, manifesto_sha256 FROM lotes_recebidos WHERE uuid = ?`, l.LoteUUID).
		Scan(&inst, &status, &hashExistente)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if _, err := tx.ExecContext(ctx, `INSERT INTO lotes_recebidos (uuid, instalacao_uuid, equipe_uuid, canal, status, manifesto_json, manifesto_sha256, recebido_em)
			VALUES (?, ?, ?, 'wifi', 'recebendo', ?, ?, ?)`, l.LoteUUID, d.InstalacaoUUID, d.EquipeUUID, string(bruto), hashManifesto, im.agora().UnixMilli()); err != nil {
			return nil, err
		}
		for _, f := range l.Fotos {
			if f.ExcluidoEm != nil || contratos.ValidarFoto(&f) != nil {
				continue // tombstone ou inválida: não precisa dos bytes
			}
			if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO fotos_esperadas (foto_uuid, lote_uuid, tamanho_bytes, sha256) VALUES (?, ?, ?, ?)`,
				f.UUID, l.LoteUUID, f.TamanhoBytes, f.SHA256); err != nil {
				return nil, err
			}
		}
	case err != nil:
		return nil, err
	default:
		if inst != d.InstalacaoUUID {
			return nil, fmt.Errorf("%w: lote de outra instalação", ErrProibido)
		}
		if hashExistente != hashManifesto {
			return nil, ErrConflitoLote
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return im.estadoLocked(ctx, l.LoteUUID)
}

// EstadoLote responde GET /v1/lotes/{lote}.
func (im *Importador) EstadoLote(ctx context.Context, d Dispositivo, lote string) (*contratos.EstadoLote, error) {
	im.mu.Lock()
	defer im.mu.Unlock()
	var inst string
	err := im.B.Escrita.QueryRowContext(ctx, `SELECT instalacao_uuid FROM lotes_recebidos WHERE uuid = ?`, lote).Scan(&inst)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNaoEncontrado
	}
	if err != nil {
		return nil, err
	}
	if inst != d.InstalacaoUUID {
		return nil, ErrProibido
	}
	return im.estadoLocked(ctx, lote)
}

func (im *Importador) estadoLocked(ctx context.Context, lote string) (*contratos.EstadoLote, error) {
	var status string
	var reciboJSON sql.NullString
	if err := im.B.Escrita.QueryRowContext(ctx, `SELECT status, recibo_json FROM lotes_recebidos WHERE uuid = ?`, lote).Scan(&status, &reciboJSON); err != nil {
		return nil, err
	}
	est := &contratos.EstadoLote{ProtocoloVersao: contratos.ProtocoloVersao, LoteUUID: lote, Status: status, FotosFaltantes: []contratos.FotoFaltante{}}
	if status == "importado" {
		var r contratos.Recibo
		if err := json.Unmarshal([]byte(reciboJSON.String), &r); err != nil {
			return nil, fmt.Errorf("recibo persistido ilegível: %w", err)
		}
		est.Recibo = &r
		return est, nil
	}
	falt, err := im.faltantes(ctx, lote)
	if err != nil {
		return nil, err
	}
	est.FotosFaltantes = falt
	return est, nil
}

type esperada struct {
	uuid    string
	tamanho int64
	sha     string
}

func (im *Importador) esperadas(ctx context.Context, lote string) ([]esperada, error) {
	rows, err := im.B.Escrita.QueryContext(ctx, `SELECT foto_uuid, tamanho_bytes, sha256 FROM fotos_esperadas WHERE lote_uuid = ? ORDER BY foto_uuid`, lote)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var es []esperada
	for rows.Next() {
		var e esperada
		if err := rows.Scan(&e.uuid, &e.tamanho, &e.sha); err != nil {
			return nil, err
		}
		es = append(es, e)
	}
	return es, rows.Err()
}

// publicada indica se a Central já tem o arquivo final desta foto com o hash esperado
// (registrado no banco ou publicado por uma importação anterior).
func (im *Importador) publicada(e esperada) bool {
	st, err := os.Stat(im.caminhoFinal(e.uuid))
	if err != nil || st.Size() != e.tamanho {
		return false
	}
	h, err := hashArquivo(im.caminhoFinal(e.uuid))
	return err == nil && h == e.sha
}

func (im *Importador) faltantes(ctx context.Context, lote string) ([]contratos.FotoFaltante, error) {
	es, err := im.esperadas(ctx, lote)
	if err != nil {
		return nil, err
	}
	falt := []contratos.FotoFaltante{}
	for _, e := range es {
		if im.publicada(e) {
			continue
		}
		var off int64
		if st, err := os.Stat(im.caminhoParte(e.uuid)); err == nil {
			off = st.Size()
		}
		if off > e.tamanho {
			_ = os.Remove(im.caminhoParte(e.uuid))
			off = 0
		}
		if off < e.tamanho {
			falt = append(falt, contratos.FotoFaltante{UUID: e.uuid, Offset: off})
		}
	}
	return falt, nil
}

// ---------- PUT/GET /v1/fotos/{foto} ----------

func (im *Importador) fotoEsperada(ctx context.Context, d Dispositivo, foto string) (esperada, error) {
	e := esperada{uuid: foto}
	err := im.B.Escrita.QueryRowContext(ctx, `SELECT fe.tamanho_bytes, fe.sha256 FROM fotos_esperadas fe
		JOIN lotes_recebidos l ON l.uuid = fe.lote_uuid
		WHERE fe.foto_uuid = ? AND l.instalacao_uuid = ? AND l.status = 'recebendo'
		ORDER BY l.recebido_em DESC LIMIT 1`, foto, d.InstalacaoUUID).Scan(&e.tamanho, &e.sha)
	if errors.Is(err, sql.ErrNoRows) {
		return e, fmt.Errorf("%w: foto não esperada em nenhum lote em recepção deste aparelho", ErrNaoEncontrado)
	}
	return e, err
}

// EstadoFoto informa quantos bytes da foto já estão em staging e se ela já foi publicada.
func (im *Importador) EstadoFoto(ctx context.Context, d Dispositivo, foto string) (offset int64, existe bool, err error) {
	im.mu.Lock()
	defer im.mu.Unlock()
	e, err := im.fotoEsperada(ctx, d, foto)
	if err != nil {
		return 0, false, err
	}
	if im.publicada(e) {
		return e.tamanho, true, nil
	}
	if st, err := os.Stat(im.caminhoParte(foto)); err == nil {
		return st.Size(), false, nil
	}
	return 0, false, nil
}

// ReceberBlocoFoto acrescenta um bloco (≤ 1 MiB) à foto em staging, com fsync.
// Devolve o novo offset. Se offset não bater com o já recebido, devolve ErrOffset e o atual.
func (im *Importador) ReceberBlocoFoto(ctx context.Context, d Dispositivo, foto string, offset int64, corpo io.Reader) (int64, error) {
	im.mu.Lock()
	defer im.mu.Unlock()
	e, err := im.fotoEsperada(ctx, d, foto)
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(im.dirStaging(), 0o700); err != nil {
		return 0, err
	}
	parte := im.caminhoParte(foto)
	var atual int64
	if st, err := os.Stat(parte); err == nil {
		atual = st.Size()
	}
	if offset != atual {
		return atual, ErrOffset
	}
	dados, err := io.ReadAll(io.LimitReader(corpo, MaxBloco+1))
	if err != nil {
		return atual, err
	}
	if len(dados) == 0 || len(dados) > MaxBloco {
		return atual, fmt.Errorf("%w: bloco deve ter de 1 byte a 1 MiB", ErrInvalido)
	}
	if atual+int64(len(dados)) > e.tamanho {
		return atual, fmt.Errorf("%w: bloco ultrapassa o tamanho declarado da foto", ErrInvalido)
	}
	f, err := os.OpenFile(parte, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return atual, err
	}
	if _, err := f.Write(dados); err != nil {
		f.Close()
		_ = os.Truncate(parte, atual) // não deixa bloco pela metade
		return atual, err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		_ = os.Truncate(parte, atual)
		return atual, err
	}
	if err := f.Close(); err != nil {
		return atual, err
	}
	return atual + int64(len(dados)), nil
}

// ---------- POST /v1/lotes/{lote}/importar ----------

// Importar confere e publica as fotos e aplica o lote numa transação que grava o recibo.
// Se faltarem fotos (ou alguma tiver hash divergente), devolve o estado com
// fotos_faltantes e ErrFotosFaltantes, sem aplicar nada.
func (im *Importador) Importar(ctx context.Context, d Dispositivo, lote string) (*contratos.EstadoLote, error) {
	im.mu.Lock()
	defer im.mu.Unlock()
	var inst, status, manifesto string
	err := im.B.Escrita.QueryRowContext(ctx, `SELECT instalacao_uuid, status, manifesto_json FROM lotes_recebidos WHERE uuid = ?`, lote).
		Scan(&inst, &status, &manifesto)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNaoEncontrado
	}
	if err != nil {
		return nil, err
	}
	if inst != d.InstalacaoUUID {
		return nil, ErrProibido
	}
	if status == "importado" {
		return im.estadoLocked(ctx, lote) // reenvio/ACK perdido: mesmo recibo
	}

	// 1–2. Conferir tamanho e SHA-256 de cada foto em staging.
	es, err := im.esperadas(ctx, lote)
	if err != nil {
		return nil, err
	}
	var paraPublicar []esperada
	falt := []contratos.FotoFaltante{}
	for _, e := range es {
		if im.publicada(e) {
			continue
		}
		parte := im.caminhoParte(e.uuid)
		st, err := os.Stat(parte)
		if err != nil || st.Size() < e.tamanho {
			var off int64
			if err == nil {
				off = st.Size()
			}
			falt = append(falt, contratos.FotoFaltante{UUID: e.uuid, Offset: off})
			continue
		}
		h, err := hashArquivo(parte)
		if err != nil {
			return nil, err
		}
		if st.Size() != e.tamanho || h != e.sha {
			_ = os.Remove(parte) // hash divergente: só esta foto é reenviada
			falt = append(falt, contratos.FotoFaltante{UUID: e.uuid, Offset: 0})
			continue
		}
		paraPublicar = append(paraPublicar, e)
	}
	if len(falt) > 0 {
		return &contratos.EstadoLote{ProtocoloVersao: contratos.ProtocoloVersao, LoteUUID: lote, Status: "recebendo", FotosFaltantes: falt}, ErrFotosFaltantes
	}

	// 3. Publicar (rename atômico no mesmo volume + fsync do diretório) ANTES dos registros.
	for _, e := range paraPublicar {
		final := im.caminhoFinal(e.uuid)
		if err := os.MkdirAll(filepath.Dir(final), 0o700); err != nil {
			return nil, err
		}
		if err := os.Rename(im.caminhoParte(e.uuid), final); err != nil {
			return nil, fmt.Errorf("publicando foto %s: %w", e.uuid, err)
		}
		if err := syncDir(filepath.Dir(final)); err != nil {
			return nil, err
		}
	}

	// 4. Transação única: registros, aplicações, auditoria e recibo.
	var l contratos.Lote
	if err := json.Unmarshal([]byte(manifesto), &l); err != nil {
		return nil, fmt.Errorf("manifesto persistido ilegível: %w", err)
	}
	recibo, err := im.aplicar(ctx, d, &l)
	if err != nil {
		return nil, err
	}
	return &contratos.EstadoLote{ProtocoloVersao: contratos.ProtocoloVersao, LoteUUID: lote, Status: "importado",
		FotosFaltantes: []contratos.FotoFaltante{}, Recibo: recibo}, nil
}

// Ack registra que o aparelho gravou o recibo (informativo).
func (im *Importador) Ack(ctx context.Context, d Dispositivo, lote string) error {
	res, err := im.B.Escrita.ExecContext(ctx, `UPDATE lotes_recebidos SET ack_em = coalesce(ack_em, ?) WHERE uuid = ? AND instalacao_uuid = ? AND status = 'importado'`,
		im.agora().UnixMilli(), lote, d.InstalacaoUUID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNaoEncontrado
	}
	return nil
}

// ---------- utilidades de arquivo ----------

func hashArquivo(caminho string) (string, error) {
	f, err := os.Open(caminho)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// syncDir grava no disco a entrada de diretório criada pelo rename. No Windows não
// é possível dar fsync em diretório; o NTFS registra o rename no seu journal.
func syncDir(dir string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}
