package contratos

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"sitposse7/desktop/internal/testutil"
)

const baseSchemas = "https://sitposse7.local/schemas/"

// compilar carrega todos os schemas de shared/schemas (para resolver os $ref entre eles).
func compilar(t *testing.T, ref string) *jsonschema.Schema {
	t.Helper()
	dir := testutil.FixturePath(t, "../schemas")
	arqs, err := filepath.Glob(filepath.Join(dir, "*.schema.json"))
	if err != nil || len(arqs) == 0 {
		t.Fatalf("schemas não encontrados em %s", dir)
	}
	c := jsonschema.NewCompiler()
	c.AssertFormat()
	for _, a := range arqs {
		f, err := os.Open(a)
		if err != nil {
			t.Fatal(err)
		}
		doc, err := jsonschema.UnmarshalJSON(f)
		f.Close()
		if err != nil {
			t.Fatalf("%s: %v", a, err)
		}
		if err := c.AddResource(baseSchemas+filepath.Base(a), doc); err != nil {
			t.Fatal(err)
		}
	}
	s, err := c.Compile(baseSchemas + ref)
	if err != nil {
		t.Fatalf("compilando %s: %v", ref, err)
	}
	return s
}

func validar(t *testing.T, s *jsonschema.Schema, dado []byte) error {
	t.Helper()
	v, err := jsonschema.UnmarshalJSON(bytes.NewReader(dado))
	if err != nil {
		t.Fatal(err)
	}
	return s.Validate(v)
}

type casoInvalido struct {
	Motivo string          `json:"motivo"`
	Dado   json.RawMessage `json:"dado"`
}

func decodificarEstrito(dado []byte, v any) error {
	d := json.NewDecoder(bytes.NewReader(dado))
	d.DisallowUnknownFields()
	return d.Decode(v)
}

func TestLoteFixtures_SchemaEGo(t *testing.T) {
	s := compilar(t, "lote.v1.schema.json")
	var fx struct {
		Validos   []json.RawMessage `json:"validos"`
		Invalidos []casoInvalido    `json:"invalidos"`
	}
	testutil.LerFixture(t, "contratos/lote.v1.json", &fx)
	if len(fx.Validos) == 0 || len(fx.Invalidos) == 0 {
		t.Fatal("fixture sem casos")
	}
	for i, v := range fx.Validos {
		if err := validar(t, s, v); err != nil {
			t.Errorf("válido %d rejeitado pelo schema: %v", i, err)
		}
		var l Lote
		if err := decodificarEstrito(v, &l); err != nil {
			t.Errorf("válido %d não decodifica: %v", i, err)
			continue
		}
		if err := ValidarLoteCompleto(&l); err != nil {
			t.Errorf("válido %d rejeitado pelo Go: %v", i, err)
		}
	}
	for _, c := range fx.Invalidos {
		if err := validar(t, s, c.Dado); err == nil {
			t.Errorf("inválido %q aceito pelo schema", c.Motivo)
		}
		var l Lote
		if err := decodificarEstrito(c.Dado, &l); err != nil {
			continue // rejeitado na decodificação (ex.: campo extra)
		}
		if err := ValidarLoteCompleto(&l); err == nil {
			t.Errorf("inválido %q aceito pelo Go", c.Motivo)
		}
	}
}

func TestReciboFixtures(t *testing.T) {
	s := compilar(t, "recibo.v1.schema.json")
	var fx struct {
		Validos   []json.RawMessage `json:"validos"`
		Invalidos []casoInvalido    `json:"invalidos"`
		Hash      struct {
			TextoCanonico string `json:"texto_canonico"`
			ReciboSHA256  string `json:"recibo_sha256"`
		} `json:"hash"`
	}
	testutil.LerFixture(t, "contratos/recibo.v1.json", &fx)
	for i, v := range fx.Validos {
		if err := validar(t, s, v); err != nil {
			t.Errorf("válido %d rejeitado: %v", i, err)
		}
		var r Recibo
		if err := decodificarEstrito(v, &r); err != nil {
			t.Fatal(err)
		}
		if got := r.TextoCanonico(); got != fx.Hash.TextoCanonico {
			t.Errorf("texto canônico divergente:\n%q\n%q", got, fx.Hash.TextoCanonico)
		}
		if got := r.CalcularHash(); got != fx.Hash.ReciboSHA256 || got != r.ReciboSHA256 {
			t.Errorf("hash = %s, esperado %s", got, fx.Hash.ReciboSHA256)
		}
	}
	for _, c := range fx.Invalidos {
		if err := validar(t, s, c.Dado); err == nil {
			t.Errorf("inválido %q aceito", c.Motivo)
		}
	}
}

func TestPareamentoFixtures(t *testing.T) {
	sPed := compilar(t, "pareamento.v1.schema.json#/$defs/pedido")
	sResp := compilar(t, "pareamento.v1.schema.json#/$defs/resposta")
	sTodo := compilar(t, "pareamento.v1.schema.json")
	var fx struct {
		Pedidos   []json.RawMessage `json:"pedidos_validos"`
		Respostas []json.RawMessage `json:"respostas_validas"`
		Invalidos []casoInvalido    `json:"invalidos"`
	}
	testutil.LerFixture(t, "contratos/pareamento.v1.json", &fx)
	for i, v := range fx.Pedidos {
		if err := validar(t, sPed, v); err != nil {
			t.Errorf("pedido %d rejeitado: %v", i, err)
		}
		var p PedidoPareamento
		if err := decodificarEstrito(v, &p); err != nil {
			t.Fatal(err)
		}
		if err := ValidarPedidoPareamento(&p); err != nil {
			t.Errorf("pedido %d rejeitado pelo Go: %v", i, err)
		}
	}
	for i, v := range fx.Respostas {
		if err := validar(t, sResp, v); err != nil {
			t.Errorf("resposta %d rejeitada: %v", i, err)
		}
	}
	for _, c := range fx.Invalidos {
		if err := validar(t, sTodo, c.Dado); err == nil {
			t.Errorf("inválido %q aceito", c.Motivo)
		}
	}
}

func TestEstadoLoteFixtures(t *testing.T) {
	s := compilar(t, "estado-lote.v1.schema.json")
	var fx struct {
		Validos   []json.RawMessage `json:"validos"`
		Invalidos []casoInvalido    `json:"invalidos"`
	}
	testutil.LerFixture(t, "contratos/estado-lote.v1.json", &fx)
	for i, v := range fx.Validos {
		if err := validar(t, s, v); err != nil {
			t.Errorf("válido %d rejeitado: %v", i, err)
		}
		var e EstadoLote
		if err := decodificarEstrito(v, &e); err != nil {
			t.Errorf("válido %d não decodifica: %v", i, err)
		}
	}
	for _, c := range fx.Invalidos {
		if err := validar(t, s, c.Dado); err == nil {
			t.Errorf("inválido %q aceito", c.Motivo)
		}
	}
}

func TestInstantes(t *testing.T) {
	ms, err := LerInstante("2026-11-14T13:05:10.250Z")
	if err != nil {
		t.Fatal(err)
	}
	if got := FormatarInstante(ms); got != "2026-11-14T13:05:10.250Z" {
		t.Errorf("ida e volta: %s", got)
	}
	for _, ruim := range []string{"2026-11-14T13:05:10Z", "2026-11-14 13:05:10.250Z", "2026-11-14T13:05:10.250-04:00", "2026-13-14T13:05:10.250Z"} {
		if _, err := LerInstante(ruim); err == nil {
			t.Errorf("%q aceito", ruim)
		}
	}
}
