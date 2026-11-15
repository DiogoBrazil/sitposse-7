package inpe

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"sitposse7/desktop/internal/cartografia"
	"sitposse7/desktop/internal/cartografia/stac"
)

func item(id, col, data string, nuvens any, ativos map[string]any, props map[string]any) map[string]any {
	p := map[string]any{"datetime": data, "eo:cloud_cover": nuvens}
	for k, v := range props {
		p[k] = v
	}
	return map[string]any{"id": id, "collection": col, "bbox": []float64{-63.6, -10.8, -62.6, -9.8}, "properties": p, "assets": ativos,
		"geometry": map[string]any{"type": "Polygon", "coordinates": [][][]float64{{{-63.6, -10.8}, {-63.6, -9.8}, {-62.6, -9.8}, {-62.6, -10.8}, {-63.6, -10.8}}}}}
}

func TestCandidatasFiltraEDeduplica(t *testing.T) {
	tci := map[string]any{"tci": map[string]any{"href": "https://x/cb.tif"}}
	s2 := map[string]any{"TCI": map[string]any{"href": "https://x/tci.tif"}, "SCL": map[string]any{"href": "https://x/scl.tif"}}
	vis := map[string]any{"visual": map[string]any{"href": "https://x/mos.tif"}}
	resp := map[string][]any{
		ColecaoCBERS: {item("CBERS4A_WPM_PCA_RGB321_20261013_227_126", ColecaoCBERS, "2026-10-13T00:00:00.000000Z", nil, tci, nil)},
		ColecaoS2: {
			item("S2A_MSIL2A_20261107T142801_N0513_R053_T20LMP_20261107T224009", ColecaoS2, "2026-11-07T14:28:01.000000Z", 99.7, s2, nil),
			item("S2B_MSIL2A_20261011T142709_N0513_R053_T20LMP_20261011T181043", ColecaoS2, "2026-10-11T14:27:09.000000Z", 17.9, s2, nil),
			item("S2B_MSIL2A_20261011T142709_N0513_R053_T20LMP_20261011T164729", ColecaoS2, "2026-10-11T14:27:09.000000Z", 18.9, s2, nil),
			item("S2C_MSIL2A_20260916T142711_N0512_R053_T20LMP_20260916T204212", ColecaoS2, "2026-09-16T14:27:11.000000Z", 0.1, s2, nil),
		},
		ColecaoMos: {
			item("MOSAIC_20260401", ColecaoMos, "2026-04-01T00:00:00.000000Z", nil, vis, map[string]any{"start_datetime": "2026-04-01T00:00:00.000000Z", "end_datetime": "2026-06-30T23:59:59.000000Z"}),
			item("MOSAIC_20260101", ColecaoMos, "2026-01-01T00:00:00.000000Z", nil, vis, nil),
			item("MOSAIC_20251001", ColecaoMos, "2025-10-01T00:00:00.000000Z", nil, vis, nil),
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var corpo struct {
			Collections []string `json:"collections"`
		}
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &corpo)
		json.NewEncoder(w).Encode(map[string]any{"features": resp[corpo.Collections[0]], "links": []any{}})
	}))
	defer srv.Close()
	c := &Catalogo{STAC: &stac.Cliente{URL: srv.URL}, P: ParametrosPadrao()}
	cs, err := c.Candidatas(context.Background(), [4]float64{-63.2, -10.1, -63.1, -10.0}, time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	var fontes []string
	for _, x := range cs {
		fontes = append(fontes, x.Fonte+":"+x.ID)
	}
	esperado := []string{
		"cbers4a-wpm:CBERS4A_WPM_PCA_RGB321_20261013_227_126",
		"sentinel2-l2a:S2C_MSIL2A_20260916T142711_N0512_R053_T20LMP_20260916T204212", // menos nuvens primeiro
		"sentinel2-l2a:S2B_MSIL2A_20261011T142709_N0513_R053_T20LMP_20261011T181043", // versão mais recente do processamento
		"mosaico-s2:MOSAIC_20260401", "mosaico-s2:MOSAIC_20260101",
	}
	if len(fontes) != len(esperado) {
		t.Fatalf("candidatas: %v", fontes)
	}
	for i := range esperado {
		if fontes[i] != esperado[i] {
			t.Fatalf("candidatas: %v", fontes)
		}
	}
	if cs[1].URLSCL == "" || cs[3].Ate == nil || cs[3].Periodo() != "01/04/2026 a 30/06/2026" {
		t.Fatalf("metadados: %+v %+v", cs[1], cs[3])
	}
	if cs[0].Atribuicao() != "Imagem CBERS-4A/WPM © INPE (CC BY 4.0)" {
		t.Fatal(cs[0].Atribuicao())
	}
}

// Teste com a rede real (opcional): SITPOSSE7_REDE=1 go test ./internal/cartografia/inpe/ -run Real -v
func TestCandidatasReaisAriquemes(t *testing.T) {
	if os.Getenv("SITPOSSE7_REDE") != "1" {
		t.Skip("defina SITPOSSE7_REDE=1 para consultar o catálogo do INPE")
	}
	cs, err := Novo().Candidatas(context.Background(), [4]float64{-63.20, -10.10, -63.14, -10.05}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	cont := map[string]int{}
	for _, c := range cs {
		cont[c.Fonte]++
		t.Logf("%s %s %s nuvens=%v", c.Fonte, c.Periodo(), c.ID, c.Nuvens)
	}
	if cont[cartografia.FonteS2] == 0 || cont[cartografia.FonteMosaico] == 0 {
		t.Fatalf("candidatas: %v", cont)
	}
}
