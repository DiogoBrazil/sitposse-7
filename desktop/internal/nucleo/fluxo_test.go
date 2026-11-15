package nucleo

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"

	"sitposse7/desktop/internal/codigos"
	"sitposse7/desktop/internal/consultas"
	"sitposse7/desktop/internal/contratos"
	"sitposse7/desktop/internal/estudos"
	"sitposse7/desktop/internal/importacao"
	"sitposse7/desktop/internal/syncserver"
)

// ---------- ambiente de teste: Central real em HTTPS ----------

type amb struct {
	t      *testing.T
	n      *Nucleo
	base   string
	http   *http.Client
	cred   string
	inst   string
	coleta contratos.Coleta
	pasta  string
	// Estudo de teste: jornada 1 com as equipes Alfa e Bravo.
	estudo *estudos.EstudoDetalhe
	alfa   estudos.Equipe
	bravo  estudos.Equipe
}

func novoAmb(t *testing.T) *amb {
	t.Helper()
	pasta := t.TempDir()
	n, err := Abrir(context.Background(), pasta, []net.IP{net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	n.Pareamento.Espera = 5 * time.Second
	if err := n.Servidor.Iniciar("127.0.0.1:0"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(n.Fechar)
	a := &amb{t: t, n: n, pasta: pasta,
		base: "https://127.0.0.1:" + strconv.Itoa(n.Servidor.Porta()),
		http: &http.Client{Timeout: 20 * time.Second, Transport: &http.Transport{TLSClientConfig: syncserver.ConfigClienteComPin(n.Servidor.Identidade.Pin())}},
		inst: uuid.NewString(),
	}
	e, err := n.Estudos.CriarEstudo(context.Background(), "Fazenda Santa Maria", "2026-10-10", []string{"Equipe Alfa", "Equipe Bravo"})
	if err != nil {
		t.Fatal(err)
	}
	a.estudo, a.alfa, a.bravo = e, e.ListaJornadas[0].Equipes[0], e.ListaJornadas[0].Equipes[1]
	a.coleta = a.novaColeta(a.alfa.Codigo)
	return a
}

func (a *amb) novaColeta(codigo string) contratos.Coleta {
	agora := contratos.FormatarInstante(time.Now().UnixMilli())
	return contratos.Coleta{UUID: uuid.NewString(), CodigoEquipe: codigo, CodigoOriginal: codigo, CreatedAt: agora, UpdatedAt: agora, Rev: 1}
}

// codigoInexistente gera um código de formato válido que nenhuma equipe tem.
func codigoInexistente(t *testing.T) string {
	c, err := codigos.Gerar(codigos.PrefixoEquipe, nil)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func (a *amb) req(metodo, caminho string, corpo []byte, auth bool) (int, []byte) {
	a.t.Helper()
	r, err := http.NewRequest(metodo, a.base+caminho, bytes.NewReader(corpo))
	if err != nil {
		a.t.Fatal(err)
	}
	if auth {
		r.Header.Set("Authorization", "Bearer "+a.cred)
	}
	resp, err := a.http.Do(r)
	if err != nil {
		a.t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b
}

func (a *amb) jsonReq(metodo, caminho string, v any, auth bool) (int, []byte) {
	a.t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		a.t.Fatal(err)
	}
	return a.req(metodo, caminho, b, auth)
}

// parear gera o token e confirma o pedido como o operador faria.
func (a *amb) parear() {
	a.t.Helper()
	ctx := context.Background()
	token, _, err := a.n.Pareamento.GerarToken(ctx)
	if err != nil {
		a.t.Fatal(err)
	}
	go func() { // operador confirma assim que o pedido aparece
		for i := 0; i < 200; i++ {
			if ps := a.n.Pareamento.Pendentes(); len(ps) > 0 {
				_ = a.n.Pareamento.Decidir(ps[0].ID, true)
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()
	st, corpo := a.jsonReq("POST", "/v1/pareamento", contratos.PedidoPareamento{ProtocoloVersao: 2, SchemaVersao: 2, Token: token, InstalacaoUUID: a.inst}, false)
	if st != 200 {
		a.t.Fatalf("pareamento: %d %s", st, corpo)
	}
	var resp contratos.RespostaPareamento
	json.Unmarshal(corpo, &resp)
	a.cred = resp.CredencialDispositivo
	// Token é de uso único.
	st, _ = a.jsonReq("POST", "/v1/pareamento", contratos.PedidoPareamento{ProtocoloVersao: 2, SchemaVersao: 2, Token: token, InstalacaoUUID: uuid.NewString()}, false)
	if st != http.StatusGone {
		a.t.Fatalf("token reutilizado: esperado 410, obteve %d", st)
	}
}

func fotoAleatoria(t *testing.T, n int) ([]byte, string) {
	b := make([]byte, n)
	rand.Read(b)
	h := sha256.Sum256(b)
	return b, hex.EncodeToString(h[:])
}

func (a *amb) novoLote(props []contratos.Propriedade, itens []contratos.Item, fotos []contratos.Foto) contratos.Lote {
	return contratos.Lote{ProtocoloVersao: 2, SchemaVersao: 2, LoteUUID: uuid.NewString(), InstalacaoUUID: a.inst,
		CriadoEm: contratos.FormatarInstante(time.Now().UnixMilli()), Coleta: a.coleta, Propriedades: props, Itens: itens, Fotos: fotos}
}

func (a *amb) prop(nome string, rev int64) contratos.Propriedade {
	agora := contratos.FormatarInstante(time.Now().UnixMilli())
	obs := "Família pede prazo.\nIdoso acamado."
	return contratos.Propriedade{UUID: uuid.NewString(), ColetaUUID: a.coleta.UUID, NomePropriedade: nome, ResponsavelFamilia: "José da Silva",
		ObservacoesRelevantes: &obs, BovAdultos: 12, CreatedAt: agora, UpdatedAt: agora, Rev: rev}
}

func (a *amb) foto(prop string, tamanho int64, sha string) contratos.Foto {
	agora := contratos.FormatarInstante(time.Now().UnixMilli())
	return contratos.Foto{UUID: uuid.NewString(), PropriedadeUUID: prop, Mime: "image/jpeg", TamanhoBytes: tamanho, SHA256: sha,
		Largura: 2560, Altura: 1920, CapturadaEm: agora, CreatedAt: agora, UpdatedAt: agora, Rev: 1}
}

func (a *amb) putLote(l contratos.Lote) contratos.EstadoLote {
	a.t.Helper()
	st, corpo := a.jsonReq("PUT", "/v1/lotes/"+l.LoteUUID, l, true)
	if st != 200 {
		a.t.Fatalf("PUT lote: %d %s", st, corpo)
	}
	var e contratos.EstadoLote
	json.Unmarshal(corpo, &e)
	return e
}

func (a *amb) enviarFoto(uuid string, dados []byte, de int64) {
	a.t.Helper()
	for off := de; off < int64(len(dados)); {
		fim := min(off+importacao.MaxBloco, int64(len(dados)))
		st, corpo := a.req("PUT", fmt.Sprintf("/v1/fotos/%s?offset=%d", uuid, off), dados[off:fim], true)
		if st != 200 {
			a.t.Fatalf("PUT foto offset %d: %d %s", off, st, corpo)
		}
		off = fim
	}
}

func (a *amb) importar(lote string) (int, contratos.EstadoLote) {
	a.t.Helper()
	st, corpo := a.req("POST", "/v1/lotes/"+lote+"/importar", nil, true)
	var e contratos.EstadoLote
	if err := json.Unmarshal(corpo, &e); err != nil {
		a.t.Fatalf("importar: %d %s", st, corpo)
	}
	return st, e
}

// decidir aplica a decisão do operador (equipe "" = cancelar).
func (a *amb) decidir(lote, equipe string) (*contratos.EstadoLote, error) {
	return a.n.Importador.Decidir(context.Background(), lote, importacao.Decisao{Cancelar: equipe == "", EquipeUUID: equipe})
}

// importarConfirmado faz o importar (que deixa o lote aguardando) e confirma o destino
// sugerido pelo código, como o operador; devolve o estado final visto pelo celular.
func (a *amb) importarConfirmado(lote string) contratos.EstadoLote {
	a.t.Helper()
	st, e := a.importar(lote)
	if st != 200 || (e.Status != contratos.StatusAguardando && e.Status != contratos.StatusImportado) {
		a.t.Fatalf("importar: %d %+v", st, e)
	}
	if e.Status == contratos.StatusAguardando {
		ls, err := a.n.Importador.Aguardando(context.Background())
		if err != nil {
			a.t.Fatal(err)
		}
		equipe := ""
		for _, x := range ls {
			if x.LoteUUID == lote {
				if x.Travado {
					equipe = x.DestinoTravado.EquipeUUID
				} else if x.Destino != nil {
					equipe = x.Destino.EquipeUUID
				}
			}
		}
		if equipe == "" {
			a.t.Fatalf("lote %s sem destino sugerido", lote)
		}
		if _, err := a.decidir(lote, equipe); err != nil {
			a.t.Fatal(err)
		}
	}
	return a.getLote(lote)
}

func (a *amb) getLote(lote string) contratos.EstadoLote {
	a.t.Helper()
	st, corpo := a.req("GET", "/v1/lotes/"+lote, nil, true)
	var e contratos.EstadoLote
	if st != 200 || json.Unmarshal(corpo, &e) != nil {
		a.t.Fatalf("GET lote: %d %s", st, corpo)
	}
	return e
}

func (a *amb) contar(sql string, args ...any) int {
	a.t.Helper()
	var n int
	if err := a.n.B.Leitura.QueryRow(sql, args...).Scan(&n); err != nil {
		a.t.Fatal(err)
	}
	return n
}

// Invariante 1 (integridade.md): toda foto vigente aponta para arquivo com o SHA-256 registrado.
func (a *amb) conferirInvariantesFotos() {
	a.t.Helper()
	rows, err := a.n.B.Leitura.Query(`SELECT uuid, caminho_relativo, sha256 FROM fotografias WHERE excluido_em IS NULL`)
	if err != nil {
		a.t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var u, c, h string
		rows.Scan(&u, &c, &h)
		b, err := os.ReadFile(filepath.Join(a.pasta, filepath.FromSlash(c)))
		if err != nil {
			a.t.Fatalf("foto %s sem arquivo: %v", u, err)
		}
		s := sha256.Sum256(b)
		if hex.EncodeToString(s[:]) != h {
			a.t.Fatalf("foto %s com hash divergente", u)
		}
	}
}

// ---------- cenários (testing/estrategia.md §4) ----------

// Cenários 1 (primeira carga), 4 (mesmo lote 3×), 5 (ACK perdido) e 6 (foto retomada).
func TestFluxoCompleto(t *testing.T) {
	a := novoAmb(t)
	a.parear()

	p1, p2 := a.prop("Sítio Boa Esperança", 3), a.prop("Chácara Dois", 1)
	agora := p1.CreatedAt
	outros := "Codorna"
	itens := []contratos.Item{
		{UUID: uuid.NewString(), PropriedadeUUID: p1.UUID, Categoria: "eletrodomestico", CodigoCatalogo: "eletro.geladeira", Quantidade: 1, CreatedAt: agora, UpdatedAt: agora, Rev: 1},
		{UUID: uuid.NewString(), PropriedadeUUID: p1.UUID, Categoria: "ave", CodigoCatalogo: "ave.outros", NomePersonalizado: &outros, Quantidade: 40, CreatedAt: agora, UpdatedAt: agora, Rev: 1},
	}
	dados1, h1 := fotoAleatoria(t, 2_500_000) // 3 blocos
	dados2, h2 := fotoAleatoria(t, 300_000)
	f1, f2 := a.foto(p1.UUID, int64(len(dados1)), h1), a.foto(p2.UUID, int64(len(dados2)), h2)
	l := a.novoLote([]contratos.Propriedade{p1, p2}, itens, []contratos.Foto{f1, f2})

	est := a.putLote(l)
	if est.Status != "recebendo" || len(est.FotosFaltantes) != 2 {
		t.Fatalf("estado inicial: %+v", est)
	}
	// Importar antes das fotos: 409 com as faltantes, nada aplicado.
	if st, e := a.importar(l.LoteUUID); st != http.StatusConflict || len(e.FotosFaltantes) != 2 {
		t.Fatalf("importar sem fotos: %d %+v", st, e)
	}
	if a.contar(`SELECT count(*) FROM propriedades`) != 0 {
		t.Fatal("aplicou registros sem as fotos")
	}

	// Foto 1: envia o 1º bloco, "cai", retoma pelo offset informado pela Central (cenário 6).
	st, _ := a.req("PUT", "/v1/fotos/"+f1.UUID+"?offset=0", dados1[:importacao.MaxBloco], true)
	if st != 200 {
		t.Fatal(st)
	}
	st, corpo := a.req("PUT", "/v1/fotos/"+f1.UUID+"?offset=0", dados1[:importacao.MaxBloco], true) // reenvio do mesmo bloco
	if st != http.StatusConflict {
		t.Fatalf("offset errado aceito: %d", st)
	}
	var erro syncserver.ErroAPI
	json.Unmarshal(corpo, &erro)
	if erro.Offset == nil || *erro.Offset != importacao.MaxBloco {
		t.Fatalf("offset informado: %+v", erro)
	}
	if est := a.putLote(l); est.FotosFaltantes[0].Offset+est.FotosFaltantes[1].Offset != importacao.MaxBloco {
		t.Fatalf("estado não reflete o parcial: %+v", est)
	}
	a.enviarFoto(f1.UUID, dados1, *erro.Offset)
	a.enviarFoto(f2.UUID, dados2, 0)

	// Fotos completas: o lote passa a aguardar o operador; nada aplicado ainda (ADR-0024).
	st, e := a.importar(l.LoteUUID)
	if st != 200 || e.Status != contratos.StatusAguardando || e.Recibo != nil || e.Motivo != nil {
		t.Fatalf("importar: %d %+v", st, e)
	}
	if a.contar(`SELECT count(*) FROM propriedades`) != 0 {
		t.Fatal("aplicou antes da decisão do operador")
	}
	if g := a.getLote(l.LoteUUID); g.Status != contratos.StatusAguardando {
		t.Fatalf("GET aguardando: %+v", g)
	}
	ls, err := a.n.Importador.Aguardando(context.Background())
	if err != nil || len(ls) != 1 {
		t.Fatalf("aguardando: %v %+v", err, ls)
	}
	if x := ls[0]; x.Destino == nil || x.Destino.EquipeUUID != a.alfa.UUID || x.Destino.JornadaData != "2026-10-10" || x.Destino.EstudoNome != "Fazenda Santa Maria" ||
		x.Novos != 2 || x.Atualizados != 0 || x.Excluidos != 0 || x.Fotos != 2 || x.Travado || x.CodigoExibicao != codigos.Exibir(a.alfa.Codigo) {
		t.Fatalf("resumo para o operador: %+v", x)
	}
	if _, err := a.decidir(l.LoteUUID, a.alfa.UUID); err != nil {
		t.Fatal(err)
	}
	e = a.getLote(l.LoteUUID)
	if e.Status != contratos.StatusImportado || e.Recibo == nil {
		t.Fatalf("importação: %+v", e)
	}
	r := e.Recibo
	if r.ReciboSHA256 != r.CalcularHash() || len(r.Resultados) != 1+2+2+2 {
		t.Fatalf("recibo: %+v", r)
	}
	if d := r.Destino; d.EquipeCodigo != a.alfa.Codigo || d.CodigoInformado != a.alfa.Codigo || d.Corrigido || d.JornadaNumero != 1 || d.EstudoCodigo != a.estudo.Codigo {
		t.Fatalf("destino no recibo: %+v", d)
	}
	if r.Resultados[0].Entidade != "coleta" || r.Resultados[0].Resultado != "aplicado" {
		t.Errorf("coleta: %+v", r.Resultados[0])
	}
	for _, x := range r.Resultados[1:] {
		if x.Resultado != "aplicado" {
			t.Errorf("%s %s: %s %s", x.Entidade, x.UUID, x.Resultado, x.Motivo)
		}
	}
	if a.contar(`SELECT count(*) FROM v_propriedades_vigentes`) != 2 || a.contar(`SELECT count(*) FROM v_itens_vigentes`) != 2 || a.contar(`SELECT count(*) FROM v_fotos_vigentes`) != 2 {
		t.Fatal("contagens divergentes após importar")
	}
	if a.contar(`SELECT count(*) FROM itens_propriedade WHERE nome_normalizado = 'codorna'`) != 1 {
		t.Fatal("nome normalizado não gravado")
	}
	if a.contar(`SELECT count(*) FROM propriedades WHERE equipe_uuid = ? AND coleta_uuid = ? AND observacoes_relevantes LIKE '%acamado%'`, a.alfa.UUID, a.coleta.UUID) != 2 {
		t.Fatal("equipe, coleta ou observações não gravadas")
	}
	if a.contar(`SELECT count(*) FROM v_propriedades_vigentes WHERE estudo_uuid = ? AND jornada_numero = 1`, a.estudo.UUID) != 2 {
		t.Fatal("visão vigente sem estudo/jornada")
	}
	if a.contar(`SELECT count(*) FROM decisoes_destino WHERE acao = 'confirmar' AND equipe_uuid = ?`, a.alfa.UUID) != 1 {
		t.Fatal("decisão não registrada")
	}
	// Decidir de novo: já decidido, nada muda.
	if _, err := a.decidir(l.LoteUUID, a.alfa.UUID); !errors.Is(err, importacao.ErrJaDecidido) {
		t.Fatalf("decisão repetida: %v", err)
	}
	a.conferirInvariantesFotos()

	// Cenário 5 (ACK perdido) e 4 (mesmo lote 3×): o mesmo recibo, nada reaplicado.
	aud := a.contar(`SELECT count(*) FROM auditoria`)
	for i := 0; i < 3; i++ {
		if est := a.putLote(l); est.Status != "importado" || est.Recibo.ReciboSHA256 != r.ReciboSHA256 {
			t.Fatalf("PUT repetido %d: %+v", i, est)
		}
		if st, e := a.importar(l.LoteUUID); st != 200 || e.Recibo == nil || e.Recibo.ReciboSHA256 != r.ReciboSHA256 {
			t.Fatalf("importar repetido %d: %d", i, st)
		}
		st, corpo := a.req("GET", "/v1/lotes/"+l.LoteUUID, nil, true)
		var g contratos.EstadoLote
		json.Unmarshal(corpo, &g)
		if st != 200 || g.Recibo.ReciboSHA256 != r.ReciboSHA256 {
			t.Fatalf("GET lote %d: %d", i, st)
		}
	}
	if a.contar(`SELECT count(*) FROM auditoria`) != aud || a.contar(`SELECT count(*) FROM aplicacoes`) != len(r.Resultados) {
		t.Fatal("lote repetido gerou novas aplicações")
	}
	if st, _ := a.req("POST", "/v1/lotes/"+l.LoteUUID+"/ack", nil, true); st != http.StatusNoContent {
		t.Fatalf("ack: %d", st)
	}

	// Mesmo UUID de lote com conteúdo diferente: recusado (não sobrescreve o recibo).
	l2 := l
	l2.Propriedades = []contratos.Propriedade{p2}
	if st, _ := a.jsonReq("PUT", "/v1/lotes/"+l.LoteUUID, l2, true); st != http.StatusConflict {
		t.Fatalf("lote divergente: %d", st)
	}

	// Lote novo com as mesmas revs: tudo "duplicado"; com rev maior: "aplicado".
	p1b := p1
	p1b.Rev, p1b.BovAdultos = 4, 99
	l3 := a.novoLote([]contratos.Propriedade{p1b, p2}, nil, []contratos.Foto{f1})
	if est := a.putLote(l3); len(est.FotosFaltantes) != 0 {
		t.Fatalf("foto já publicada pedida de novo: %+v", est.FotosFaltantes)
	}
	e3 := a.importarConfirmado(l3.LoteUUID)
	got := map[string]string{}
	for _, x := range e3.Recibo.Resultados {
		got[x.UUID] = x.Resultado
	}
	if got[p1.UUID] != "aplicado" || got[p2.UUID] != "duplicado" || got[f1.UUID] != "duplicado" || got[a.coleta.UUID] != "duplicado" {
		t.Fatalf("resultados do 2º lote: %v", got)
	}
	if a.contar(`SELECT bov_adultos FROM propriedades WHERE uuid = ?`, p1.UUID) != 99 {
		t.Fatal("rev maior não aplicada")
	}
}

// Cenário 7 (hash divergente) e 8 (tombstone).
func TestHashDivergenteETombstone(t *testing.T) {
	a := novoAmb(t)
	a.parear()
	p := a.prop("Sítio Hash", 1)
	dados, h := fotoAleatoria(t, 50_000)
	f := a.foto(p.UUID, int64(len(dados)), h)
	l := a.novoLote([]contratos.Propriedade{p}, nil, []contratos.Foto{f})
	a.putLote(l)

	ruim := bytes.Clone(dados)
	ruim[100] ^= 0xFF
	a.enviarFoto(f.UUID, ruim, 0)
	st, e := a.importar(l.LoteUUID)
	if st != http.StatusConflict || len(e.FotosFaltantes) != 1 || e.FotosFaltantes[0].Offset != 0 {
		t.Fatalf("hash divergente: %d %+v", st, e)
	}
	if a.contar(`SELECT count(*) FROM propriedades`) != 0 {
		t.Fatal("aplicou com foto corrompida")
	}
	a.enviarFoto(f.UUID, dados, 0)
	if e := a.importarConfirmado(l.LoteUUID); e.Status != contratos.StatusImportado {
		t.Fatalf("reenvio correto: %+v", e)
	}
	a.conferirInvariantesFotos()

	// Tombstone da propriedade e da foto: some das visões vigentes, o registro continua.
	excl := contratos.FormatarInstante(time.Now().UnixMilli())
	p.Rev, p.ExcluidoEm = 2, &excl
	f.Rev, f.ExcluidoEm = 2, &excl
	l2 := a.novoLote([]contratos.Propriedade{p}, nil, []contratos.Foto{f})
	if est := a.putLote(l2); len(est.FotosFaltantes) != 0 {
		t.Fatal("tombstone de foto pediu bytes")
	}
	a.importarConfirmado(l2.LoteUUID)
	if a.contar(`SELECT count(*) FROM v_propriedades_vigentes`) != 0 || a.contar(`SELECT count(*) FROM propriedades`) != 1 || a.contar(`SELECT count(*) FROM v_fotos_vigentes`) != 0 {
		t.Fatal("tombstone mal aplicado")
	}
	// Reenviar a versão antiga não "ressuscita" (invariante 5).
	p.Rev, p.ExcluidoEm = 1, nil
	l3 := a.novoLote([]contratos.Propriedade{p}, nil, nil)
	a.putLote(l3)
	e3 := a.importarConfirmado(l3.LoteUUID)
	if e3.Recibo.Resultados[1].Resultado != "duplicado" || a.contar(`SELECT count(*) FROM v_propriedades_vigentes`) != 0 {
		t.Fatal("tombstone ressuscitou")
	}
}

func TestRegistroInvalidoNaoDerrubaOLote(t *testing.T) {
	a := novoAmb(t)
	a.parear()
	p := a.prop("Válida", 1)
	agora := p.CreatedAt
	itens := []contratos.Item{
		{UUID: uuid.NewString(), PropriedadeUUID: p.UUID, Categoria: "ave", CodigoCatalogo: "ave.emu", Quantidade: 1, CreatedAt: agora, UpdatedAt: agora, Rev: 1},
		{UUID: uuid.NewString(), PropriedadeUUID: p.UUID, Categoria: "movel", CodigoCatalogo: "movel.sofa", Quantidade: 2, CreatedAt: agora, UpdatedAt: agora, Rev: 1},
		{UUID: uuid.NewString(), PropriedadeUUID: p.UUID, Categoria: "movel", CodigoCatalogo: "movel.sofa", Quantidade: 1, CreatedAt: agora, UpdatedAt: agora, Rev: 1},
		{UUID: uuid.NewString(), PropriedadeUUID: uuid.NewString(), Categoria: "movel", CodigoCatalogo: "movel.mesa_jantar", Quantidade: 1, CreatedAt: agora, UpdatedAt: agora, Rev: 1},
	}
	l := a.novoLote([]contratos.Propriedade{p}, itens, nil)
	a.putLote(l)
	e := a.importarConfirmado(l.LoteUUID)
	res := e.Recibo.Resultados
	if res[1].Resultado != "aplicado" || res[2].Resultado != "rejeitado" || res[3].Resultado != "aplicado" || res[4].Resultado != "rejeitado" || res[5].Resultado != "rejeitado" {
		for _, x := range res {
			t.Logf("%s %s %s", x.Entidade, x.Resultado, x.Motivo)
		}
		t.Fatal("resultados inesperados")
	}
}

func TestAutenticacaoEAcessoAoLoteDeOutroAparelho(t *testing.T) {
	a := novoAmb(t)
	a.parear()
	l := a.novoLote([]contratos.Propriedade{a.prop("X", 1)}, nil, nil)
	a.putLote(l)
	cred := a.cred
	a.cred = "credencial-falsa-credencial-falsa-credencial"
	if st, _ := a.req("GET", "/v1/lotes/"+l.LoteUUID, nil, true); st != http.StatusUnauthorized {
		t.Fatalf("credencial falsa: %d", st)
	}
	a.cred = cred
	// Lote declarando outra instalação.
	l2 := l
	l2.LoteUUID, l2.InstalacaoUUID = uuid.NewString(), uuid.NewString()
	if st, _ := a.jsonReq("PUT", "/v1/lotes/"+l2.LoteUUID, l2, true); st != http.StatusForbidden {
		t.Fatalf("lote de outra instalação: %d", st)
	}
}

func TestReconciliador(t *testing.T) {
	a := novoAmb(t)
	a.parear()
	// Parte órfã (nenhum lote a espera) e arquivo publicado sem registro.
	os.MkdirAll(filepath.Join(a.pasta, "staging"), 0o700)
	os.WriteFile(filepath.Join(a.pasta, "staging", uuid.NewString()+".part"), []byte("x"), 0o600)
	orfa := uuid.NewString()
	os.MkdirAll(filepath.Join(a.pasta, "fotos", orfa[:2]), 0o700)
	os.WriteFile(filepath.Join(a.pasta, filepath.FromSlash(importacao.CaminhoRelativoFoto(orfa))), []byte("y"), 0o600)
	// Parte de lote em recepção: deve ficar.
	p := a.prop("R", 1)
	dados, h := fotoAleatoria(t, 10)
	f := a.foto(p.UUID, int64(len(dados)), h)
	a.putLote(a.novoLote([]contratos.Propriedade{p}, nil, []contratos.Foto{f}))
	a.req("PUT", "/v1/fotos/"+f.UUID+"?offset=0", dados[:5], true)

	rel, err := a.n.Importador.Reconciliar(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rel.PartesRemovidas != 1 || rel.OrfasRemovidas != 1 || len(rel.RegistrosSemArquivo) != 0 {
		t.Fatalf("relatório: %+v", rel)
	}
	if _, err := os.Stat(filepath.Join(a.pasta, "staging", f.UUID+".part")); err != nil {
		t.Fatal("removeu parte de lote em recepção")
	}
}

func TestConsultasAposImportar(t *testing.T) {
	a := novoAmb(t)
	a.parear()
	p1, p2 := a.prop("Sítio Área", 1), a.prop("Chácara Sem Área", 1)
	m, u := int64(1000), "alq_paulista" // 1 alqueire paulista = 2,42 ha
	p1.AreaValorMilesimos, p1.AreaUnidade = &m, &u
	lat, lon, orig := -10.5, -62.3, "manual"
	p1.Lat, p1.Lon, p1.GPSOrigem = &lat, &lon, &orig
	dados, h := fotoAleatoria(t, 1000)
	f := a.foto(p1.UUID, int64(len(dados)), h)
	l := a.novoLote([]contratos.Propriedade{p1, p2}, nil, []contratos.Foto{f})
	a.putLote(l)
	a.enviarFoto(f.UUID, dados, 0)
	a.importarConfirmado(l.LoteUUID)
	c := &consultas.Consultas{DB: a.n.B.Leitura}
	ctx := context.Background()
	r, err := c.Resumo(ctx, a.estudo.UUID)
	if err != nil {
		t.Fatal(err)
	}
	if r2, _ := c.Resumo(ctx, "outro-estudo"); r2.Propriedades != 0 {
		t.Fatal("resumo de outro estudo deveria ser vazio")
	}
	if r.Propriedades != 2 || r.Equipes != 2 || r.Bovinos != 24 || r.AreaHa != "2,42" || r.ComArea != 1 || r.SemArea != 1 || r.ComGPS != 1 || r.Fotos != 1 || r.LotesImportados != 1 {
		t.Fatalf("resumo: %+v", r)
	}
	lista, err := c.Propriedades(ctx, "área", a.estudo.UUID, "")
	if err != nil || len(lista) != 2 { // "Sítio Área" e "Chácara Sem Área"
		t.Fatalf("busca: %v %+v", err, lista)
	}
	if lista, _ := c.Propriedades(ctx, "SITIO", "", ""); len(lista) != 1 {
		t.Fatal("busca sem acento/maiúsculas falhou")
	}
	d, err := c.Propriedade(ctx, p1.UUID)
	if err != nil || d == nil || len(d.Fotos) != 1 || d.AreaHa == nil || *d.AreaHa != "2,42" || d.Estudo != "Fazenda Santa Maria" || d.JornadaNumero != 1 ||
		d.Equipe != "Equipe Alfa" || d.Observacoes == nil || d.Responsavel != "José da Silva" {
		t.Fatalf("detalhe: %v %+v", err, d)
	}
	if cam, _ := c.CaminhoFotoVigente(ctx, f.UUID); cam != importacao.CaminhoRelativoFoto(f.UUID) {
		t.Fatalf("caminho da foto: %q", cam)
	}
	lotes, _ := c.Lotes(ctx, 10)
	if len(lotes) != 1 || lotes[0].Aplicados != 4 || lotes[0].Duplicados != 0 || lotes[0].Equipe == nil || *lotes[0].Equipe != "Equipe Alfa" || lotes[0].Corrigido {
		t.Fatalf("lotes: %+v", lotes)
	}
}

// Código inexistente (erro de digitação): o operador corrige no Central escolhendo a
// equipe; o código recebido é preservado e a decisão fica registrada (cenário A28-3).
func TestCodigoInexistenteCorrigidoNaCentral(t *testing.T) {
	a := novoAmb(t)
	a.parear()
	errado := codigoInexistente(t)
	a.coleta = a.novaColeta(errado)
	p := a.prop("Sítio Digitado Errado", 1)
	l := a.novoLote([]contratos.Propriedade{p}, nil, nil)
	a.putLote(l)
	if st, e := a.importar(l.LoteUUID); st != 200 || e.Status != contratos.StatusAguardando {
		t.Fatalf("importar: %d %+v", st, e)
	}
	ls, _ := a.n.Importador.Aguardando(context.Background())
	if len(ls) != 1 || ls[0].Destino != nil || ls[0].Travado {
		t.Fatalf("código inexistente deveria vir sem destino: %+v", ls)
	}
	if _, err := a.decidir(l.LoteUUID, a.bravo.UUID); err != nil {
		t.Fatal(err)
	}
	e := a.getLote(l.LoteUUID)
	if e.Status != contratos.StatusImportado || !e.Recibo.Destino.Corrigido || e.Recibo.Destino.CodigoInformado != errado || e.Recibo.Destino.EquipeCodigo != a.bravo.Codigo {
		t.Fatalf("recibo: %+v", e.Recibo)
	}
	if a.contar(`SELECT count(*) FROM decisoes_destino WHERE acao = 'alterar' AND codigo_informado = ? AND codigo_confirmado = ? AND equipe_uuid = ?`, errado, a.bravo.Codigo, a.bravo.UUID) != 1 {
		t.Fatal("correção não registrada")
	}
	if a.contar(`SELECT count(*) FROM coletas WHERE uuid = ? AND codigo_informado = ? AND equipe_uuid = ?`, a.coleta.UUID, errado, a.bravo.UUID) != 1 ||
		a.contar(`SELECT count(*) FROM propriedades WHERE uuid = ? AND equipe_uuid = ?`, p.UUID, a.bravo.UUID) != 1 {
		t.Fatal("destino corrigido não aplicado")
	}
	// Os códigos oficiais das equipes não mudam.
	if a.contar(`SELECT count(*) FROM equipes WHERE codigo IN (?, ?)`, a.alfa.Codigo, a.bravo.Codigo) != 2 {
		t.Fatal("código oficial alterado")
	}
}

// Código válido de outra equipe: o operador altera o destino antes de importar.
func TestCodigoDeOutraEquipeAlterarDestino(t *testing.T) {
	a := novoAmb(t)
	a.parear()
	a.coleta = a.novaColeta(a.bravo.Codigo) // digitou o código da Bravo, mas é a Alfa
	l := a.novoLote([]contratos.Propriedade{a.prop("Sítio", 1)}, nil, nil)
	a.putLote(l)
	a.importar(l.LoteUUID)
	ls, _ := a.n.Importador.Aguardando(context.Background())
	if ls[0].Destino == nil || ls[0].Destino.EquipeUUID != a.bravo.UUID {
		t.Fatalf("destino sugerido: %+v", ls[0].Destino)
	}
	if _, err := a.decidir(l.LoteUUID, a.alfa.UUID); err != nil {
		t.Fatal(err)
	}
	if d := a.getLote(l.LoteUUID).Recibo.Destino; !d.Corrigido || d.EquipeCodigo != a.alfa.Codigo || d.CodigoInformado != a.bravo.Codigo {
		t.Fatalf("destino: %+v", d)
	}
}

// Cancelamento: nada importado nem apagado; o lote cancelado é terminal; o celular
// corrige o código e envia um lote novo da mesma coleta, reaproveitando a foto em staging.
func TestCancelamentoECorrecaoNoCelular(t *testing.T) {
	a := novoAmb(t)
	a.parear()
	a.coleta = a.novaColeta(codigoInexistente(t))
	p := a.prop("Sítio Cancelado", 1)
	dados, h := fotoAleatoria(t, 20_000)
	f := a.foto(p.UUID, int64(len(dados)), h)
	l := a.novoLote([]contratos.Propriedade{p}, nil, []contratos.Foto{f})
	a.putLote(l)
	a.enviarFoto(f.UUID, dados, 0)
	a.importar(l.LoteUUID)
	if _, err := a.decidir(l.LoteUUID, ""); err != nil {
		t.Fatal(err)
	}
	e := a.getLote(l.LoteUUID)
	if e.Status != contratos.StatusCancelado || e.Motivo == nil || *e.Motivo != contratos.MotivoCodigoInexistente || e.Recibo != nil {
		t.Fatalf("cancelado: %+v", e)
	}
	if a.contar(`SELECT count(*) FROM propriedades`) != 0 || a.contar(`SELECT count(*) FROM coletas WHERE equipe_uuid IS NOT NULL`) != 0 {
		t.Fatal("cancelamento importou dados")
	}
	if st, corpo := a.jsonReq("PUT", "/v1/lotes/"+l.LoteUUID, l, true); st != http.StatusConflict || !bytes.Contains(corpo, []byte("lote_cancelado")) {
		t.Fatalf("PUT em lote cancelado: %d %s", st, corpo)
	}
	if st, _ := a.req("POST", "/v1/lotes/"+l.LoteUUID+"/importar", nil, true); st != http.StatusConflict {
		t.Fatalf("importar lote cancelado: %d", st)
	}
	if _, err := a.decidir(l.LoteUUID, a.alfa.UUID); !errors.Is(err, importacao.ErrJaDecidido) {
		t.Fatalf("decidir lote cancelado: %v", err)
	}
	// O .part fica (reenvio da mesma coleta), mesmo depois do reconciliador.
	if _, err := a.n.Importador.Reconciliar(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(a.pasta, "staging", f.UUID+".part")); err != nil {
		t.Fatal("foto em staging apagada após cancelamento")
	}
	// Celular corrige o código (nova rev da coleta) e envia lote novo.
	a.coleta.CodigoEquipe, a.coleta.Rev = a.alfa.Codigo, 2
	l2 := a.novoLote([]contratos.Propriedade{p}, nil, []contratos.Foto{f})
	if est := a.putLote(l2); len(est.FotosFaltantes) != 0 {
		t.Fatalf("foto já recebida pedida de novo: %+v", est.FotosFaltantes)
	}
	e2 := a.importarConfirmado(l2.LoteUUID)
	if e2.Status != contratos.StatusImportado || e2.Recibo.Destino.Corrigido || e2.Recibo.Destino.EquipeCodigo != a.alfa.Codigo {
		t.Fatalf("reenvio: %+v", e2)
	}
	if a.contar(`SELECT count(*) FROM v_propriedades_vigentes`) != 1 || a.contar(`SELECT count(*) FROM decisoes_destino`) != 2 {
		t.Fatal("contagens após o reenvio")
	}
	a.conferirInvariantesFotos()
	if _, err := a.n.B.Escrita.Exec(`DELETE FROM decisoes_destino`); err == nil {
		t.Fatal("decisões deveriam ser somente acréscimo")
	}
}

// Destino travado: o 2º lote da mesma coleta só pode ir para a equipe já confirmada.
func TestDestinoTravado(t *testing.T) {
	a := novoAmb(t)
	a.parear()
	p := a.prop("Sítio Travado", 1)
	a.putLote(a.novoLote([]contratos.Propriedade{p}, nil, nil))
	l1 := a.novoLote([]contratos.Propriedade{p}, nil, nil)
	a.putLote(l1)
	a.importarConfirmado(l1.LoteUUID)

	p.Rev, p.BovAdultos = 2, 50
	a.coleta.CodigoEquipe, a.coleta.Rev = a.bravo.Codigo, 2 // mesmo que o celular mandasse outro código
	l2 := a.novoLote([]contratos.Propriedade{p}, nil, nil)
	a.putLote(l2)
	a.importar(l2.LoteUUID)
	ls, _ := a.n.Importador.Aguardando(context.Background())
	var x importacao.LoteAguardando
	for _, y := range ls {
		if y.LoteUUID == l2.LoteUUID {
			x = y
		}
	}
	if !x.Travado || x.DestinoTravado == nil || x.DestinoTravado.EquipeUUID != a.alfa.UUID || x.Atualizados != 1 {
		t.Fatalf("lote travado: %+v", x)
	}
	if _, err := a.decidir(l2.LoteUUID, a.bravo.UUID); !errors.Is(err, importacao.ErrDestinoTravado) {
		t.Fatalf("destino travado mudou: %v", err)
	}
	if _, err := a.decidir(l2.LoteUUID, a.alfa.UUID); err != nil {
		t.Fatal(err)
	}
	if a.contar(`SELECT count(*) FROM propriedades WHERE equipe_uuid = ? AND bov_adultos = 50`, a.alfa.UUID) != 1 {
		t.Fatal("2º lote não aplicado na equipe travada")
	}
	// Equipe com coleta não pode ser excluída.
	if err := a.n.Estudos.ExcluirEquipe(context.Background(), a.alfa.UUID); !errors.Is(err, estudos.ErrComColetas) {
		t.Fatalf("excluir equipe com coleta: %v", err)
	}
}

// O lote aguardando sobrevive ao reinício da Central (estado no banco, fotos em staging).
func TestLoteAguardandoSobreviveReinicio(t *testing.T) {
	a := novoAmb(t)
	a.parear()
	p := a.prop("Sítio Reinício", 1)
	dados, h := fotoAleatoria(t, 5000)
	f := a.foto(p.UUID, int64(len(dados)), h)
	l := a.novoLote([]contratos.Propriedade{p}, nil, []contratos.Foto{f})
	a.putLote(l)
	a.enviarFoto(f.UUID, dados, 0)
	a.importar(l.LoteUUID)
	a.n.Fechar()

	n, err := Abrir(context.Background(), a.pasta, []net.IP{net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer n.Fechar()
	ls, err := n.Importador.Aguardando(context.Background())
	if err != nil || len(ls) != 1 || ls[0].LoteUUID != l.LoteUUID {
		t.Fatalf("aguardando após reiniciar: %v %+v", err, ls)
	}
	if _, err := n.Importador.Decidir(context.Background(), l.LoteUUID, importacao.Decisao{EquipeUUID: a.alfa.UUID}); err != nil {
		t.Fatal(err)
	}
	var k int
	n.B.Leitura.QueryRow(`SELECT count(*) FROM v_fotos_vigentes`).Scan(&k)
	if k != 1 {
		t.Fatal("foto não publicada após reinício")
	}
}

// Uma coleta pertence a um aparelho: outro aparelho não envia lote dela.
func TestColetaDeOutroAparelho(t *testing.T) {
	a := novoAmb(t)
	a.parear()
	a.putLote(a.novoLote([]contratos.Propriedade{a.prop("A", 1)}, nil, nil))
	a.inst = uuid.NewString()
	a.parear()
	if st, _ := a.jsonReq("PUT", "/v1/lotes/x", a.novoLote(nil, nil, nil), true); st != http.StatusBadRequest {
		t.Fatalf("lote_uuid diferente do endereço: %d", st)
	}
	l := a.novoLote(nil, nil, nil)
	if st, _ := a.jsonReq("PUT", "/v1/lotes/"+l.LoteUUID, l, true); st != http.StatusForbidden {
		t.Fatalf("coleta de outro aparelho: %d", st)
	}
}
