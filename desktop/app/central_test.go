package app

import (
	"context"
	"testing"
)

func TestCentralIniciaEPreservaID(t *testing.T) {
	dir := t.TempDir()
	c := &Central{PastaDados: dir}
	if err := c.Iniciar(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	inf := c.info()
	if !inf.ServidorAtivo || inf.CentralID == "" || len(inf.Pin) != 43 {
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
