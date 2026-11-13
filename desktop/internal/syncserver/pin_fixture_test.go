package syncserver

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"

	"sitposse7/desktop/internal/testutil"
)

// O pin calculado pela Central deve coincidir com o da fixture (gerada pelo OpenSSL)
// e com o calculado pelo Android (mobile/core/sync, PinTest).
func TestPinFixtureTLS(t *testing.T) {
	var fx struct {
		Casos []struct {
			Certificado string `json:"certificado"`
			Pin         string `json:"pin"`
		} `json:"casos"`
	}
	testutil.LerFixture(t, "tls/pin.json", &fx)
	if len(fx.Casos) == 0 {
		t.Fatal("fixture sem casos")
	}
	for _, c := range fx.Casos {
		b, err := os.ReadFile(testutil.FixturePath(t, "tls/"+c.Certificado))
		if err != nil {
			t.Fatal(err)
		}
		blk, _ := pem.Decode(b)
		if blk == nil {
			t.Fatalf("%s: PEM inválido", c.Certificado)
		}
		cert, err := x509.ParseCertificate(blk.Bytes)
		if err != nil {
			t.Fatal(err)
		}
		if got := PinDaChavePublica(cert.PublicKey); got != c.Pin {
			t.Errorf("%s: pin %s, esperado %s", c.Certificado, got, c.Pin)
		}
	}
}
