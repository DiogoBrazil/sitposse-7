// Package importacao é o núcleo único de recepção de lotes (Wi-Fi agora; USB depois):
// recebe o manifesto de uma coleta, recebe as fotos em staging, confere tamanho e
// SHA-256 e deixa o lote aguardando a decisão do operador sobre o destino (ADR-0024).
// Na confirmação, publica os arquivos ANTES dos registros e aplica tudo numa
// transação que grava o destino da coleta, a decisão e o recibo.
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
	ErrLoteCancelado  = errors.New("este envio foi cancelado na Central; envie de novo")
	ErrJaDecidido     = errors.New("o destino deste lote já foi decidido")
	ErrDestinoTravado = errors.New("esta coleta já foi importada para outra equipe; o destino não pode mudar")
	ErrNaoAguardando  = errors.New("o lote ainda está sendo recebido")
)

// MaxBloco é o maior bloco aceito em PUT /v1/fotos (protocolo.md §3).
const MaxBloco = 1 << 20

// Dispositivo é o aparelho autenticado pela credencial.
type Dispositivo struct {
	InstalacaoUUID string
}

// Importador implementa a recepção. Seguro para uso concorrente.
type Importador struct {
	B          *banco.Banco
	PastaDados string
	CentralID  string
	Agora      func() time.Time
	mu         sync.Mutex // serializa escrita de fotos, importação e decisões
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
	if l.InstalacaoUUID != d.InstalacaoUUID {
		return nil, fmt.Errorf("%w: o lote é de outra instalação", ErrProibido)
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
		agora := im.agora().UnixMilli()
		if err := im.registrarColeta(ctx, tx, d, &l.Coleta, agora); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO lotes_recebidos (uuid, instalacao_uuid, coleta_uuid, codigo_informado, canal, status, manifesto_json, manifesto_sha256, recebido_em)
			VALUES (?, ?, ?, ?, 'wifi', 'recebendo', ?, ?, ?)`, l.LoteUUID, d.InstalacaoUUID, l.Coleta.UUID, l.Coleta.CodigoEquipe, string(bruto), hashManifesto, agora); err != nil {
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
		if status == contratos.StatusCancelado {
			return nil, ErrLoteCancelado
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return im.estadoLocked(ctx, l.LoteUUID)
}

// registrarColeta guarda a coleta do lote (só dados de recepção; nada é importado
// antes da decisão do operador). Uma coleta pertence a um único aparelho.
func (im *Importador) registrarColeta(ctx context.Context, tx *sql.Tx, d Dispositivo, c *contratos.Coleta, agora int64) error {
	var inst string
	var equipe sql.NullString
	err := tx.QueryRowContext(ctx, `SELECT instalacao_uuid, equipe_uuid FROM coletas WHERE uuid = ?`, c.UUID).Scan(&inst, &equipe)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		_, err = tx.ExecContext(ctx, `INSERT INTO coletas (uuid, instalacao_uuid, codigo_informado, codigo_original, created_at, updated_at, recebido_em)
			VALUES (?, ?, ?, ?, ?, ?, ?)`, c.UUID, d.InstalacaoUUID, c.CodigoEquipe, c.CodigoOriginal, ms(c.CreatedAt), ms(c.UpdatedAt), agora)
		return err
	case err != nil:
		return err
	case inst != d.InstalacaoUUID:
		return fmt.Errorf("%w: a coleta é de outro aparelho", ErrProibido)
	}
	if equipe.Valid {
		var vigente int
		if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM equipes e JOIN jornadas j ON j.uuid=e.jornada_uuid JOIN estudos s ON s.uuid=j.estudo_uuid WHERE e.uuid=? AND e.excluido_em IS NULL AND j.excluido_em IS NULL AND s.excluido_em IS NULL`, equipe.String).Scan(&vigente); err != nil {
			return err
		}
		if vigente == 0 {
			return fmt.Errorf("%w: o estudo de destino desta coleta foi excluído na Central", ErrProibido)
		}
	}
	_, err = tx.ExecContext(ctx, `UPDATE coletas SET codigo_informado = ?, recebido_em = ? WHERE uuid = ?`, c.CodigoEquipe, agora, c.UUID)
	return err
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
	var reciboJSON, motivo sql.NullString
	if err := im.B.Escrita.QueryRowContext(ctx, `SELECT status, recibo_json, motivo_cancelamento FROM lotes_recebidos WHERE uuid = ?`, lote).
		Scan(&status, &reciboJSON, &motivo); err != nil {
		return nil, err
	}
	est := &contratos.EstadoLote{ProtocoloVersao: contratos.ProtocoloVersao, LoteUUID: lote, Status: status, FotosFaltantes: []contratos.FotoFaltante{}}
	switch status {
	case contratos.StatusImportado:
		var r contratos.Recibo
		if err := json.Unmarshal([]byte(reciboJSON.String), &r); err != nil {
			return nil, fmt.Errorf("recibo persistido ilegível: %w", err)
		}
		est.Recibo = &r
		return est, nil
	case contratos.StatusCancelado:
		m := motivo.String
		est.Motivo = &m
		return est, nil
	case contratos.StatusAguardando:
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

// Importar confere as fotos do lote e o deixa aguardando a decisão do operador
// (ADR-0024). Se faltarem fotos (ou alguma tiver hash divergente), devolve o estado
// com fotos_faltantes e ErrFotosFaltantes, sem mudar nada. Lote já decidido devolve o
// estado atual (o mesmo recibo, se importado).
func (im *Importador) Importar(ctx context.Context, d Dispositivo, lote string) (*contratos.EstadoLote, error) {
	im.mu.Lock()
	defer im.mu.Unlock()
	var inst, status string
	err := im.B.Escrita.QueryRowContext(ctx, `SELECT instalacao_uuid, status FROM lotes_recebidos WHERE uuid = ?`, lote).Scan(&inst, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNaoEncontrado
	}
	if err != nil {
		return nil, err
	}
	if inst != d.InstalacaoUUID {
		return nil, ErrProibido
	}
	switch status {
	case contratos.StatusCancelado:
		return nil, ErrLoteCancelado
	case contratos.StatusImportado, contratos.StatusAguardando:
		return im.estadoLocked(ctx, lote) // reenvio/resposta perdida: mesmo estado
	}
	_, falt, err := im.conferirFotos(ctx, lote)
	if err != nil {
		return nil, err
	}
	if len(falt) > 0 {
		return &contratos.EstadoLote{ProtocoloVersao: contratos.ProtocoloVersao, LoteUUID: lote, Status: contratos.StatusRecebendo, FotosFaltantes: falt}, ErrFotosFaltantes
	}
	if _, err := im.B.Escrita.ExecContext(ctx, `UPDATE lotes_recebidos SET status = 'aguardando_confirmacao', aguardando_desde = ? WHERE uuid = ? AND status = 'recebendo'`,
		im.agora().UnixMilli(), lote); err != nil {
		return nil, err
	}
	return im.estadoLocked(ctx, lote)
}

// conferirFotos confere tamanho e SHA-256 de cada foto esperada em staging. Devolve as
// prontas para publicar e as que faltam (hash divergente: o .part é apagado e a foto
// volta a ser pedida do zero).
func (im *Importador) conferirFotos(ctx context.Context, lote string) ([]esperada, []contratos.FotoFaltante, error) {
	es, err := im.esperadas(ctx, lote)
	if err != nil {
		return nil, nil, err
	}
	var prontas []esperada
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
			return nil, nil, err
		}
		if st.Size() != e.tamanho || h != e.sha {
			_ = os.Remove(parte) // hash divergente: só esta foto é reenviada
			falt = append(falt, contratos.FotoFaltante{UUID: e.uuid, Offset: 0})
			continue
		}
		prontas = append(prontas, e)
	}
	return prontas, falt, nil
}

// publicar move as fotos conferidas para o lugar final (rename atômico no mesmo volume
// + fsync do diretório), ANTES da transação dos registros.
func (im *Importador) publicar(prontas []esperada) error {
	for _, e := range prontas {
		final := im.caminhoFinal(e.uuid)
		if err := os.MkdirAll(filepath.Dir(final), 0o700); err != nil {
			return err
		}
		if err := os.Rename(im.caminhoParte(e.uuid), final); err != nil {
			return fmt.Errorf("publicando foto %s: %w", e.uuid, err)
		}
		if err := syncDir(filepath.Dir(final)); err != nil {
			return err
		}
	}
	return nil
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
