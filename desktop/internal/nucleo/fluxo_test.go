package nucleo

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

	"sitposse7/desktop/internal/consultas"
	"sitposse7/desktop/internal/contratos"
	"sitposse7/desktop/internal/importacao"
	"sitposse7/desktop/internal/syncserver"
)

// ---------- ambiente de teste: Central real em HTTPS ----------

type amb struct {
	t     *testing.T
	n     *Nucleo
	base  string
	http  *http.Client
	cred  string
	inst  string
	eq    contratos.Equipe
	pasta string
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
	agora := contratos.FormatarInstante(time.Now().UnixMilli())
	a.eq = contratos.Equipe{UUID: uuid.NewString(), Nome: "Equipe Linha 25", Responsavel: "Sgt. Teste", CreatedAt: agora, UpdatedAt: agora, Rev: 1}
	return a
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

// parear cria a operação, gera o token e confirma o pedido como o operador faria.
func (a *amb) parear() {
	a.t.Helper()
	ctx := context.Background()
	if _, err := a.n.Pareamento.CriarOperacao(ctx, "Operação Teste", ""); err != nil {
		a.t.Fatal(err)
	}
	token, _, _, err := a.n.Pareamento.GerarToken(ctx)
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
	st, corpo := a.jsonReq("POST", "/v1/pareamento", contratos.PedidoPareamento{ProtocoloVersao: 1, SchemaVersao: 1, Token: token, InstalacaoUUID: a.inst, Equipe: a.eq}, false)
	if st != 200 {
		a.t.Fatalf("pareamento: %d %s", st, corpo)
	}
	var resp contratos.RespostaPareamento
	json.Unmarshal(corpo, &resp)
	a.cred = resp.CredencialDispositivo
	// Token é de uso único.
	st, _ = a.jsonReq("POST", "/v1/pareamento", contratos.PedidoPareamento{ProtocoloVersao: 1, SchemaVersao: 1, Token: token, InstalacaoUUID: uuid.NewString(), Equipe: a.eq}, false)
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
	return contratos.Lote{ProtocoloVersao: 1, SchemaVersao: 1, LoteUUID: uuid.NewString(), InstalacaoUUID: a.inst,
		CriadoEm: contratos.FormatarInstante(time.Now().UnixMilli()), Equipe: a.eq, Propriedades: props, Itens: itens, Fotos: fotos}
}

func (a *amb) prop(nome string, rev int64) contratos.Propriedade {
	agora := contratos.FormatarInstante(time.Now().UnixMilli())
	return contratos.Propriedade{UUID: uuid.NewString(), EquipeUUID: a.eq.UUID, NomePropriedade: nome, BovAdultos: 12, CreatedAt: agora, UpdatedAt: agora, Rev: rev}
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

	st, e := a.importar(l.LoteUUID)
	if st != 200 || e.Status != "importado" || e.Recibo == nil {
		t.Fatalf("importação: %d %+v", st, e)
	}
	r := e.Recibo
	if r.ReciboSHA256 != r.CalcularHash() || len(r.Resultados) != 1+2+2+2 {
		t.Fatalf("recibo: %+v", r)
	}
	if r.Resultados[0].Entidade != "equipe" || r.Resultados[0].Resultado != "duplicado" {
		t.Errorf("equipe gravada no pareamento com a mesma rev deveria vir como duplicado: %+v", r.Resultados[0])
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
	a.conferirInvariantesFotos()

	// Cenário 5 (ACK perdido) e 4 (mesmo lote 3×): o mesmo recibo, nada reaplicado.
	aud := a.contar(`SELECT count(*) FROM auditoria`)
	for i := 0; i < 3; i++ {
		if est := a.putLote(l); est.Status != "importado" || est.Recibo.ReciboSHA256 != r.ReciboSHA256 {
			t.Fatalf("PUT repetido %d: %+v", i, est)
		}
		if st, e := a.importar(l.LoteUUID); st != 200 || e.Recibo.ReciboSHA256 != r.ReciboSHA256 {
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
	_, e3 := a.importar(l3.LoteUUID)
	got := map[string]string{}
	for _, x := range e3.Recibo.Resultados {
		got[x.UUID] = x.Resultado
	}
	if got[p1.UUID] != "aplicado" || got[p2.UUID] != "duplicado" || got[f1.UUID] != "duplicado" || got[a.eq.UUID] != "duplicado" {
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
	if st, _ := a.importar(l.LoteUUID); st != 200 {
		t.Fatalf("reenvio correto: %d", st)
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
	if st, _ := a.importar(l2.LoteUUID); st != 200 {
		t.Fatal(st)
	}
	if a.contar(`SELECT count(*) FROM v_propriedades_vigentes`) != 0 || a.contar(`SELECT count(*) FROM propriedades`) != 1 || a.contar(`SELECT count(*) FROM v_fotos_vigentes`) != 0 {
		t.Fatal("tombstone mal aplicado")
	}
	// Reenviar a versão antiga não "ressuscita" (invariante 5).
	p.Rev, p.ExcluidoEm = 1, nil
	l3 := a.novoLote([]contratos.Propriedade{p}, nil, nil)
	a.putLote(l3)
	_, e3 := a.importar(l3.LoteUUID)
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
	st, e := a.importar(l.LoteUUID)
	if st != 200 {
		t.Fatal(st)
	}
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
	if st, _ := a.importar(l.LoteUUID); st != 200 {
		t.Fatal(st)
	}
	c := &consultas.Consultas{DB: a.n.B.Leitura}
	ctx := context.Background()
	r, err := c.Resumo(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if r.Propriedades != 2 || r.Bovinos != 24 || r.AreaHa != "2,42" || r.ComArea != 1 || r.SemArea != 1 || r.ComGPS != 1 || r.Fotos != 1 || r.LotesImportados != 1 {
		t.Fatalf("resumo: %+v", r)
	}
	lista, err := c.Propriedades(ctx, "área", "")
	if err != nil || len(lista) != 2 { // "Sítio Área" e "Chácara Sem Área"
		t.Fatalf("busca: %v %+v", err, lista)
	}
	if lista, _ := c.Propriedades(ctx, "SITIO", ""); len(lista) != 1 {
		t.Fatal("busca sem acento/maiúsculas falhou")
	}
	d, err := c.Propriedade(ctx, p1.UUID)
	if err != nil || d == nil || len(d.Fotos) != 1 || d.AreaHa == nil || *d.AreaHa != "2,42" {
		t.Fatalf("detalhe: %v %+v", err, d)
	}
	if cam, _ := c.CaminhoFotoVigente(ctx, f.UUID); cam != importacao.CaminhoRelativoFoto(f.UUID) {
		t.Fatalf("caminho da foto: %q", cam)
	}
	lotes, _ := c.Lotes(ctx, 10)
	if len(lotes) != 1 || lotes[0].Aplicados != 3 || lotes[0].Duplicados != 1 {
		t.Fatalf("lotes: %+v", lotes)
	}
}
