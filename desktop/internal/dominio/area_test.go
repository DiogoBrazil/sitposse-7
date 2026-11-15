package dominio

import (
	"errors"
	"testing"

	"sitposse7/desktop/internal/testutil"
)

type casoArea struct {
	Entrada    string   `json:"entrada"`
	Unidade    *string  `json:"unidade"`
	Milesimos  int64    `json:"milesimos"`
	Dm2        int64    `json:"dm2"`
	M2         float64  `json:"m2"`
	HaExibicao string   `json:"ha_exibicao"`
	Erro       string   `json:"erro"`
	Resultado  string   `json:"resultado"`
}

func unidadePtr(s *string) *UnidadeArea {
	if s == nil {
		return nil
	}
	u := UnidadeArea(*s)
	return &u
}

func TestAreaFixturesConversao(t *testing.T) {
	var fx struct {
		Casos     []casoArea `json:"casos"`
		Invalidos []casoArea `json:"invalidos"`
		Vazios    []casoArea `json:"vazios"`
	}
	testutil.LerFixture(t, "area/conversao.json", &fx)
	if len(fx.Casos) == 0 {
		t.Fatal("fixture sem casos")
	}
	for _, c := range fx.Casos {
		a, err := ParseArea(c.Entrada, unidadePtr(c.Unidade))
		if err != nil || a == nil {
			t.Errorf("%q %v: erro inesperado %v", c.Entrada, *c.Unidade, err)
			continue
		}
		if a.Milesimos != c.Milesimos || a.Dm2() != c.Dm2 || FormatarHa(a.Dm2()) != c.HaExibicao {
			t.Errorf("%q %s: obteve milesimos=%d dm2=%d ha=%s; esperado %d %d %s",
				c.Entrada, *c.Unidade, a.Milesimos, a.Dm2(), FormatarHa(a.Dm2()), c.Milesimos, c.Dm2, c.HaExibicao)
		}
		if float64(a.Dm2())/100 != c.M2 {
			t.Errorf("%q: m2=%v esperado %v", c.Entrada, float64(a.Dm2())/100, c.M2)
		}
	}
	for _, c := range fx.Invalidos {
		_, err := ParseArea(c.Entrada, unidadePtr(c.Unidade))
		if err == nil || !errors.Is(err, map[string]error{
			"negativo": ErrAreaNegativa, "mais_de_3_casas": ErrAreaMaisDe3Casas,
			"nao_numerico": ErrAreaNaoNumerica, "separador_de_milhar_nao_suportado": ErrAreaSeparadorMilhar,
			"unidade_obrigatoria": ErrAreaUnidadeObrigatoria, "unidade_invalida": ErrAreaUnidadeInvalida,
		}[c.Erro]) {
			t.Errorf("%q: esperado erro %s, obteve %v", c.Entrada, c.Erro, err)
		}
	}
	for _, c := range fx.Vazios {
		a, err := ParseArea(c.Entrada, nil)
		if a != nil || err != nil {
			t.Errorf("%q: esperado sem_area, obteve %v %v", c.Entrada, a, err)
		}
	}
}

func TestAreaFixturesSomas(t *testing.T) {
	var fx struct {
		Casos []struct {
			Propriedades []struct {
				Milesimos *int64  `json:"milesimos"`
				Unidade   *string `json:"unidade"`
			} `json:"propriedades"`
			TotalDm2        int64  `json:"total_dm2"`
			TotalHaExibicao string `json:"total_ha_exibicao"`
			ComArea         int    `json:"com_area"`
			SemArea         int    `json:"sem_area"`
		} `json:"casos"`
	}
	testutil.LerFixture(t, "area/somas.json", &fx)
	for i, c := range fx.Casos {
		var areas []*Area
		for _, p := range c.Propriedades {
			if p.Milesimos == nil {
				areas = append(areas, nil)
				continue
			}
			areas = append(areas, &Area{Milesimos: *p.Milesimos, Unidade: UnidadeArea(*p.Unidade)})
		}
		total, com, sem := SomaAreas(areas)
		if total != c.TotalDm2 || FormatarHa(total) != c.TotalHaExibicao || com != c.ComArea || sem != c.SemArea {
			t.Errorf("caso %d: obteve %d %s %d %d", i, total, FormatarHa(total), com, sem)
		}
	}
}
