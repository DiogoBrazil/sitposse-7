// Package selecao escolhe, de forma automática e reproduzível, as cenas que compõem a imagem
// da carta (ADR-0027, RN-95): avaliação por célula da área das propriedades (não pela nuvem
// da cena inteira), filtros eliminatórios, pontuação configurável e escolha gulosa com
// desempate determinístico.
package selecao

import (
	"math"
	"sort"
	"time"

	"sitposse7/desktop/internal/cartografia"
	"sitposse7/desktop/internal/cartografia/raster"
)

// Criterios reúne TODOS os pesos e limiares da seleção (calibráveis num só lugar).
type Criterios struct {
	MaxSemDados    float64       // fração de pixels sem dados tolerada na célula
	MaxNuvem       float64       // fração de nuvem + sombra tolerada na célula
	IdadeRef       time.Duration // idade que zera o termo de atualidade
	PesoNivel      float64       // nível por resolução: domina os demais (2 m > 10 m > mosaico)
	PesoNuvem      float64
	PesoIdade      float64
	PesoAjuste     float64 // qualidade do ajuste de cor à referência (R²): névoa, mudanças
	MinFracaoCena  float64 // uma cena só entra se atender ao menos esta fração das células
	MinCelulasCena int     // e ao menos estas células (evita retalhos)
}

// Padrao (ADR-0027). Calibração inicial; ajustes ficam registrados no changelog.
func Padrao() Criterios {
	return Criterios{MaxSemDados: 0.005, MaxNuvem: 0.02, IdadeRef: 366 * 24 * time.Hour,
		PesoNivel: 1000, PesoNuvem: 10, PesoIdade: 4, PesoAjuste: 3, MinFracaoCena: 0.02, MinCelulasCena: 3}
}

// Nivel por resolução: CBERS 2 m = 2; Sentinel-2 (INPE ou CDSE) = 1; mosaico = 0.
func Nivel(c cartografia.Candidata) float64 {
	switch c.Fonte {
	case cartografia.FonteCBERS:
		return 2
	case cartografia.FonteS2, cartografia.FonteCDSE:
		return 1
	}
	return 0
}

// Entrada: uma candidata avaliada nos painéis.
type Entrada struct {
	Candidata cartografia.Candidata
	Av        *raster.Avaliacao
}

// Escolha: a receita (cenas em ordem de prioridade e a cena de cada célula) e o diagnóstico.
type Escolha struct {
	Receita   raster.Receita
	Total     int   // células avaliadas
	Limitadas int   // células sem candidata dentro dos limites (usada a menos ruim)
	SemImagem int   // células sem nenhuma imagem
	Celulas   []int // células atribuídas a cada cena da receita
}

type celula struct{ p, k int }

func (cr Criterios) elegivel(e Entrada, c celula) bool {
	return float64(e.Av.SemDados[c.p][c.k]) <= cr.MaxSemDados && float64(e.Av.Nuvem[c.p][c.k]) <= cr.MaxNuvem
}

func (cr Criterios) pontuacao(e Entrada, c celula, agora time.Time) float64 {
	ref := e.Candidata.Data
	if e.Candidata.Ate != nil {
		ref = *e.Candidata.Ate
	}
	idade := math.Min(1, math.Max(0, agora.Sub(ref).Hours()/cr.IdadeRef.Hours()))
	return cr.PesoNivel*Nivel(e.Candidata) + cr.PesoNuvem*(1-float64(e.Av.Nuvem[c.p][c.k])/math.Max(cr.MaxNuvem, 1e-9)) +
		cr.PesoIdade*(1-idade) + cr.PesoAjuste*e.Av.Cor.R2
}

// antes define o desempate determinístico: fonte preferida, data mais recente, id.
func antes(a, b cartografia.Candidata) bool {
	if pa, pb := cartografia.PrioridadeFonte(a.Fonte), cartografia.PrioridadeFonte(b.Fonte); pa != pb {
		return pa < pb
	}
	if !a.Data.Equal(b.Data) {
		return a.Data.After(b.Data)
	}
	return a.ID < b.ID
}

// Escolher monta a receita. paineis[0] é o da carta geral; os demais, os dos setores.
func Escolher(ents []Entrada, paineis []raster.Painel, agora time.Time, cr Criterios) *Escolha {
	// Ordem estável de entrada, independente da ordem do catálogo.
	ents = append([]Entrada(nil), ents...)
	sort.SliceStable(ents, func(i, j int) bool { return antes(ents[i].Candidata, ents[j].Candidata) })

	var todas []celula
	for p, pn := range paineis {
		for k := 0; k < pn.Cel.NX*pn.Cel.NY; k++ {
			todas = append(todas, celula{p, k})
		}
	}
	livre := map[celula]bool{}
	for _, c := range todas {
		livre[c] = true
	}
	atrib := map[celula]int{} // célula → índice em ents
	var escolhidas []int
	usada := make([]bool, len(ents))
	minimo := max(cr.MinCelulasCena, int(math.Ceil(cr.MinFracaoCena*float64(len(todas)))))
	for len(livre) > 0 {
		melhor, melhorGanho := -1, 0.0
		for i, e := range ents {
			if usada[i] {
				continue
			}
			ganho, n := 0.0, 0
			for _, c := range todas {
				if livre[c] && cr.elegivel(e, c) {
					ganho += cr.pontuacao(e, c, agora)
					n++
				}
			}
			if n == 0 || (n < minimo && n < len(livre)) {
				continue
			}
			// Regra hierárquica: primeiro o nível de resolução (2 m ≤ 12 meses vence sempre
			// que atende ao mínimo de células), depois o ganho de pontuação.
			if melhor < 0 || Nivel(e.Candidata) > Nivel(ents[melhor].Candidata) ||
				(Nivel(e.Candidata) == Nivel(ents[melhor].Candidata) && ganho > melhorGanho+1e-9) {
				melhor, melhorGanho = i, ganho
			}
		}
		if melhor < 0 {
			break
		}
		usada[melhor] = true
		escolhidas = append(escolhidas, melhor)
		for _, c := range todas {
			if livre[c] && cr.elegivel(ents[melhor], c) {
				atrib[c] = melhor
				delete(livre, c)
			}
		}
	}
	es := &Escolha{Total: len(todas)}
	// Células sem candidata dentro dos limites: a menos ruim (menos sem dados, menos nuvem,
	// maior nível); sem nenhuma imagem, a célula fica vazia (RN-93).
	for _, c := range todas {
		if !livre[c] {
			continue
		}
		mi := -1
		for i, e := range ents {
			if e.Av.SemDados[c.p][c.k] >= 0.5 {
				continue
			}
			if mi < 0 {
				mi = i
				continue
			}
			m := ents[mi]
			sd, sdm := e.Av.SemDados[c.p][c.k], m.Av.SemDados[c.p][c.k]
			nv, nvm := e.Av.Nuvem[c.p][c.k], m.Av.Nuvem[c.p][c.k]
			if sd < sdm-1e-6 || (math.Abs(float64(sd-sdm)) <= 1e-6 && (nv < nvm-1e-6 || (math.Abs(float64(nv-nvm)) <= 1e-6 && Nivel(e.Candidata) > Nivel(m.Candidata)))) {
				mi = i
			}
		}
		if mi < 0 {
			es.SemImagem++
			continue
		}
		es.Limitadas++
		atrib[c] = mi
		if !usada[mi] {
			usada[mi] = true
			escolhidas = append(escolhidas, mi)
		}
	}
	// O mosaico de referência (o mais recente) entra por último para preencher buracos.
	for i, e := range ents {
		if e.Candidata.Fonte == cartografia.FonteMosaico && !usada[i] {
			usada[i] = true
			escolhidas = append(escolhidas, i)
			break
		}
	}
	pos := map[int]int{}
	for p, i := range escolhidas {
		pos[i] = p
		es.Receita.Cenas = append(es.Receita.Cenas, raster.CenaReceita{Candidata: ents[i].Candidata, Cor: ents[i].Av.Cor, NuvemEstimada: ents[i].Av.NuvemEstimada})
	}
	es.Celulas = make([]int, len(escolhidas))
	grades := make([]raster.Celulas, len(paineis))
	for p, pn := range paineis {
		g := pn.Cel
		g.Cena = make([]int16, len(pn.Cel.Cena))
		for k := range g.Cena {
			g.Cena[k] = -1
			if i, ok := atrib[celula{p, k}]; ok {
				g.Cena[k] = int16(pos[i])
				es.Celulas[pos[i]]++
			}
		}
		grades[p] = g
	}
	if len(grades) > 0 {
		es.Receita.Geral, es.Receita.Detalhe = grades[0], grades[1:]
	}
	es.Receita.Versao = raster.VersaoReceita
	es.Receita.Tom = raster.TomNeutro()
	return es
}

// FracaoNivel: fração das células do painel p atribuídas a cenas do nível dado (ex.: 2 = 2 m).
func (es *Escolha) FracaoNivel(p int, nivel float64) float64 {
	var g raster.Celulas
	if p == 0 {
		g = es.Receita.Geral
	} else {
		g = es.Receita.Detalhe[p-1]
	}
	n, t := 0, 0
	for _, s := range g.Cena {
		t++
		if s >= 0 && Nivel(es.Receita.Cenas[s].Candidata) == nivel {
			n++
		}
	}
	if t == 0 {
		return 0
	}
	return float64(n) / float64(t)
}
