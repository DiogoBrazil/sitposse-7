package app

import (
	"context"
	"net/http/httptest"
	"testing"
)

func TestCentralIniciaEPreservaID(t *testing.T) {
	dir := t.TempDir()
	c := &Central{PastaDados: dir}
	if err := c.Iniciar(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	inf := c.info()
	if !inf.ServidorAtivo || inf.CentralID == "" || len(inf.Pin) != 43 || inf.VersaoEsquema < 2 {
		t.Fatalf("info inesperada: %+v", inf)
	}
	c.Encerrar()

	c2 := &Central{PastaDados: dir}
	if err := c2.Iniciar(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	defer c2.Encerrar()
	if c2.info().CentralID != inf.CentralID || c2.info().Pin != inf.Pin {
		t.Fatal("central_id ou pin mudou ao reiniciar")
	}
}

func TestQRExigeOperacaoETrazToken(t *testing.T) {
	c := &Central{PastaDados: t.TempDir()}
	if err := c.Iniciar(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	defer c.Encerrar()
	sinc := &SincronizacaoService{C: c}
	if _, err := sinc.GerarQRPareamento(); err == nil {
		t.Fatal("QR gerado sem operação ativa")
	}
	if _, err := (&OperacaoService{C: c}).Criar("Operação Teste", ""); err != nil {
		t.Fatal(err)
	}
	qr, err := sinc.GerarQRPareamento()
	if err != nil || len(qr.Payload.Token) < 32 || qr.Payload.OperacaoNome != "Operação Teste" {
		t.Fatalf("QR: %v %+v", err, qr)
	}
}

func TestHandlerFotosNaoServeForaDaPasta(t *testing.T) {
	c := &Central{PastaDados: t.TempDir()}
	if err := c.Iniciar(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	defer c.Encerrar()
	for _, url := range []string{"/fotos/../sitposse7.db", "/fotos/3e9d4f5b-6a7c-4d8e-9f9a-0b1c2d3e4f5a.jpg", "/outra"} {
		w := httptest.NewRecorder()
		c.HandlerFotos().ServeHTTP(w, httptest.NewRequest("GET", url, nil))
		if w.Code != 404 {
			t.Errorf("%s: %d", url, w.Code)
		}
	}
}
