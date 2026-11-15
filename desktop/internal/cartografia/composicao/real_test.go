package composicao

import (
	"context"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"sitposse7/desktop/internal/cartografia/geo"
	"sitposse7/desktop/internal/cartografia/inpe"
)

// Teste regional com a rede real (opcional):
//
//	SITPOSSE7_REDE=1 SITPOSSE7_SAIDA=/tmp/x go test ./internal/cartografia/composicao/ -run Real -v -timeout 30m
//
// Ariquemes (−10,075; −63,169): cenário pequeno (5 propriedades) e faixa de 2 × 10 km (20).
func TestRealAriquemes(t *testing.T) {
	if os.Getenv("SITPOSSE7_REDE") != "1" {
		t.Skip("defina SITPOSSE7_REDE=1")
	}
	saida := os.Getenv("SITPOSSE7_SAIDA")
	dir := t.TempDir()
	if d := os.Getenv("SITPOSSE7_CACHE"); d != "" {
		dir = d
	}
	r := rand.New(rand.NewSource(7))
	cenarios := map[string][]geo.Ponto{}
	for i := 0; i < 5; i++ {
		cenarios["5-proximas"] = append(cenarios["5-proximas"], geo.Ponto{Lat: -10.075 + r.Float64()*0.01 - 0.005, Lon: -63.169 + r.Float64()*0.01 - 0.005})
	}
	for i := 0; i < 20; i++ {
		cenarios["20-faixa-2x10km"] = append(cenarios["20-faixa-2x10km"], geo.Ponto{Lat: -10.03 - r.Float64()*0.09, Lon: -63.18 + r.Float64()*0.018})
	}
	for _, nome := range []string{"5-proximas", "20-faixa-2x10km"} {
		s := &Servico{Dir: dir, Catalogo: inpe.Novo()}
		ini := time.Now()
		var ultimo Progresso
		comp, err := s.Preparar(context.Background(), cenarios[nome], geo.ConfigPadrao, func(p Progresso) { ultimo = p }, false)
		if err != nil {
			t.Fatal(nome, err)
		}
		t.Logf("%s: %s, %d setores, %.0f m/px geral, rede %.1f MB, cache %.1f MB, mosaico=%v", nome, time.Since(ini).Round(time.Second),
			len(comp.Folhas.Setores), comp.Folhas.MetrosPxGeral, float64(ultimo.BytesRede)/1e6, float64(ultimo.BytesCache)/1e6, comp.Mosaico)
		for _, p := range comp.Partes {
			t.Logf("   %s %s %.0f m %.0f%% estimada=%v", p.Rotulo, p.Periodo, p.ResolucaoM, 100*p.Fracao, p.NuvemEstimada)
		}
		for _, c := range comp.Receita.Cenas {
			t.Logf("   cor %s %s R²=%.2f", c.Candidata.ID, c.Cor.Tipo, c.Cor.R2)
		}
		for _, a := range comp.Avisos {
			t.Logf("   aviso: %s", a)
		}
		if saida == "" {
			continue
		}
		for i := 0; i <= len(comp.Folhas.Setores) && i < 3; i++ {
			img, cob, err := s.Folha(context.Background(), comp.ID, i)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("   folha %d cobertura %.4f", i, cob)
			f, _ := os.Create(filepath.Join(saida, nome+"-folha-"+string(rune('0'+i))+".png"))
			png.Encode(f, img)
			f.Close()
		}
	}
}
