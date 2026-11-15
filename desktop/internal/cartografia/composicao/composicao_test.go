package composicao

import (
	"bytes"
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"sitposse7/desktop/internal/cartografia"
	"sitposse7/desktop/internal/cartografia/cog/cogteste"
	"sitposse7/desktop/internal/cartografia/geo"
)

// terreno: cor "real" de um lugar (padrão suave + talhões), a mesma para todas as fontes.
func terreno(p geo.Ponto) [3]float64 {
	u, v := p.Lon*900, p.Lat*900
	t := 0.5 + 0.5*math.Sin(u)*math.Cos(v)
	talhao := math.Mod(math.Floor(u/3)+math.Floor(v/3), 2)
	return [3]float64{60 + 80*t + 30*talhao, 90 + 60*t, 40 + 40*t}
}

// cena gera um COG cujos pixels são f(terreno) no SRC dado.
func cena(epsg int, ox, oy, res float64, l, a int, f func(c [3]float64, p geo.Ponto) []byte, amostras int) []byte {
	im := cogteste.Imagem{L: l, A: a, Amostras: amostras, Pix: make([]byte, l*a*amostras)}
	for y := 0; y < a; y++ {
		for x := 0; x < l; x++ {
			p, _ := geo.DeSRC(epsg, ox+(float64(x)+0.5)*res, oy-(float64(y)+0.5)*res)
			copy(im.Pix[(y*l+x)*amostras:], f(terreno(p), p))
		}
	}
	zero := 0
	return cogteste.Escrever([]cogteste.Imagem{im, cogteste.Reduzir(im), cogteste.Reduzir(cogteste.Reduzir(im))},
		cogteste.Opcoes{Big: true, EPSG: epsg, OrigemX: ox, OrigemY: oy, ResX: res, ResY: res, NoData: &zero, Bloco: 128})
}

type catalogoFalso struct {
	cands []cartografia.Candidata
	err   error
}

func (c *catalogoFalso) Candidatas(context.Context, [4]float64, time.Time) ([]cartografia.Candidata, error) {
	return c.cands, c.err
}

func TestPrepararCompoeGuardaEReabreOffline(t *testing.T) {
	centro := geo.Ponto{Lat: -10.075, Lon: -63.169}
	z, _ := geo.ZonaEPSG(32720)
	le, no := geo.UTM(centro, z)
	b8 := func(v float64) byte { return byte(math.Max(1, math.Min(255, v))) }
	arquivos := map[string][]byte{
		// CBERS: 4 × 4 km a 4 m, arroxeado e com névoa (precisa de harmonização).
		"/cbers.tif": cena(32720, le-2000, no+2000, 4, 1000, 1000, func(c [3]float64, _ geo.Ponto) []byte {
			return []byte{b8(0.6*c[0] + 70), b8(0.5*c[1] + 60), b8(0.7*c[2] + 90)}
		}, 3),
		// Mosaico de referência (4326, ≈ 11 m), cobre tudo.
		"/mosaico.tif": cena(4326, -63.25, -10.0, 0.0001, 1600, 1600, func(c [3]float64, _ geo.Ponto) []byte {
			return []byte{b8(c[0]), b8(c[1]), b8(c[2])}
		}, 3),
		// Sentinel-2 (UTM, 10 m) e SCL (20 m) com nuvem a leste.
		"/tci.tif": cena(32720, le-4000, no+4000, 10, 800, 800, func(c [3]float64, _ geo.Ponto) []byte {
			return []byte{b8(c[0]), b8(c[1]), b8(c[2])}
		}, 3),
		"/scl.tif": cena(32720, le-4000, no+4000, 20, 400, 400, func(_ [3]float64, p geo.Ponto) []byte {
			if p.Lon > centro.Lon+0.01 {
				return []byte{9}
			}
			return []byte{4}
		}, 1),
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, ok := arquivos[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("ETag", `"1"`)
		http.ServeContent(w, r, "x.tif", time.Unix(0, 0), bytes.NewReader(b))
	}))
	agora := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	ate := agora.AddDate(0, -5, 0)
	cands := []cartografia.Candidata{
		{ID: "S2", Fonte: cartografia.FonteS2, Data: agora.AddDate(0, 0, -20), ResolucaoM: 10, URLRGB: srv.URL + "/tci.tif", URLSCL: srv.URL + "/scl.tif"},
		{ID: "MOS", Fonte: cartografia.FonteMosaico, Data: agora.AddDate(0, -8, 0), Ate: &ate, ResolucaoM: 10, URLRGB: srv.URL + "/mosaico.tif"},
		{ID: "CB", Fonte: cartografia.FonteCBERS, Data: agora.AddDate(0, -1, 0), ResolucaoM: 2, URLRGB: srv.URL + "/cbers.tif"},
	}
	dir := t.TempDir()
	cat := &catalogoFalso{cands: cands}
	s := &Servico{Dir: dir, Catalogo: cat, Agora: func() time.Time { return agora }}
	pts := []geo.Ponto{centro, {Lat: -10.078, Lon: -63.166}, {Lat: -10.072, Lon: -63.171}}
	cfg := geo.ConfigPadrao
	var fases []string
	comp, err := s.Preparar(context.Background(), pts, cfg, func(p Progresso) {
		if len(fases) == 0 || fases[len(fases)-1] != p.Fase {
			fases = append(fases, p.Fase)
		}
		if p.Total > 0 && p.Feito > p.Total {
			t.Errorf("progresso %d/%d", p.Feito, p.Total)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(fases, "|") != "Buscando imagens disponíveis|Avaliando a qualidade das imagens|Preparando a imagem da região" {
		t.Fatalf("fases: %v", fases)
	}
	cb := comp.Receita.Cenas[0]
	if cb.Candidata.ID != "CB" || cb.Cor.Tipo == "identidade" || !cb.NuvemEstimada {
		t.Fatalf("cena principal: %s %s estimada=%v", cb.Candidata.ID, cb.Cor.Tipo, cb.NuvemEstimada)
	}
	if cb.Cor.Tipo == "afim" && cb.Cor.R2 < 0.8 {
		t.Fatalf("harmonização fraca: R²=%.2f", cb.Cor.R2)
	}
	if comp.ResolucaoM != 2 || comp.Partes[0].Fracao <= 0 {
		t.Fatalf("partes %+v", comp.Partes)
	}
	// A folha geral foi gerada, cobre tudo e o CBERS harmonizado se parece com o terreno.
	img, cob, err := s.Folha(context.Background(), comp.ID, 0)
	if err != nil || cob < 0.999 {
		t.Fatalf("folha: cobertura %.3f %v", cob, err)
	}
	x, y := geo.Pixel(centro, comp.Folhas.Geral.Retangulo, LarguraFolha, AlturaFolha)
	k := 4 * (int(y)*LarguraFolha + int(x))
	esperado := terreno(centro)
	if math.Abs(float64(img.Pix[k+1])-esperado[1]) > 45 {
		t.Fatalf("cor no centro %v, terreno %v", img.Pix[k:k+3], esperado)
	}
	// Tiles pré-gerados: existe ao menos um no zoom nativo sobre o centro.
	cx, cy := geo.XY(centro)
	tz := comp.ZNativo
	nt := float64(int(1) << tz)
	tx := int((cx + math.Pi*geo.Raio) / (2 * math.Pi * geo.Raio) * nt)
	ty := int((math.Pi*geo.Raio - cy) / (2 * math.Pi * geo.Raio) * nt)
	if _, err := os.Stat(filepath.Join(dir, "composicoes", comp.ID, "tiles", itoa(tz), itoa(tx), itoa(ty)+".jpg")); err != nil {
		t.Fatalf("tile nativo z%d não pré-gerado", tz)
	}

	// Sem internet: catálogo e arquivos fora do ar; a mesma área reabre pela composição guardada.
	srv.Close()
	off := &Servico{Dir: dir, Catalogo: &catalogoFalso{err: cartografia.ErrSemConexao}, Agora: func() time.Time { return agora }}
	c2, err := off.Preparar(context.Background(), pts, cfg, nil)
	if err != nil || !c2.Guardada || c2.ID != comp.ID {
		t.Fatalf("offline: %v %+v", err, c2)
	}
	if _, cob, err := off.Folha(context.Background(), c2.ID, 0); err != nil || cob < 0.999 {
		t.Fatal("folha offline", err)
	}
	if b, err := off.Tile(context.Background(), c2.ID, tz, tx, ty); err != nil || len(b) == 0 {
		t.Fatal("tile offline", err)
	}
	// Área nova, sem internet e sem reserva: erro de conexão (RN-91).
	if _, err := off.Preparar(context.Background(), []geo.Ponto{{Lat: -11, Lon: -62}}, cfg, nil); err == nil {
		t.Fatal("área nova sem internet deveria falhar")
	}
	// Fora da composição: nenhum tile.
	if b, err := off.Tile(context.Background(), c2.ID, 2, 1, 1); err != nil || b != nil {
		t.Fatal("tile fora da composição", err)
	}
}

func itoa(v int) string { return strconv.Itoa(v) }
