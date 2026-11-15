package estatisticas

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"sitposse7/desktop/internal/banco"
	"sitposse7/desktop/internal/testutil"
)

// Fixture shared/test-fixtures/estatisticas (docs/requirements/estatisticas.md §22-A).
type itemFx struct {
	Codigo            string  `json:"codigo"`
	NomePersonalizado *string `json:"nome_personalizado"`
	NomeNormalizado   *string `json:"nome_normalizado"`
	Quantidade        int64   `json:"quantidade"`
	FamiliaUUID       string  `json:"familia_uuid"`
}
type familiaFx struct {
	UUID        string           `json:"uuid"`
	Nome        string           `json:"nome"`
	Responsavel string           `json:"responsavel"`
	EquipeUUID  string           `json:"equipe_uuid"`
	Quantidades map[string]int64 `json:"quantidades"`
	Area        *struct {
		Milesimos int64  `json:"milesimos"`
		Unidade   string `json:"unidade"`
	} `json:"area"`
	Localizacao bool     `json:"localizacao"`
	Fotos       int      `json:"fotos"`
	CriadoEm    int64    `json:"criado_em"`
	RecebidoEm  int64    `json:"recebido_em"`
	Itens       []itemFx `json:"itens"`
}
type baseFx struct {
	Estudos  []struct{ UUID, Codigo, Nome string } `json:"estudos"`
	Jornadas []struct {
		UUID       string `json:"uuid"`
		EstudoUUID string `json:"estudo_uuid"`
		Numero     int    `json:"numero"`
		Data       string `json:"data"`
	} `json:"jornadas"`
	Equipes []struct {
		UUID        string `json:"uuid"`
		JornadaUUID string `json:"jornada_uuid"`
		Nome        string `json:"nome"`
		Codigo      string `json:"codigo"`
	} `json:"equipes"`
	Familias            []familiaFx `json:"familias"`
	FamiliasOutroEstudo []familiaFx `json:"familias_outro_estudo"`
	Excluidos           struct {
		Familias []familiaFx `json:"familias"`
		Itens    []itemFx    `json:"itens"`
		Fotos    []struct {
			FamiliaUUID string `json:"familia_uuid"`
		} `json:"fotos"`
	} `json:"excluidos"`
}
type esperadoFx struct {
	Familias       int                           `json:"familias"`
	FamiliasBase   *int                          `json:"familias_base"`
	UUIDs          []string                      `json:"uuids"`
	Valores        map[string]int64              `json:"valores"`
	AreaHa         *string                       `json:"area_ha"`
	ComArea        *int                          `json:"com_area"`
	SemArea        *int                          `json:"sem_area"`
	ComLocalizacao *int                          `json:"com_localizacao"`
	SemLocalizacao *int                          `json:"sem_localizacao"`
	ComFoto        *int                          `json:"com_foto"`
	SemFoto        *int                          `json:"sem_foto"`
	Itens          map[string][2]int64           `json:"itens"` // chave → [total, famílias]
	Faixas         map[string]int                `json:"faixas"`
	PorEquipe      map[string][2]json.RawMessage `json:"por_equipe"` // uuid → [famílias, área ha]
	PorFamilia     map[string]map[string]int64   `json:"por_familia"`
}
type casosFx struct {
	Casos []struct {
		ID       string          `json:"id"`
		Pergunta string          `json:"pergunta"`
		Filtro   json.RawMessage `json:"filtro"`
		Esperado esperadoFx      `json:"esperado"`
	} `json:"casos"`
}

var seq int

func novoUUID() string { seq++; return fmt.Sprintf("0b1e0000-0000-4000-8000-%012d", seq) }

func inserir(t testing.TB, db *sql.DB, f familiaFx, excluido bool) {
	t.Helper()
	var exc any
	if excluido {
		exc = f.RecebidoEm + 1
	}
	var mil, un, lat, lon, gps any
	if f.Area != nil {
		mil, un = f.Area.Milesimos, f.Area.Unidade
	}
	if f.Localizacao {
		lat, lon, gps = -10.4, -62.4, "gnss"
	}
	q := f.Quantidades
	exec(t, db, `INSERT INTO propriedades (uuid, coleta_uuid, equipe_uuid, nome_propriedade, responsavel_familia, pessoas_adultos, pessoas_jovens, pessoas_criancas,
		veic_caminhoes, veic_automoveis_utilitarios, veic_motos_ciclomotores, bov_adultos, bov_bezerros, equ_adultos, equ_potros,
		area_valor_milesimos, area_unidade, lat, lon, gps_origem, created_at, updated_at, excluido_em, rev_origem, recebido_em, origem_instalacao, registro_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, 'teste', '{}')`,
		f.UUID, "coleta-"+f.EquipeUUID, f.EquipeUUID, f.Nome, f.Responsavel, q["pessoas_adultos"], q["pessoas_jovens"], q["pessoas_criancas"],
		q["veic_caminhoes"], q["veic_automoveis_utilitarios"], q["veic_motos_ciclomotores"], q["bov_adultos"], q["bov_bezerros"], q["equ_adultos"], q["equ_potros"],
		mil, un, lat, lon, gps, f.CriadoEm, f.CriadoEm, exc, f.RecebidoEm)
	for _, it := range f.Itens {
		it.FamiliaUUID = f.UUID
		inserirItem(t, db, it, f.RecebidoEm, false)
	}
	for i := 0; i < f.Fotos; i++ {
		inserirFoto(t, db, f.UUID, f.RecebidoEm, false)
	}
}

func inserirItem(t testing.TB, db *sql.DB, it itemFx, quando int64, excluido bool) {
	t.Helper()
	var exc any
	if excluido {
		exc = quando + 1
	}
	cat := map[string]string{"eletro": "eletrodomestico", "movel": "movel", "ave": "ave"}[strings.SplitN(it.Codigo, ".", 2)[0]]
	exec(t, db, `INSERT INTO itens_propriedade (uuid, propriedade_uuid, categoria, codigo_catalogo, nome_personalizado, nome_normalizado, quantidade,
		created_at, updated_at, excluido_em, rev_origem, recebido_em, origem_instalacao, registro_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, 'teste', '{}')`,
		novoUUID(), it.FamiliaUUID, cat, it.Codigo, it.NomePersonalizado, it.NomeNormalizado, it.Quantidade, quando, quando, exc, quando)
}

func inserirFoto(t testing.TB, db *sql.DB, prop string, quando int64, excluida bool) {
	t.Helper()
	var exc, caminho any = nil, "ab/foto.jpg"
	if excluida {
		exc, caminho = quando+1, nil
	}
	exec(t, db, `INSERT INTO fotografias (uuid, propriedade_uuid, caminho_relativo, mime, tamanho_bytes, sha256, largura, altura, capturada_em,
		created_at, updated_at, excluido_em, rev_origem, recebido_em, origem_instalacao, registro_json)
		VALUES (?, ?, ?, 'image/jpeg', 1000, ?, 2560, 1920, ?, ?, ?, ?, 1, ?, 'teste', '{}')`,
		novoUUID(), prop, caminho, strings.Repeat("a", 64), quando, quando, quando, exc, quando)
}

func exec(t testing.TB, db *sql.DB, q string, args ...any) {
	t.Helper()
	if _, err := db.Exec(q, args...); err != nil {
		t.Fatalf("%v\n%s", err, q)
	}
}

// bancoComBase grava a fixture num banco migrado (com os registros excluídos).
func bancoComBase(t testing.TB, b baseFx) *banco.Banco {
	t.Helper()
	ctx := context.Background()
	bd, err := banco.Abrir(ctx, filepath.Join(t.TempDir(), "central.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { bd.Fechar() })
	db := bd.Escrita
	exec(t, db, `INSERT INTO dispositivos (instalacao_uuid, credencial_hash, pareado_em) VALUES ('inst-teste', 'h', 1)`)
	for i, e := range b.Estudos {
		exec(t, db, `INSERT INTO estudos (uuid, codigo, nome, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`, e.UUID, e.Codigo, e.Nome, i+1, i+1)
	}
	for _, j := range b.Jornadas {
		exec(t, db, `INSERT INTO jornadas (uuid, estudo_uuid, numero, data, created_at, updated_at) VALUES (?, ?, ?, ?, 1, 1)`, j.UUID, j.EstudoUUID, j.Numero, j.Data)
	}
	for _, e := range b.Equipes {
		exec(t, db, `INSERT INTO equipes (uuid, jornada_uuid, nome, nome_normalizado, codigo, created_at, updated_at) VALUES (?, ?, ?, lower(?), ?, 1, 1)`,
			e.UUID, e.JornadaUUID, e.Nome, e.Nome, e.Codigo)
		exec(t, db, `INSERT INTO coletas (uuid, instalacao_uuid, codigo_informado, codigo_original, equipe_uuid, destino_confirmado_em, created_at, updated_at, recebido_em)
			VALUES (?, 'inst-teste', ?, ?, ?, 1, 1, 1, 1)`, "coleta-"+e.UUID, e.Codigo, e.Codigo, e.UUID)
	}
	for _, f := range append(append([]familiaFx{}, b.Familias...), b.FamiliasOutroEstudo...) {
		inserir(t, db, f, false)
	}
	for _, f := range b.Excluidos.Familias {
		inserir(t, db, f, true)
	}
	for _, it := range b.Excluidos.Itens {
		inserirItem(t, db, it, 1, true)
	}
	for _, f := range b.Excluidos.Fotos {
		inserirFoto(t, db, f.FamiliaUUID, 1, true)
	}
	return bd
}

func TestCasosDaFixture(t *testing.T) {
	var b baseFx
	var c casosFx
	testutil.LerFixture(t, "estatisticas/base.v2.json", &b)
	testutil.LerFixture(t, "estatisticas/casos.v2.json", &c)
	bd := bancoComBase(t, b)
	base, err := Carregar(context.Background(), bd.Leitura)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Casos) < 22 {
		t.Fatalf("fixture com %d casos; esperado pelo menos 19", len(c.Casos))
	}
	for _, cs := range c.Casos {
		t.Run(cs.ID, func(t *testing.T) {
			var f Filtro
			if err := json.Unmarshal(cs.Filtro, &f); err != nil {
				t.Fatal(err)
			}
			r, err := Calcular(base, f)
			if err != nil {
				t.Fatal(err)
			}
			e := cs.Esperado
			if r.Totais.Familias != e.Familias || len(r.Familias) != e.Familias {
				t.Errorf("%s: famílias = %d (lista %d), esperado %d", cs.Pergunta, r.Totais.Familias, len(r.Familias), e.Familias)
			}
			base := len(b.Familias)
			if e.FamiliasBase != nil {
				base = *e.FamiliasBase
			}
			if r.FamiliasBase != base {
				t.Errorf("base = %d, esperado %d (excluídos e outros estudos não contam)", r.FamiliasBase, base)
			}
			if e.UUIDs != nil {
				var got []string
				for _, p := range r.Familias {
					got = append(got, p.UUID)
				}
				sort.Strings(got)
				if strings.Join(got, ",") != strings.Join(e.UUIDs, ",") {
					t.Errorf("%s: famílias %v, esperado %v", cs.Pergunta, got, e.UUIDs)
				}
			}
			for k, v := range e.Valores {
				if r.Totais.Valores[k] != v {
					t.Errorf("%s: %s = %d, esperado %d", cs.Pergunta, k, r.Totais.Valores[k], v)
				}
			}
			if e.AreaHa != nil && r.Totais.AreaHa != *e.AreaHa {
				t.Errorf("área = %s, esperado %s", r.Totais.AreaHa, *e.AreaHa)
			}
			for nome, par := range map[string][2]*int{
				"com_area": {&r.Totais.ComArea, e.ComArea}, "sem_area": {&r.Totais.SemArea, e.SemArea},
				"com_localizacao": {&r.Totais.ComLocalizacao, e.ComLocalizacao}, "sem_localizacao": {&r.Totais.SemLocalizacao, e.SemLocalizacao},
				"com_foto": {&r.Totais.ComFoto, e.ComFoto}, "sem_foto": {&r.Totais.SemFoto, e.SemFoto},
			} {
				if par[1] != nil && *par[0] != *par[1] {
					t.Errorf("%s = %d, esperado %d", nome, *par[0], *par[1])
				}
			}
			itens := map[string]TipoItem{}
			for _, it := range r.Itens {
				itens[it.Campo.Chave] = it
			}
			for k, v := range e.Itens {
				if it := itens[k]; it.Total != v[0] || int64(it.Familias) != v[1] {
					t.Errorf("item %s = (%d, %d famílias), esperado (%d, %d)", k, it.Total, it.Familias, v[0], v[1])
				}
			}
			faixas := map[string]int{}
			for _, x := range r.FaixasArea {
				faixas[x.Chave] = x.Familias
			}
			for k, v := range e.Faixas {
				if faixas[k] != v {
					t.Errorf("faixa %s = %d, esperado %d", k, faixas[k], v)
				}
			}
			eq := map[string]Grupo{}
			for _, g := range r.PorEquipe {
				eq[g.Chave] = g
			}
			for k, v := range e.PorEquipe {
				var n int
				var ha string
				_ = json.Unmarshal(v[0], &n)
				_ = json.Unmarshal(v[1], &ha)
				if g := eq[k]; g.Totais.Familias != n || g.Totais.AreaHa != ha {
					t.Errorf("equipe %s = (%d, %s), esperado (%d, %s)", k, g.Totais.Familias, g.Totais.AreaHa, n, ha)
				}
			}
			fam := map[string]Familia{}
			for _, p := range r.Familias {
				fam[p.UUID] = p
			}
			for u, vs := range e.PorFamilia {
				for k, v := range vs {
					if fam[u].Valores[k] != v {
						t.Errorf("família %s: %s = %d, esperado %d", u, k, fam[u].Valores[k], v)
					}
				}
			}
			// Coerência (RN-81): a soma das famílias e a soma das equipes batem com os totais.
			somaFam, somaEq := map[string]int64{}, map[string]int64{}
			for _, p := range r.Familias {
				for k, v := range p.Valores {
					somaFam[k] += v
				}
			}
			somaJor := map[string]int64{}
			for _, g := range r.PorEquipe {
				for k, v := range g.Totais.Valores {
					somaEq[k] += v
				}
			}
			for _, g := range r.PorJornada {
				for k, v := range g.Totais.Valores {
					somaJor[k] += v
				}
			}
			for k, v := range r.Totais.Valores {
				if somaFam[k] != v || somaEq[k] != v || somaJor[k] != v {
					t.Errorf("incoerente em %s: total %d, famílias %d, equipes %d, jornadas %d", k, v, somaFam[k], somaEq[k], somaJor[k])
				}
			}
		})
	}
}

func TestFiltroInvalido(t *testing.T) {
	b := &Base{}
	for _, f := range []Filtro{
		{Condicoes: []Condicao{{Campo: "inexistente", Operador: "tem"}}},
		{Condicoes: []Condicao{{Campo: "bov_total", Operador: "aproximadamente"}}},
		{Condicoes: []Condicao{{Campo: "bov_total", Operador: "maior", Valor: -1}}},
		{PeriodoCampo: "ontem"},
	} {
		if _, err := Calcular(b, f); !errors.Is(err, ErrFiltroInvalido) {
			t.Errorf("%+v: erro = %v, esperado ErrFiltroInvalido", f, err)
		}
	}
}

// RNF-08: painel com filtros em < 300 ms no volume 3× (900 propriedades, RNF-07).
func TestDesempenhoVolume3x(t *testing.T) {
	var b baseFx
	testutil.LerFixture(t, "estatisticas/base.v2.json", &b)
	modelo := b.Familias
	b.Familias, b.FamiliasOutroEstudo = nil, nil
	b.Excluidos.Familias, b.Excluidos.Itens, b.Excluidos.Fotos = nil, nil, nil
	for i := 0; i < 900; i++ {
		f := modelo[i%len(modelo)]
		f.UUID = fmt.Sprintf("9a7e0000-0000-4000-8000-%012d", i)
		f.Nome = fmt.Sprintf("%s %d", f.Nome, i)
		f.Fotos = 5
		b.Familias = append(b.Familias, f)
	}
	bd := bancoComBase(t, b)
	f := Filtro{Equipes: []string{b.Equipes[0].UUID}, Condicoes: []Condicao{{Campo: "bov_total", Operador: "maior", Valor: 5}}, Busca: "sitio"}
	inicio := time.Now()
	base, err := Carregar(context.Background(), bd.Leitura)
	if err != nil {
		t.Fatal(err)
	}
	r, err := Calcular(base, f)
	if err != nil {
		t.Fatal(err)
	}
	d := time.Since(inicio)
	t.Logf("900 propriedades, 4.500 fotos: carregar + calcular em %v (%d famílias no filtro)", d, r.Totais.Familias)
	if d > 300*time.Millisecond {
		t.Errorf("levou %v; o limite é 300 ms (RNF-08)", d)
	}
}
