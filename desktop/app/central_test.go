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

func TestQRTrazTokenSemOperacao(t *testing.T) {
	c := &Central{PastaDados: t.TempDir()}
	if err := c.Iniciar(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	defer c.Encerrar()
	qr, err := (&SincronizacaoService{C: c}).GerarQRPareamento()
	if err != nil || len(qr.Payload.Token) < 32 || qr.Payload.V != 2 {
		t.Fatalf("QR: %v %+v", err, qr)
	}
}

func TestEstudoEmFoco(t *testing.T) {
	c := &Central{PastaDados: t.TempDir()}
	if err := c.Iniciar(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	defer c.Encerrar()
	s := &EstudoService{C: c}
	if f, err := s.EmFoco(); err != nil || f != "" {
		t.Fatalf("sem estudos: %q %v", f, err)
	}
	a, _ := s.Criar("A", "2026-10-10", nil)
	b, _ := s.Criar("B", "2026-10-11", []string{"Alfa"})
	if f, _ := s.EmFoco(); f != b.UUID {
		t.Fatalf("o estudo criado por último deveria estar em foco: %q", f)
	}
	if err := s.DefinirEmFoco(a.UUID); err != nil {
		t.Fatal(err)
	}
	if f, _ := s.EmFoco(); f != a.UUID {
		t.Fatalf("em foco: %q", f)
	}
}

// Banco da geração anterior: a Central sobe só com a mensagem de bloqueio (ADR-0025).
func TestBancoDaGeracaoAnteriorBloqueia(t *testing.T) {
	dir := t.TempDir()
	c := &Central{PastaDados: dir}
	if err := c.Iniciar(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	c.n.B.Escrita.Exec(`DELETE FROM meta WHERE chave = 'esquema_geracao'`)
	c.Encerrar()
	c2 := &Central{PastaDados: dir}
	if err := c2.Iniciar(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	if inf := (&SincronizacaoService{C: c2}).Info(); inf.Bloqueio == "" || inf.ServidorAtivo {
		t.Fatalf("esperado bloqueio: %+v", inf)
	}
	if ls, err := (&SincronizacaoService{C: c2}).Aguardando(); err != nil || len(ls) != 0 {
		t.Fatal("aguardando com banco bloqueado")
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
