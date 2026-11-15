// Package inpe busca, no catálogo STAC do INPE (Brazil Data Cube), as imagens candidatas
// para a carta (ADR-0027): CBERS-4A/WPM fusionado (2 m), Sentinel-2 L2A (10 m, com SCL) e o
// mosaico trimestral Sentinel-2. Acesso público, sem credenciais. Endereços e nomes de
// coleção conferidos em 15/11/2026.
package inpe

import (
	"context"
	"sort"
	"strings"
	"time"

	"sitposse7/desktop/internal/cartografia"
	"sitposse7/desktop/internal/cartografia/stac"
)

const (
	URLSTAC      = "https://data.inpe.br/bdc/stac/v1"
	ColecaoCBERS = "CB4A-WPM-PCA-FUSED-1"
	ColecaoS2    = "S2_L2A-1"
	ColecaoMos   = "mosaic-s2-brazil-3m-1"
)

// Parametros da busca.
type Parametros struct {
	IdadeMaxCBERS time.Duration // 12 meses (decisão do usuário, RN-95)
	IdadeMaxS2    time.Duration // período de busca do Sentinel-2
	NuvensMaxS2   float64       // pré-filtro pela nuvem da cena inteira (%)
	MaxS2         int           // quantas cenas S2 (as de menos nuvens) seguem para avaliação
	MaxMosaicos   int
}

// ParametrosPadrao (ADR-0027).
func ParametrosPadrao() Parametros {
	return Parametros{IdadeMaxCBERS: 366 * 24 * time.Hour, IdadeMaxS2: 366 * 24 * time.Hour, NuvensMaxS2: 60, MaxS2: 12, MaxMosaicos: 2}
}

// Catalogo do INPE.
type Catalogo struct {
	STAC *stac.Cliente
	P    Parametros
}

// Novo catálogo com os endereços oficiais.
func Novo() *Catalogo {
	return &Catalogo{STAC: &stac.Cliente{URL: URLSTAC}, P: ParametrosPadrao()}
}

func intersecta(a, b [4]float64) bool {
	return a[0] <= b[2] && b[0] <= a[2] && a[1] <= b[3] && b[1] <= a[3]
}

// Candidatas sobre o retângulo (graus: oeste, sul, leste, norte).
func (c *Catalogo) Candidatas(ctx context.Context, bbox [4]float64, agora time.Time) ([]cartografia.Candidata, error) {
	var out []cartografia.Candidata
	// CBERS-4A WPM fusionado: sem metadado de nuvens; todas as cenas recentes seguem.
	its, err := c.STAC.Buscar(ctx, stac.Busca{Colecoes: []string{ColecaoCBERS}, BBox: bbox, De: agora.Add(-c.P.IdadeMaxCBERS), Ate: agora})
	if err != nil {
		return nil, err
	}
	for _, it := range its {
		if a, ok := it.Ativos["tci"]; ok && intersecta(it.BBox, bbox) {
			out = append(out, cartografia.Candidata{ID: it.ID, Fonte: cartografia.FonteCBERS, Data: it.Data, ResolucaoM: 2, URLRGB: a.Href, BBox: it.BBox})
		}
	}
	// Sentinel-2 L2A: TCI + SCL; uma versão por tile e data (a de processamento mais recente).
	its, err = c.STAC.Buscar(ctx, stac.Busca{Colecoes: []string{ColecaoS2}, BBox: bbox, De: agora.Add(-c.P.IdadeMaxS2), Ate: agora, MaxItens: 800})
	if err != nil {
		return nil, err
	}
	porChave := map[string]stac.Item{}
	for _, it := range its {
		if it.Nuvens != nil && *it.Nuvens > c.P.NuvensMaxS2 {
			continue
		}
		if _, ok := it.Ativos["TCI"]; !ok || !intersecta(it.BBox, bbox) {
			continue
		}
		k := it.ID
		if i := strings.LastIndex(k, "_"); i > 0 {
			k = k[:i]
		}
		if ant, ok := porChave[k]; !ok || it.ID > ant.ID {
			porChave[k] = it
		}
	}
	var s2 []stac.Item
	for _, it := range porChave {
		s2 = append(s2, it)
	}
	sort.Slice(s2, func(i, j int) bool {
		ni, nj := nuvem(s2[i]), nuvem(s2[j])
		if ni != nj {
			return ni < nj
		}
		if !s2[i].Data.Equal(s2[j].Data) {
			return s2[i].Data.After(s2[j].Data)
		}
		return s2[i].ID < s2[j].ID
	})
	if len(s2) > c.P.MaxS2 {
		s2 = s2[:c.P.MaxS2]
	}
	for _, it := range s2 {
		out = append(out, cartografia.Candidata{ID: it.ID, Fonte: cartografia.FonteS2, Data: it.Data, ResolucaoM: 10,
			URLRGB: it.Ativos["TCI"].Href, URLSCL: it.Ativos["SCL"].Href, Nuvens: it.Nuvens, BBox: it.BBox})
	}
	// Mosaico trimestral: os mais recentes.
	its, err = c.STAC.Buscar(ctx, stac.Busca{Colecoes: []string{ColecaoMos}, BBox: bbox, De: agora.Add(-3 * 366 * 24 * time.Hour), Ate: agora})
	if err != nil {
		return nil, err
	}
	sort.Slice(its, func(i, j int) bool { return its[i].Data.After(its[j].Data) })
	n := 0
	for _, it := range its {
		a, ok := it.Ativos["visual"]
		if !ok || n >= c.P.MaxMosaicos {
			continue
		}
		cand := cartografia.Candidata{ID: it.ID, Fonte: cartografia.FonteMosaico, Data: it.Inicio, ResolucaoM: 10, URLRGB: a.Href, BBox: it.BBox}
		if cand.Data.IsZero() {
			cand.Data = it.Data
		}
		if !it.Fim.IsZero() {
			f := it.Fim
			cand.Ate = &f
		}
		out = append(out, cand)
		n++
	}
	return out, nil
}

func nuvem(it stac.Item) float64 {
	if it.Nuvens == nil {
		return 100
	}
	return *it.Nuvens
}
