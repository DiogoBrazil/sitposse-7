package banco

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestAbrirAplicaPragmasEMigracoes(t *testing.T) {
	ctx := context.Background()
	caminho := filepath.Join(t.TempDir(), "central.db")
	b, err := Abrir(ctx, caminho)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerificarPragmas(ctx, b.Escrita); err != nil {
		t.Fatal(err)
	}
	if err := VerificarPragmas(ctx, b.Leitura); err != nil {
		t.Fatal(err)
	}
	var v int
	if err := b.Escrita.QueryRow("PRAGMA user_version").Scan(&v); err != nil || v != VersaoMaxima() || v < 1 {
		t.Fatalf("user_version=%d err=%v", v, err)
	}
	if _, err := b.Escrita.Exec(`INSERT INTO meta VALUES ('k','v',1,1)`); err != nil {
		t.Fatal(err)
	}
	// Leitura não escreve.
	if _, err := b.Leitura.Exec(`INSERT INTO meta VALUES ('k2','v',1,1)`); err == nil {
		t.Fatal("conexão de leitura aceitou escrita")
	}
	b.Fechar()

	// Reabrir é idempotente e preserva dados.
	b, err = Abrir(ctx, caminho)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Fechar()
	var n int
	b.Leitura.QueryRow(`SELECT count(*) FROM meta WHERE chave = 'k'`).Scan(&n)
	if n != 1 {
		t.Fatalf("dados perdidos após reabrir: %d", n)
	}
}

func TestRecusaBancoMaisNovo(t *testing.T) {
	ctx := context.Background()
	caminho := filepath.Join(t.TempDir(), "central.db")
	b, err := Abrir(ctx, caminho)
	if err != nil {
		t.Fatal(err)
	}
	b.Escrita.Exec("PRAGMA user_version = 999")
	b.Fechar()
	_, err = Abrir(ctx, caminho)
	if err == nil || !strings.Contains(err.Error(), "mais novo") {
		t.Fatalf("esperado erro de versão mais nova, obteve %v", err)
	}
}

func TestStrictRejeitaTipoErrado(t *testing.T) {
	b, err := Abrir(context.Background(), filepath.Join(t.TempDir(), "c.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer b.Fechar()
	if _, err := b.Escrita.Exec(`INSERT INTO meta VALUES ('k','v','não é inteiro',1)`); err == nil {
		t.Fatal("STRICT não rejeitou texto em coluna INTEGER")
	}
}

// Banco criado pela versão da Fase 0 (só a migração 0001, com dados) migra para a
// versão atual sem perder nada (AGENTS.md §3/§4: migração não destrutiva testada).
func TestMigraBancoDaFase0SemPerderDados(t *testing.T) {
	ctx := context.Background()
	caminho := filepath.Join(t.TempDir(), "fase0.db")
	db, err := abrir(caminho, false)
	if err != nil {
		t.Fatal(err)
	}
	m1, err := migracoesFS.ReadFile("migracoes/0001_meta.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{string(m1), "PRAGMA user_version = 1",
		`INSERT INTO meta VALUES ('central_id','3f1c9a52-6a0e-4c1b-9d55-2f7f1f0b8e21',1,1)`} {
		if _, err := db.ExecContext(ctx, s); err != nil {
			t.Fatal(err)
		}
	}
	db.Close()

	b, err := Abrir(ctx, caminho)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Fechar()
	var id string
	if err := b.Leitura.QueryRow(`SELECT valor FROM meta WHERE chave='central_id'`).Scan(&id); err != nil || id != "3f1c9a52-6a0e-4c1b-9d55-2f7f1f0b8e21" {
		t.Fatalf("central_id perdido na migração: %q %v", id, err)
	}
	var v, n int
	b.Leitura.QueryRow("PRAGMA user_version").Scan(&v)
	b.Leitura.QueryRow("SELECT count(*) FROM catalogo_itens").Scan(&n)
	if v != VersaoMaxima() || n == 0 {
		t.Fatalf("user_version=%d catálogo=%d", v, n)
	}
}

func TestAuditoriaSomenteAcrescimo(t *testing.T) {
	b, err := Abrir(context.Background(), filepath.Join(t.TempDir(), "c.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer b.Fechar()
	if _, err := b.Escrita.Exec(`INSERT INTO auditoria (quando, origem, entidade, entidade_uuid, acao) VALUES (1,'central','x','u','teste')`); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Escrita.Exec(`UPDATE auditoria SET acao='outra'`); err == nil {
		t.Fatal("UPDATE em auditoria aceito")
	}
	if _, err := b.Escrita.Exec(`DELETE FROM auditoria`); err == nil {
		t.Fatal("DELETE em auditoria aceito")
	}
}

// Banco da geração anterior (MVP, migração 0002 antiga): a Central recusa abrir e
// não altera nada no arquivo (ADR-0025).
func TestRecusaBancoDaGeracaoAnteriorSemAlterar(t *testing.T) {
	ctx := context.Background()
	caminho := filepath.Join(t.TempDir(), "mvp.db")
	db, err := abrir(caminho, false)
	if err != nil {
		t.Fatal(err)
	}
	m1, _ := migracoesFS.ReadFile("migracoes/0001_meta.sql")
	for _, s := range []string{string(m1),
		`CREATE TABLE operacoes (uuid TEXT PRIMARY KEY, nome TEXT NOT NULL) STRICT`,
		`INSERT INTO operacoes VALUES ('op-1', 'Operação de teste')`,
		`INSERT INTO meta VALUES ('central_id','3f1c9a52-6a0e-4c1b-9d55-2f7f1f0b8e21',1,1)`,
		"PRAGMA user_version = 2"} {
		if _, err := db.ExecContext(ctx, s); err != nil {
			t.Fatal(err)
		}
	}
	db.Close()

	if _, err := Abrir(ctx, caminho); !errors.Is(err, ErrBancoGeracaoAnterior) {
		t.Fatalf("esperado ErrBancoGeracaoAnterior, obteve %v", err)
	}
	db, _ = abrir(caminho, true)
	defer db.Close()
	var v, ops, tabelas int
	db.QueryRow("PRAGMA user_version").Scan(&v)
	db.QueryRow("SELECT count(*) FROM operacoes").Scan(&ops)
	db.QueryRow("SELECT count(*) FROM sqlite_master WHERE name = 'estudos'").Scan(&tabelas)
	if v != 2 || ops != 1 || tabelas != 0 {
		t.Fatalf("banco antigo alterado: user_version=%d operações=%d estudos=%d", v, ops, tabelas)
	}
}

func TestGeracaoGravadaEDestinoTravado(t *testing.T) {
	b, err := Abrir(context.Background(), filepath.Join(t.TempDir(), "c.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer b.Fechar()
	var g string
	if err := b.Leitura.QueryRow(`SELECT valor FROM meta WHERE chave = 'esquema_geracao'`).Scan(&g); err != nil || g != GeracaoEsquema {
		t.Fatalf("geração = %q, %v", g, err)
	}
	for _, s := range []string{
		`INSERT INTO estudos VALUES ('e1', 'SITPOSSE7A82F91C6', 'Fazenda', 1, 1, NULL)`,
		`INSERT INTO jornadas VALUES ('j1', 'e1', 1, '2026-10-10', 1, 1, NULL)`,
		`INSERT INTO equipes VALUES ('q1', 'j1', 'Alfa', 'alfa', 'EQ7K3M9QX1', 1, 1, NULL)`,
		`INSERT INTO equipes VALUES ('q2', 'j1', 'Bravo', 'bravo', 'EQ4HZ2RTB5', 1, 1, NULL)`,
		`INSERT INTO dispositivos VALUES ('i1', 'h', NULL, 1, NULL)`,
		`INSERT INTO coletas (uuid, instalacao_uuid, codigo_informado, codigo_original, created_at, updated_at, recebido_em) VALUES ('c1', 'i1', 'EQ7K3M9QX1', 'EQ7K3M9QX1', 1, 1, 1)`,
		`UPDATE coletas SET equipe_uuid = 'q1', destino_confirmado_em = 2 WHERE uuid = 'c1'`,
	} {
		if _, err := b.Escrita.Exec(s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}
	if _, err := b.Escrita.Exec(`UPDATE coletas SET equipe_uuid = 'q2' WHERE uuid = 'c1'`); err == nil {
		t.Fatal("destino travado alterado")
	}
	if _, err := b.Escrita.Exec(`INSERT INTO equipes VALUES ('q3', 'j1', 'ALFA ', 'alfa', 'EQ0000000X', 1, 1, NULL)`); err == nil {
		t.Fatal("nome repetido na mesma jornada aceito")
	}
}

// Migração 0003 sobre um banco da geração 2 com dados: numera os cadastros existentes por
// estudo, em ordem de recebimento (inclusive excluídos), sem alterar nada além disso (RN-96).
func TestMigracao0003NumeraCadastrosExistentes(t *testing.T) {
	ctx := context.Background()
	caminho := filepath.Join(t.TempDir(), "g2.db")
	db, err := abrir(caminho, false)
	if err != nil {
		t.Fatal(err)
	}
	var cmds []string
	for _, n := range []string{"0001_meta.sql", "0002_estudos_e_recepcao.sql"} {
		m, err := migracoesFS.ReadFile("migracoes/" + n)
		if err != nil {
			t.Fatal(err)
		}
		cmds = append(cmds, string(m))
	}
	cmds = append(cmds, "PRAGMA user_version = 2",
		`INSERT INTO estudos (uuid, codigo, nome, created_at, updated_at) VALUES ('e1','SITPOSSE7A82F91C6','Estudo A',1,1), ('e2','SITPOSSE7B82F91C6','Estudo B',1,1)`,
		`INSERT INTO jornadas (uuid, estudo_uuid, numero, data, created_at, updated_at) VALUES ('j1','e1',1,'2026-10-10',1,1), ('j2','e2',1,'2026-10-10',1,1)`,
		`INSERT INTO equipes (uuid, jornada_uuid, nome, nome_normalizado, codigo, created_at, updated_at) VALUES ('q1','j1','Alfa','alfa','EQ7K3M9QX1',1,1), ('q2','j2','Bravo','bravo','EQ4HZ2RTB5',1,1)`,
		`INSERT INTO dispositivos (instalacao_uuid, credencial_hash, pareado_em) VALUES ('i1','h',1)`,
		`INSERT INTO coletas (uuid, instalacao_uuid, codigo_informado, codigo_original, equipe_uuid, destino_confirmado_em, created_at, updated_at, recebido_em) VALUES ('c1','i1','EQ7K3M9QX1','EQ7K3M9QX1','q1',1,1,1,1)`)
	ins := `INSERT INTO propriedades (uuid, coleta_uuid, equipe_uuid, nome_propriedade, responsavel_familia, created_at, updated_at, rev_origem, recebido_em, origem_instalacao, registro_json, excluido_em) VALUES `
	cmds = append(cmds, ins+`('pc','c1','q1','C','R',1,1,1,30,'i1','{}',NULL), ('pa','c1','q1','A','R',1,1,1,10,'i1','{}',NULL), ('pb','c1','q1','B','R',1,1,1,20,'i1','{}',99), ('px','c1','q2','X','R',1,1,1,5,'i1','{}',NULL)`)
	for _, s := range cmds {
		if _, err := db.ExecContext(ctx, s); err != nil {
			t.Fatal(s[:min(len(s), 80)], err)
		}
	}
	db.Close()

	b, err := Abrir(ctx, caminho)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Fechar()
	esperado := map[string]int{"pa": 1, "pb": 2, "pc": 3, "px": 1}
	for u, n := range esperado {
		var got int
		if err := b.Leitura.QueryRow(`SELECT numero FROM numeracao_propriedades WHERE propriedade_uuid = ?`, u).Scan(&got); err != nil || got != n {
			t.Fatalf("%s: número %d (esperado %d) %v", u, got, n, err)
		}
	}
	var vig, num int
	b.Leitura.QueryRow(`SELECT count(*), sum(numero_no_estudo) FROM v_propriedades_vigentes`).Scan(&vig, &num)
	if vig != 3 || num != 1+3+1 {
		t.Fatalf("visão: %d vigentes, soma dos números %d", vig, num)
	}
}
