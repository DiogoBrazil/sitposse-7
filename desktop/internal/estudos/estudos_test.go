package estudos

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"

	"sitposse7/desktop/internal/banco"
	"sitposse7/desktop/internal/codigos"
)

func novoServico(t *testing.T) *Servico {
	t.Helper()
	b, err := banco.Abrir(context.Background(), filepath.Join(t.TempDir(), "c.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { b.Fechar() })
	return &Servico{B: b}
}

func TestEstudoJornadasEEquipes(t *testing.T) {
	ctx := context.Background()
	s := novoServico(t)
	e, err := s.CriarEstudo(ctx, "  Fazenda   Santa Maria ", "2026-10-10", []string{"Equipe Alfa", "Equipe Bravo"})
	if err != nil {
		t.Fatal(err)
	}
	if e.Nome != "Fazenda Santa Maria" || !codigos.Valido(e.Codigo, codigos.PrefixoEstudo) || e.Jornadas != 1 || e.Equipes != 2 {
		t.Fatalf("estudo: %+v", e.Estudo)
	}
	j1 := e.ListaJornadas[0]
	if j1.Numero != 1 || j1.Data != "2026-10-10" || len(j1.Equipes) != 2 {
		t.Fatalf("jornada 1: %+v", j1)
	}
	// Nova jornada reaproveitando nomes: códigos novos (RN-01).
	j2, err := s.CriarJornada(ctx, e.UUID, "2026-10-11", []string{"Equipe Alfa", "Equipe Charlie"})
	if err != nil {
		t.Fatal(err)
	}
	// Duas jornadas na mesma data são permitidas.
	j3, err := s.CriarJornada(ctx, e.UUID, "2026-10-11", nil)
	if err != nil || j2.Numero != 2 || j3.Numero != 3 {
		t.Fatalf("números: %d %d %v", j2.Numero, j3.Numero, err)
	}
	codigosVistos := map[string]bool{}
	for _, q := range append(j1.Equipes, j2.Equipes...) {
		if !codigos.Valido(q.Codigo, codigos.PrefixoEquipe) || codigosVistos[q.Codigo] {
			t.Fatalf("código inválido ou repetido: %s", q.Codigo)
		}
		codigosVistos[q.Codigo] = true
	}
	if j1.Equipes[0].Nome != j2.Equipes[0].Nome || j1.Equipes[0].Codigo == j2.Equipes[0].Codigo {
		t.Fatal("Equipe Alfa da jornada 2 deveria ter outro código")
	}
	// Nome repetido na mesma jornada (comparando sem acento e maiúsculas).
	if _, err := s.AdicionarEquipes(ctx, j2.UUID, []string{"equipe  ALFA"}); !errors.Is(err, ErrNomeRepetido) {
		t.Fatalf("esperado ErrNomeRepetido, obteve %v", err)
	}
	if _, err := s.CriarEstudo(ctx, "X", "2026-02-30", nil); !errors.Is(err, ErrData) {
		t.Fatalf("data inválida aceita: %v", err)
	}
	if _, err := s.CriarEstudo(ctx, "   ", "2026-02-10", nil); !errors.Is(err, ErrNomeVazio) {
		t.Fatalf("nome vazio aceito: %v", err)
	}
	// Destino pelo código.
	d, err := DestinoPorCodigo(ctx, s.B.Leitura, j2.Equipes[1].Codigo)
	if err != nil || d == nil || d.EquipeNome != "Equipe Charlie" || d.JornadaNumero != 2 || d.JornadaData != "2026-10-11" || d.EstudoNome != "Fazenda Santa Maria" {
		t.Fatalf("destino: %+v %v", d, err)
	}
	if d, _ := DestinoPorCodigo(ctx, s.B.Leitura, "EQ0000000X"); d != nil {
		t.Fatal("código inexistente resolvido")
	}
	// Renomear e excluir.
	if err := s.RenomearEstudo(ctx, e.UUID, "Fazenda S. Maria"); err != nil {
		t.Fatal(err)
	}
	if err := s.RenomearEquipe(ctx, j2.Equipes[1].UUID, "Equipe Alfa"); !errors.Is(err, ErrNomeRepetido) {
		t.Fatalf("renomear para nome repetido: %v", err)
	}
	if err := s.ExcluirEquipe(ctx, j2.Equipes[1].UUID); err != nil {
		t.Fatal(err)
	}
	if d, _ := DestinoPorCodigo(ctx, s.B.Leitura, j2.Equipes[1].Codigo); d != nil {
		t.Fatal("equipe excluída ainda resolve pelo código")
	}
	if err := s.ExcluirJornada(ctx, j3.UUID); err != nil {
		t.Fatal(err)
	}
	// Estudo independente.
	if _, err := s.CriarEstudo(ctx, "Fazenda Santa Maria", "2026-11-01", []string{"Equipe Alfa"}); err != nil {
		t.Fatal(err)
	}
	lista, err := s.Listar(ctx)
	if err != nil || len(lista) != 2 {
		t.Fatalf("lista: %d %v", len(lista), err)
	}
	det, _ := s.Detalhe(ctx, e.UUID)
	if det.Nome != "Fazenda S. Maria" || len(det.ListaJornadas) != 2 || len(det.ListaJornadas[1].Equipes) != 1 {
		t.Fatalf("detalhe: %+v", det)
	}
	var aud int
	s.B.Leitura.QueryRow(`SELECT count(*) FROM auditoria WHERE origem = 'central'`).Scan(&aud)
	if aud < 10 {
		t.Fatalf("auditoria incompleta: %d", aud)
	}
}

// Colisão forçada: o gerador devolve primeiro um código já usado; o serviço sorteia outro.
func TestColisaoDeCodigoSorteiaOutro(t *testing.T) {
	ctx := context.Background()
	s := novoServico(t)
	repetido := bytes.Repeat([]byte{1, 2, 3, 4, 5, 6, 7}, 1)
	s.Aleatorio = bytes.NewReader(append(append([]byte{}, repetido...), repetido...)) // estudo + 1ª equipe
	e, err := s.CriarEstudo(ctx, "A", "2026-10-10", []string{"Alfa"})
	if err != nil {
		t.Fatal(err)
	}
	usado := e.ListaJornadas[0].Equipes[0].Codigo
	// Próxima equipe: 1º sorteio repete o código usado, 2º é diferente.
	s.Aleatorio = bytes.NewReader(append(append([]byte{}, repetido...), 9, 9, 9, 9, 9, 9, 9))
	q, err := s.AdicionarEquipes(ctx, e.ListaJornadas[0].UUID, []string{"Bravo"})
	if err != nil {
		t.Fatal(err)
	}
	if q[0].Codigo == usado || !codigos.Valido(q[0].Codigo, codigos.PrefixoEquipe) {
		t.Fatalf("colisão não tratada: %s", q[0].Codigo)
	}
	// Sempre o mesmo código: desiste depois de TentativasCodigo.
	s.Aleatorio = bytes.NewReader(bytes.Repeat(repetido, TentativasCodigo))
	if _, err := s.AdicionarEquipes(ctx, e.ListaJornadas[0].UUID, []string{"Charlie"}); !errors.Is(err, ErrCodigoIndisponivel) {
		t.Fatalf("esperado ErrCodigoIndisponivel, obteve %v", err)
	}
}
