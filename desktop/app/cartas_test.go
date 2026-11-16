package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"sitposse7/desktop/internal/cartografia"
	"sitposse7/desktop/internal/cartografia/cog/cogteste"
	"sitposse7/desktop/internal/cartografia/geo"
	"sitposse7/desktop/internal/credenciais"
	"sitposse7/desktop/internal/estatisticas"
)

type provedorFalso struct {
	chamadas      int
	transparencia bool
}

func (p *provedorFalso) Nome() string { return "falso" }
func (p *provedorFalso) Atribuicao(ano int) string {
	return fmt.Sprintf("Contains modified Copernicus Sentinel data %d", ano)
}
func (p *provedorFalso) Passagens(ctx context.Context, r geo.Retangulo, de, ate time.Time) ([]cartografia.Passagem, error) {
	return []cartografia.Passagem{{Data: time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC), Nuvens: 3}}, nil
}
func (p *provedorFalso) Imagem(ctx context.Context, ped cartografia.PedidoImagem) (*cartografia.Imagem, error) {
	p.chamadas++
	img := image.NewNRGBA(image.Rect(0, 0, ped.Largura, ped.Altura))
	semDados := 0.0
	for y := 0; y < ped.Altura; y++ {
		for x := 0; x < ped.Largura; x++ {
			a := uint8(255)
			if p.transparencia && x < ped.Largura/4 {
				a = 0
			}
			img.SetNRGBA(x, y, color.NRGBA{40, 100, 40, a})
		}
	}
	if p.transparencia {
		semDados = 0.25
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return &cartografia.Imagem{PNG: b.Bytes(), Ret: ped.Ret, Largura: ped.Largura, Altura: ped.Altura, Data: ped.Data, De: ped.De, Ate: ped.Ate, SemDados: semDados, ResolucaoNativaM: 10}, nil
}

func centralComPropriedades(t *testing.T) *Central {
	t.Helper()
	c := &Central{PastaDados: t.TempDir()}
	if err := c.Iniciar(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Encerrar)
	db := c.n.B.Escrita
	ex := func(q string, a ...any) {
		if _, err := db.Exec(q, a...); err != nil {
			t.Fatal(err)
		}
	}
	e, err := c.n.Estudos.CriarEstudo(context.Background(), "Levantamento Linha C-25", "2026-10-10", []string{"Bravo"})
	if err != nil {
		t.Fatal(err)
	}
	eq := e.ListaJornadas[0].Equipes[0]
	ex(`INSERT INTO dispositivos (instalacao_uuid, credencial_hash, pareado_em) VALUES ('i', 'h', 1)`)
	ex(`INSERT INTO coletas (uuid, instalacao_uuid, codigo_informado, codigo_original, equipe_uuid, destino_confirmado_em, created_at, updated_at, recebido_em)
		VALUES ('col', 'i', ?, ?, ?, 1, 1, 1, 1)`, eq.Codigo, eq.Codigo, eq.UUID)
	prop := func(uuid, nome string, lat, lon any) {
		gps := any(nil)
		if lat != nil {
			gps = "gnss"
		}
		ex(`INSERT INTO propriedades (uuid, coleta_uuid, equipe_uuid, nome_propriedade, responsavel_familia, lat, lon, gps_origem, gps_precisao_m, created_at, updated_at, rev_origem, recebido_em, origem_instalacao, registro_json)
			VALUES (?, 'col', ?, ?, 'Fulano', ?, ?, ?, 5, 1, 1, 1, 1, 'x', '{}')`, uuid, eq.UUID, nome, lat, lon, gps)
	}
	prop("u1", "Sítio A", -10.40, -62.50)
	prop("u2", "Sítio B", -10.42, -62.47)
	prop("u3", "Sítio Sem GPS", nil, nil)
	return c
}

// catalogoFalso: um mosaico sintético (EPSG:4326, ≈ 110 m) servido por HTTP com Range.
type catalogoFalso struct {
	url string
	err error
}

func (c *catalogoFalso) Candidatas(context.Context, [4]float64, time.Time) ([]cartografia.Candidata, error) {
	if c.err != nil {
		return nil, c.err
	}
	ate := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	return []cartografia.Candidata{{ID: "MOS", Fonte: cartografia.FonteMosaico, Data: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), Ate: &ate, ResolucaoM: 10, URLRGB: c.url}}, nil
}

func servidorMosaico(t *testing.T) string {
	t.Helper()
	im := cogteste.Imagem{L: 1100, A: 1000, Amostras: 3, Pix: make([]byte, 1100*1000*3)}
	for y := 0; y < im.A; y++ {
		for x := 0; x < im.L; x++ {
			k := 3 * (y*im.L + x)
			im.Pix[k], im.Pix[k+1], im.Pix[k+2] = byte(60+x%80), byte(90+y%60), byte(40+(x+y)%40)
		}
	}
	zero := 0
	arq := cogteste.Escrever([]cogteste.Imagem{im, cogteste.Reduzir(im)}, cogteste.Opcoes{Big: true, EPSG: 4326, OrigemX: -63, OrigemY: -10, ResX: 0.001, ResY: 0.001, NoData: &zero, Bloco: 256})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"1"`)
		http.ServeContent(w, r, "m.tif", time.Unix(0, 0), bytes.NewReader(arq))
	}))
	t.Cleanup(srv.Close)
	return srv.URL + "/mosaico.tif"
}

func servico(t *testing.T, c *Central, p cartografia.Provedor, cat *catalogoFalso) *CartaService {
	return &CartaService{C: c, Provedor: p, Catalogo: cat, cofre: &credenciais.Memoria{}, Agora: func() time.Time { return time.Date(2026, 11, 15, 22, 0, 0, 0, time.UTC) }}
}

// esperar a tarefa de preparo terminar.
func esperar(t *testing.T, s *CartaService, id string) *Andamento {
	t.Helper()
	limite := time.Now().Add(60 * time.Second)
	for time.Now().Before(limite) {
		a, err := s.Andamento(id)
		if err != nil {
			t.Fatal(err)
		}
		if a.Concluida {
			return a
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("preparo não terminou")
	return nil
}

func TestCartasDePontaAPonta(t *testing.T) {
	c := centralComPropriedades(t)
	cat := &catalogoFalso{url: servidorMosaico(t)}
	s := servico(t, c, nil, cat)

	plano, err := s.PlanejarGeral(estatisticas.Filtro{})
	if err != nil {
		t.Fatal(err)
	}
	if len(plano.Marcadores) != 2 || len(plano.SemLocalizacao) != 1 || plano.SemLocalizacao[0] != "Sítio Sem GPS" {
		t.Fatalf("plano = %+v", plano)
	}
	if plano.Marcadores[0].Numero == plano.Marcadores[1].Numero || plano.Marcadores[0].Responsavel != "Fulano" || plano.Marcadores[0].UUID == "" {
		t.Errorf("marcadores: %+v", plano.Marcadores)
	}
	if _, err := s.PlanejarIndividual("u3"); err == nil || !strings.Contains(err.Error(), "não tem coordenada") {
		t.Errorf("sem coordenada: %v", err)
	}

	id, err := s.Preparar(PedidoPreparo{Tipo: "geral", UUIDs: plano.UUIDs})
	if err != nil {
		t.Fatal(err)
	}
	a := esperar(t, s, id)
	if a.Erro != "" || a.Resultado == nil {
		t.Fatalf("preparo: %+v", a)
	}
	res := a.Resultado
	if res.Guardada || len(res.Partes) == 0 || res.Partes[0].Rotulo != "Mosaico Sentinel-2 trimestral (INPE)" || !strings.Contains(res.Tiles, res.ComposicaoID) {
		t.Fatalf("resultado: %+v", res)
	}

	reg, err := s.GerarCarta(PedidoCarta{Tipo: "geral", UUIDs: plano.UUIDs, ComposicaoID: res.ComposicaoID, Criterios: []string{"Equipe: Bravo"}})
	if err != nil {
		t.Fatal(err)
	}
	if reg.Propriedades != 2 || strings.Join(reg.Arquivos, ",") != "carta-01-geral.png,carta.pdf" || !strings.Contains(reg.Periodo, "01/04/2026") {
		t.Errorf("carta: %+v", reg)
	}

	// Reinício do app sem internet: as cartas continuam lá; a mesma área reabre a composição.
	cat.err = cartografia.ErrSemConexao
	s2 := servico(t, c, nil, cat)
	l := s2.Cartas()
	if len(l) != 1 || l[0].ID != reg.ID {
		t.Fatalf("cartas após reiniciar = %+v", l)
	}
	id2, _ := s2.Preparar(PedidoPreparo{Tipo: "geral", UUIDs: plano.UUIDs, Atualizar: true})
	if a2 := esperar(t, s2, id2); a2.Resultado == nil || !a2.Resultado.Guardada || a2.Resultado.ComposicaoID != res.ComposicaoID {
		t.Fatalf("offline: %+v", a2)
	}
	get := func(u string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		HandlerCartografia(s2).ServeHTTP(w, httptest.NewRequest("GET", u, nil))
		return w
	}
	if w := get("/cartografia/cartas/" + reg.ID + "/carta-01-geral.png"); w.Code != 200 || w.Header().Get("Content-Type") != "image/png" {
		t.Errorf("rota da carta: %d", w.Code)
	}
	cx, cy := geo.XY(plano.Marcadores[0].Ponto)
	z := res.ZGeral
	n := float64(int(1) << z)
	tx, ty := int((cx+20037508.342789244)/40075016.68557849*n), int((20037508.342789244-cy)/40075016.68557849*n)
	if w := get(fmt.Sprintf("/cartografia/composicoes/%s/%d/%d/%d.jpg", res.ComposicaoID, z, tx, ty)); w.Code != 200 || w.Header().Get("Content-Type") != "image/jpeg" {
		t.Errorf("tile: %d %s", w.Code, w.Header().Get("Content-Type"))
	}
	if w := get(fmt.Sprintf("/cartografia/composicoes/%s/3/1/1.jpg", res.ComposicaoID)); w.Header().Get("Content-Type") != "image/png" {
		t.Errorf("tile fora da composição deveria ser transparente")
	}
	for _, u := range []string{"/cartografia/cartas/" + reg.ID + "/carta.json", "/cartografia/imagens/../../sitposse7.db", "/cartografia/cartas/x/carta.png",
		"/cartografia/composicoes/../../sitposse7.db", "/cartografia/composicoes/" + res.ComposicaoID + "/composicao.json"} {
		if w := get(u); w.Code != 404 {
			t.Errorf("%s: %d", u, w.Code)
		}
	}
	// Limpar imagens mantém as cartas e apaga composições e blocos.
	if _, err := s2.LimparImagens(); err != nil || len(s2.Cartas()) != 1 {
		t.Errorf("limpar: %v, %d cartas", err, len(s2.Cartas()))
	}
	if est := s2.Estado(); est.Configurado || est.Uso.Cartas != 1 || est.Uso.Imagens != 0 {
		t.Errorf("estado = %+v", est)
	}
	// Sem composição: a carta pede para preparar a imagem.
	if _, err := s2.GerarCarta(PedidoCarta{Tipo: "geral", UUIDs: plano.UUIDs, ComposicaoID: res.ComposicaoID}); err == nil {
		t.Error("carta sem composição")
	}
}

func TestCartaComSetores(t *testing.T) {
	c := centralComPropriedades(t)
	// Um grupo a cerca de 60 km dos dois primeiros: carta geral + 2 setores.
	if _, err := c.n.B.Escrita.Exec(`INSERT INTO propriedades (uuid, coleta_uuid, equipe_uuid, nome_propriedade, responsavel_familia, lat, lon, gps_origem, created_at, updated_at, rev_origem, recebido_em, origem_instalacao, registro_json)
		SELECT 'u4', 'col', equipe_uuid, 'Sítio Longe', 'Beltrano', -10.80, -62.05, 'gnss', 1, 1, 1, 1, 'x', '{}' FROM coletas WHERE uuid = 'col'`); err != nil {
		t.Fatal(err)
	}
	s := servico(t, c, nil, &catalogoFalso{url: servidorMosaico(t)})
	plano, err := s.PlanejarGeral(estatisticas.Filtro{})
	if err != nil {
		t.Fatal(err)
	}
	id, _ := s.Preparar(PedidoPreparo{Tipo: "geral", UUIDs: plano.UUIDs})
	a := esperar(t, s, id)
	if a.Resultado == nil || len(a.Resultado.Setores) != 2 {
		t.Fatalf("setores: %+v", a)
	}
	reg, err := s.GerarCarta(PedidoCarta{Tipo: "geral", UUIDs: plano.UUIDs, ComposicaoID: a.Resultado.ComposicaoID})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(reg.Arquivos, ",") != "carta-01-geral.png,carta-02-setor-A.png,carta-03-setor-B.png,carta.pdf" {
		t.Fatalf("arquivos: %v", reg.Arquivos)
	}
}

func TestReservaCDSEQuandoOINPEFalha(t *testing.T) {
	c := centralComPropriedades(t)
	pf := &provedorFalso{}
	s := servico(t, c, pf, &catalogoFalso{err: fmt.Errorf("%w: INPE fora do ar", cartografia.ErrSemConexao)})
	plano, _ := s.PlanejarGeral(estatisticas.Filtro{})
	id, _ := s.Preparar(PedidoPreparo{Tipo: "geral", UUIDs: plano.UUIDs})
	a := esperar(t, s, id)
	if a.Resultado == nil || !a.Resultado.Reserva || pf.chamadas == 0 {
		t.Fatalf("reserva: %+v (chamadas %d)", a, pf.chamadas)
	}
	if _, err := s.GerarCarta(PedidoCarta{Tipo: "geral", UUIDs: plano.UUIDs, ComposicaoID: a.Resultado.ComposicaoID}); err != nil {
		t.Fatal(err)
	}
	// Reserva com imagem incompleta: nenhuma carta (RN-93).
	s3 := servico(t, centralComPropriedades(t), &provedorFalso{transparencia: true}, &catalogoFalso{err: cartografia.ErrSemConexao})
	id3, _ := s3.Preparar(PedidoPreparo{Tipo: "geral", UUIDs: plano.UUIDs})
	if a3 := esperar(t, s3, id3); a3.Resultado != nil || !strings.Contains(a3.Erro, "não cobre toda a área") {
		t.Fatalf("reserva incompleta: %+v", a3)
	}
	// Sem INPE e sem credenciais: mensagem de conexão (RN-91).
	s4 := servico(t, centralComPropriedades(t), nil, &catalogoFalso{err: cartografia.ErrSemConexao})
	s4.Provedor = nil
	id4, _ := s4.Preparar(PedidoPreparo{Tipo: "geral", UUIDs: plano.UUIDs})
	if a4 := esperar(t, s4, id4); !a4.SemConexao || a4.Erro != cartografia.ErrSemConexao.Error() {
		t.Fatalf("sem nada: %+v", a4)
	}
}

func TestCancelarPreparo(t *testing.T) {
	c := centralComPropriedades(t)
	s := servico(t, c, nil, &catalogoFalso{url: servidorMosaico(t)})
	plano, _ := s.PlanejarGeral(estatisticas.Filtro{})
	id, _ := s.Preparar(PedidoPreparo{Tipo: "geral", UUIDs: plano.UUIDs})
	s.Cancelar(id)
	a := esperar(t, s, id)
	if a.Resultado != nil && a.Erro == "" {
		t.Skip("o preparo terminou antes do cancelamento") // máquina rápida: nada a conferir
	}
	if a.Erro != "Operação cancelada." {
		t.Fatalf("cancelado: %+v", a)
	}
}

// RN-98: coordenada alterada depois do preparo — a carta é recusada até atualizar as
// referências; sem internet, vale a imagem anterior se ainda cobrir todos os pinos.
func TestReferenciasAtualizadas(t *testing.T) {
	c := centralComPropriedades(t)
	cat := &catalogoFalso{url: servidorMosaico(t)}
	s := servico(t, c, nil, cat)
	mover := func(uuid string, lat float64) {
		if _, err := c.n.B.Escrita.Exec(`UPDATE propriedades SET lat = ?, gps_origem = 'manual', gps_precisao_m = NULL WHERE uuid = ?`, lat, uuid); err != nil {
			t.Fatal(err)
		}
	}
	preparar := func(uuids []string) *Andamento {
		id, err := s.Preparar(PedidoPreparo{Tipo: "geral", UUIDs: uuids})
		if err != nil {
			t.Fatal(err)
		}
		return esperar(t, s, id)
	}
	plano, err := s.PlanejarGeral(estatisticas.Filtro{})
	if err != nil {
		t.Fatal(err)
	}
	a := preparar(plano.UUIDs)
	if a.Resultado == nil || a.Resultado.Anterior {
		t.Fatalf("preparo: %+v", a)
	}
	antiga := a.Resultado.ComposicaoID

	// Só o nome mudou: a imagem continua valendo (a chave depende só das coordenadas).
	if _, err := c.n.B.Escrita.Exec(`UPDATE propriedades SET nome_propriedade = 'Sítio A Renomeado' WHERE uuid = 'u1'`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GerarCarta(PedidoCarta{Tipo: "geral", UUIDs: plano.UUIDs, ComposicaoID: antiga}); err != nil {
		t.Fatalf("mudança só de nome: %v", err)
	}

	// Coordenada alterada (≈ 550 m, ainda dentro da geral): recusada sem o aceite.
	mover("u1", -10.405)
	if _, err := s.GerarCarta(PedidoCarta{Tipo: "geral", UUIDs: plano.UUIDs, ComposicaoID: antiga}); !errors.Is(err, ErrReferenciasDesatualizadas) {
		t.Fatalf("coordenada alterada: %v", err)
	}

	// Sem internet: a imagem anterior serve (cobre os pinos), sinalizada.
	cat.err = cartografia.ErrSemConexao
	a = preparar(plano.UUIDs)
	if a.Erro != "" || a.Resultado == nil || !a.Resultado.Anterior || a.Resultado.ComposicaoID != antiga || !a.Resultado.Guardada {
		t.Fatalf("anterior sem internet: %+v", a)
	}
	if _, err := s.GerarCarta(PedidoCarta{Tipo: "geral", UUIDs: plano.UUIDs, ComposicaoID: antiga, EnquadramentoAnterior: true}); err != nil {
		t.Fatalf("carta com a imagem anterior aceita: %v", err)
	}

	// Pino fora da folha anterior: sem internet não há imagem (RN-91), e a carta é recusada
	// mesmo com o aceite.
	mover("u1", -10.95)
	a = preparar(plano.UUIDs)
	if a.Resultado != nil || !a.SemConexao {
		t.Fatalf("fora da folha anterior: %+v", a)
	}
	if _, err := s.GerarCarta(PedidoCarta{Tipo: "geral", UUIDs: plano.UUIDs, ComposicaoID: antiga, EnquadramentoAnterior: true}); !errors.Is(err, ErrReferenciasDesatualizadas) {
		t.Fatalf("aceite com pino fora: %v", err)
	}

	// Com internet: "Atualizar referências" prepara a área nova e a carta sai sem aceite.
	cat.err = nil
	mover("u1", -10.405)
	a = preparar(plano.UUIDs)
	if a.Resultado == nil || a.Resultado.Anterior || a.Resultado.ComposicaoID == antiga {
		t.Fatalf("referências atualizadas: %+v", a)
	}
	if _, err := s.GerarCarta(PedidoCarta{Tipo: "geral", UUIDs: plano.UUIDs, ComposicaoID: a.Resultado.ComposicaoID}); err != nil {
		t.Fatalf("carta com a imagem nova: %v", err)
	}
}
