package syncserver

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"sitposse7/desktop/internal/testutil"
)

func compilarSchema(t *testing.T, nome string) *jsonschema.Schema {
	t.Helper()
	c := jsonschema.NewCompiler()
	c.AssertFormat()
	caminho := testutil.FixturePath(t, "../schemas/"+nome)
	s, err := c.Compile(caminho)
	if err != nil {
		t.Fatalf("compilando %s: %v", nome, err)
	}
	return s
}

func validarJSON(t *testing.T, s *jsonschema.Schema, dado []byte) error {
	t.Helper()
	v, err := jsonschema.UnmarshalJSON(bytes.NewReader(dado))
	if err != nil {
		t.Fatal(err)
	}
	return s.Validate(v)
}

func TestContratoQRFixtures(t *testing.T) {
	s := compilarSchema(t, "qr-pareamento.v2.schema.json")
	var fx struct {
		Validos   []json.RawMessage `json:"validos"`
		Invalidos []struct {
			Motivo string          `json:"motivo"`
			Dado   json.RawMessage `json:"dado"`
		} `json:"invalidos"`
	}
	testutil.LerFixture(t, "contratos/qr-pareamento.v2.json", &fx)
	for i, v := range fx.Validos {
		if err := validarJSON(t, s, v); err != nil {
			t.Errorf("válido %d rejeitado: %v", i, err)
		}
	}
	for _, c := range fx.Invalidos {
		if err := validarJSON(t, s, c.Dado); err == nil {
			t.Errorf("inválido %q aceito", c.Motivo)
		}
	}
}

func TestContratoSaidasReaisDoServidor(t *testing.T) {
	srv := novoServidor(t, t.TempDir())
	srv.CentralID = "3f1c9a52-6a0e-4c1b-9d55-2f7f1f0b8e21"

	qr, _ := json.Marshal(srv.QR(""))
	if err := validarJSON(t, compilarSchema(t, "qr-pareamento.v2.schema.json"), qr); err != nil {
		t.Fatalf("QR real fora do contrato: %v\n%s", err, qr)
	}

	resp, err := clienteComPin(srv.Identidade.Pin()).Get("https://127.0.0.1:" + itoa(srv.Porta()) + "/v1/status")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var buf bytes.Buffer
	buf.ReadFrom(resp.Body)
	if err := validarJSON(t, compilarSchema(t, "status.v2.schema.json"), buf.Bytes()); err != nil {
		t.Fatalf("status real fora do contrato: %v\n%s", err, buf.String())
	}
}
