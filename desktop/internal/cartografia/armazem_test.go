package cartografia

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"sitposse7/desktop/internal/cartografia/geo"
)

func TestArmazemImagensECartas(t *testing.T) {
	a := Armazem{Dir: t.TempDir()}
	d := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	ped := PedidoImagem{Ret: geo.Retangulo{MinX: -1, MinY: -2, MaxX: 3, MaxY: 4}, Largura: 40, Altura: 30, Data: &d}
	k := ChaveImagem("cdse", "v1", ped)
	if k != ChaveImagem("cdse", "v1", ped) || k == ChaveImagem("cdse", "v2", ped) {
		t.Fatal("chave deve ser determinística e mudar com o script")
	}
	if _, ok := a.LerImagem(k); ok {
		t.Fatal("imagem inexistente encontrada")
	}
	img := &Imagem{PNG: []byte("png"), Ret: ped.Ret, Largura: 40, Altura: 30, Data: &d, Fonte: "x"}
	if err := a.GravarImagem(k, img); err != nil {
		t.Fatal(err)
	}
	got, ok := a.LerImagem(k)
	if !ok || string(got.PNG) != "png" || got.Data == nil || !got.Data.Equal(d) {
		t.Fatalf("lido %+v", got)
	}
	// Reinício do aplicativo: um novo Armazem na mesma pasta encontra tudo.
	c, err := a.GravarCarta(CartaGuardada{Titulo: "Carta", Tipo: "geral", GeradaEm: time.Now()}, map[string][]byte{"carta.pdf": []byte("%PDF"), "carta.png": []byte("png")})
	if err != nil {
		t.Fatal(err)
	}
	b := Armazem{Dir: a.Dir}
	if l := b.ListarCartas(); len(l) != 1 || l[0].ID != c.ID || len(l[0].Arquivos) != 2 {
		t.Fatalf("cartas = %+v", l)
	}
	if _, err := b.CaminhoCarta(c.ID, "carta.pdf"); err != nil {
		t.Fatal(err)
	}
	// Nomes fora do padrão são recusados (a rota da prévia não serve arquivos arbitrários).
	for _, id := range []string{"../x", c.ID + "/..", ""} {
		if _, err := b.CaminhoCarta(id, "carta.pdf"); err == nil {
			t.Errorf("aceitou id %q", id)
		}
	}
	if _, err := b.CaminhoCarta(c.ID, "../../central.db"); err == nil {
		t.Error("aceitou arquivo fora da carta")
	}
	if _, ok := b.CaminhoImagem("../central"); ok {
		t.Error("aceitou chave inválida")
	}
	u := b.Uso()
	if u.Imagens != 1 || u.Cartas != 1 || u.BytesImagens == 0 {
		t.Errorf("uso = %+v", u)
	}
	// Limpar apaga só as imagens; as cartas ficam.
	n, err := b.LimparImagens()
	if err != nil || n != 1 {
		t.Fatalf("limpar = %d, %v", n, err)
	}
	if _, ok := b.LerImagem(k); ok {
		t.Error("imagem continua após limpar")
	}
	if len(b.ListarCartas()) != 1 {
		t.Error("limpar imagens apagou carta")
	}
	// Carta interrompida (sem carta.json) não aparece na lista.
	_ = os.MkdirAll(filepath.Join(a.Dir, "cartas", "20260101-000000-deadbeef"), 0o755)
	if len(b.ListarCartas()) != 1 {
		t.Error("carta incompleta listada")
	}
}
