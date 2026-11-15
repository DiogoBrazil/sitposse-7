package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http/httptest"

	"strings"
	"testing"
	"time"

	"sitposse7/desktop/internal/cartografia"
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
	ex(`INSERT INTO operacoes (uuid, nome, status, created_at, updated_at) VALUES ('op', 'Operação', 'ativa', 1, 1)`)
	ex(`INSERT INTO equipes (uuid, operacao_uuid, nome, responsavel, identificacao_area, created_at, updated_at, rev_origem, recebido_em, registro_json) VALUES ('eq', 'op', 'Bravo', 'Sgt', 'Linha C-25', 1, 1, 1, 1, '{}')`)
	prop := func(uuid, nome string, lat, lon any) {
		gps := any(nil)
		if lat != nil {
			gps = "gnss"
		}
		ex(`INSERT INTO propriedades (uuid, equipe_uuid, nome_propriedade, lat, lon, gps_origem, gps_precisao_m, created_at, updated_at, rev_origem, recebido_em, origem_instalacao, registro_json)
			VALUES (?, 'eq', ?, ?, ?, ?, 5, 1, 1, 1, 1, 'x', '{}')`, uuid, nome, lat, lon, gps)
	}
	prop("u1", "Sítio A", -10.40, -62.50)
	prop("u2", "Sítio B", -10.42, -62.47)
	prop("u3", "Sítio Sem GPS", nil, nil)
	return c
}

func servico(t *testing.T, c *Central, p cartografia.Provedor) *CartaService {
	return &CartaService{C: c, Provedor: p, cofre: &credenciais.Memoria{}, Agora: func() time.Time { return time.Date(2026, 11, 14, 22, 0, 0, 0, time.UTC) }}
}

func TestCartasDePontaAPonta(t *testing.T) {
	c := centralComPropriedades(t)
	pf := &provedorFalso{}
	s := servico(t, c, pf)

	plano, err := s.PlanejarGeral(estatisticas.Filtro{})
	if err != nil {
		t.Fatal(err)
	}
	if len(plano.Marcadores) != 2 || len(plano.SemLocalizacao) != 1 || plano.SemLocalizacao[0] != "Sítio Sem GPS" {
		t.Fatalf("plano = %+v", plano)
	}
	if plano.Marcadores[0].Numero != 1 || plano.Marcadores[1].Numero != 2 || len(plano.Grupos) != 0 {
		t.Errorf("numeração/grupos: %+v", plano)
	}
	if _, err := s.PlanejarIndividual("u3"); err == nil || !strings.Contains(err.Error(), "não tem coordenada") {
		t.Errorf("sem coordenada: %v", err)
	}

	d := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	tela := PedidoTela{Retangulo: plano.Retangulo, Data: &d}
	prev, err := s.Imagem(tela)
	if err != nil || prev.Guardada || pf.chamadas != 1 || prev.Periodo != "Aquisição de 28/10/2026" {
		t.Fatalf("prévia: %+v, %v, chamadas %d", prev, err, pf.chamadas)
	}
	// A mesma área e data não gastam cota de novo.
	prev2, _ := s.Imagem(tela)
	if !prev2.Guardada || pf.chamadas != 1 {
		t.Errorf("imagem guardada não reaproveitada (chamadas %d)", pf.chamadas)
	}

	reg, err := s.GerarCarta(PedidoCarta{PedidoTela: tela, Tipo: "geral", UUIDs: plano.UUIDs, Criterios: []string{"Equipe: Bravo"}})
	if err != nil {
		t.Fatal(err)
	}
	if pf.chamadas != 1 || reg.Propriedades != 2 || len(reg.Arquivos) != 2 {
		t.Errorf("carta: %+v (chamadas %d)", reg, pf.chamadas)
	}

	// Reinício do app: as cartas continuam lá e abrem sem provedor (sem internet).
	s2 := servico(t, c, nil)
	l := s2.Cartas()
	if len(l) != 1 || l[0].ID != reg.ID {
		t.Fatalf("cartas após reiniciar = %+v", l)
	}
	w := httptest.NewRecorder()
	HandlerCartografia(s2).ServeHTTP(w, httptest.NewRequest("GET", "/cartografia/cartas/"+reg.ID+"/carta.png", nil))
	if w.Code != 200 || w.Header().Get("Content-Type") != "image/png" {
		t.Errorf("rota da carta: %d", w.Code)
	}
	for _, u := range []string{"/cartografia/cartas/" + reg.ID + "/carta.json", "/cartografia/imagens/../../sitposse7.db", "/cartografia/cartas/x/carta.png"} {
		w := httptest.NewRecorder()
		HandlerCartografia(s2).ServeHTTP(w, httptest.NewRequest("GET", u, nil))
		if w.Code != 404 {
			t.Errorf("%s: %d", u, w.Code)
		}
	}
	// Sem credenciais e sem provedor: mensagem para configurar (não trava nada).
	s2.Provedor = nil
	outra := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	if _, err := s2.Imagem(PedidoTela{Retangulo: plano.Retangulo, Data: &outra}); !errors.Is(err, cartografia.ErrSemCredenciais) {
		t.Errorf("sem credenciais: %v", err)
	}
	// Limpar imagens mantém as cartas.
	if n, _ := s2.LimparImagens(); n != 1 || len(s2.Cartas()) != 1 {
		t.Errorf("limpar: %d imagens, %d cartas", n, len(s2.Cartas()))
	}
	if est := s2.Estado(); est.Configurado || est.Uso.Cartas != 1 {
		t.Errorf("estado = %+v", est)
	}
}

func TestCartaRecusaImagemIncompletaEPontoFora(t *testing.T) {
	c := centralComPropriedades(t)
	s := servico(t, c, &provedorFalso{transparencia: true})
	plano, _ := s.PlanejarGeral(estatisticas.Filtro{})
	d := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	_, err := s.GerarCarta(PedidoCarta{PedidoTela: PedidoTela{Retangulo: plano.Retangulo, Data: &d}, Tipo: "geral", UUIDs: plano.UUIDs})
	if !errors.Is(err, cartografia.ErrImagemIncompleta) || len(s.Cartas()) != 0 {
		t.Errorf("imagem incompleta: %v (cartas %d)", err, len(s.Cartas()))
	}
	s = servico(t, c, &provedorFalso{})
	r, _ := geo.Enquadrar([]geo.Ponto{{Lat: -10.40, Lon: -62.50}}, 100, 200, 2000, 1500) // só o Sítio A
	_, err = s.GerarCarta(PedidoCarta{PedidoTela: PedidoTela{Retangulo: r, Data: &d}, Tipo: "geral", UUIDs: plano.UUIDs})
	if err == nil || !strings.Contains(err.Error(), "fora da carta") {
		t.Errorf("ponto fora: %v", err)
	}
}

func TestPontosDistantesSaoAgrupados(t *testing.T) {
	c := centralComPropriedades(t)
	if _, err := c.n.B.Escrita.Exec(`INSERT INTO propriedades (uuid, equipe_uuid, nome_propriedade, lat, lon, gps_origem, created_at, updated_at, rev_origem, recebido_em, origem_instalacao, registro_json)
		VALUES ('u4', 'eq', 'Sítio Longe', -11.9, -61.0, 'gnss', 1, 1, 1, 1, 'x', '{}')`); err != nil {
		t.Fatal(err)
	}
	s := servico(t, c, &provedorFalso{})
	plano, err := s.PlanejarGeral(estatisticas.Filtro{})
	if err != nil {
		t.Fatal(err)
	}
	if plano.ResolucaoM <= 20 || len(plano.Grupos) != 2 {
		t.Errorf("resolução %.1f m/px, grupos %v", plano.ResolucaoM, plano.Grupos)
	}

}

// Prévia aberta no zoom máximo (área de dezenas de metros): o pedido é ampliado para 1,5 km.
func TestPedidoMinimoDeLargura(t *testing.T) {
	c := centralComPropriedades(t)
	s := servico(t, c, &provedorFalso{})
	x, y := geo.XY(geo.Ponto{Lat: -10.40, Lon: -62.50})
	d := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	ped, err := s.pedido(PedidoTela{Retangulo: geo.Retangulo{MinX: x - 40, MaxX: x + 40, MinY: y - 30, MaxY: y + 30}, Data: &d})
	if err != nil {
		t.Fatal(err)
	}
	// Tolerância de 0,5%: a escala usa a esfera do Web Mercator (R = 6.378.137 m) e a distância
	// usa o raio médio (haversine); a diferença (~0,1%) fica bem abaixo do 1% da RN-92.
	if w := geo.DistanciaM(geo.LatLon(ped.Ret.MinX, y), geo.LatLon(ped.Ret.MaxX, y)); w < 1492 {
		t.Errorf("largura do pedido = %.0f m; mínimo 1500", w)
	}
}
