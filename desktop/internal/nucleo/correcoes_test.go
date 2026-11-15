package nucleo

import (
	"context"

	"github.com/google/uuid"
	"os"
	"path/filepath"
	"sitposse7/desktop/internal/contratos"
	"sitposse7/desktop/internal/correcoes"
	"sitposse7/desktop/internal/importacao"
	"testing"
)

func TestCorrecoesConcorrenciaRecusaECascata(t *testing.T) {
	a := novoAmb(t)
	a.parear()
	ctx := context.Background()
	p := a.prop("Original", 1)
	it := contratos.Item{UUID: uuid.NewString(), PropriedadeUUID: p.UUID, Categoria: "eletrodomestico", CodigoCatalogo: "eletro.geladeira", Quantidade: 1, Rev: 1, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt}
	l := a.novoLote([]contratos.Propriedade{p}, []contratos.Item{it}, nil)
	a.putLote(l)
	a.importarConfirmado(l.LoteUUID)
	srv := &correcoes.Servico{B: a.n.B}
	d, e := srv.Carregar(ctx, p.UUID)
	if e != nil {
		t.Fatal(e)
	}
	old := d.Token
	d.Propriedade["nome_propriedade"] = "Corrigido"
	d.Itens[0]["quantidade"] = float64(7)
	if e = srv.Salvar(ctx, p.UUID, d); e != nil {
		t.Fatal(e)
	}
	if e = srv.Salvar(ctx, p.UUID, d); e == nil {
		t.Fatal("aceitou token desatualizado")
	}
	d, e = srv.Carregar(ctx, p.UUID)
	if e != nil || d.Token == old {
		t.Fatal(e, "token não mudou")
	}
	if d.Propriedade["nome_propriedade"] != "Corrigido" {
		t.Fatal(d)
	}
	// Erro em item desfaz inclusive a edição da propriedade.
	d.Propriedade["nome_propriedade"] = "Não deve gravar"
	d.Itens[0]["quantidade"] = -1
	if e = srv.Salvar(ctx, p.UUID, d); e == nil {
		t.Fatal("quantidade negativa aceita")
	}
	d, _ = srv.Carregar(ctx, p.UUID)
	if d.Propriedade["nome_propriedade"] != "Corrigido" {
		t.Fatal("sem rollback")
	}
	it.Rev = 2
	it.Quantidade = 99
	l2 := a.novoLote(nil, []contratos.Item{it}, nil)
	a.putLote(l2)
	r := a.importarConfirmado(l2.LoteUUID)
	for _, x := range r.Recibo.Resultados {
		if x.Entidade == "item" && x.Resultado != "rejeitado" {
			t.Fatal(x)
		}
	}
	if a.contar(`SELECT quantidade FROM itens_propriedade WHERE uuid=?`, it.UUID) != 7 {
		t.Fatal("sobrescrita parcial")
	}
	d, _ = srv.Carregar(ctx, p.UUID)
	if e = srv.Excluir(ctx, p.UUID, d.Token); e != nil {
		t.Fatal(e)
	}
	if a.contar(`SELECT count(*) FROM v_propriedades_vigentes`) != 0 || a.contar(`SELECT count(*) FROM v_itens_vigentes`) != 0 {
		t.Fatal("cascata incompleta")
	}
	if a.contar(`SELECT count(*) FROM propriedades`) != 1 {
		t.Fatal("exclusão física")
	}
	if a.getLote(l.LoteUUID).Recibo == nil {
		t.Fatal("recibo perdido")
	}
	if a.contar(`SELECT count(*) FROM auditoria WHERE origem='central'`) < 4 {
		t.Fatal("falta auditoria")
	}
}
func TestExcluirEstudoCancelaPendentesPreservaOutros(t *testing.T) {
	a := novoAmb(t)
	a.parear()
	ctx := context.Background()
	p := a.prop("Um", 1)
	l := a.novoLote([]contratos.Propriedade{p}, nil, nil)
	a.putLote(l)
	a.importarConfirmado(l.LoteUUID)
	outro, e := a.n.Estudos.CriarEstudo(ctx, "Outro", "2026-11-15", nil)
	if e != nil {
		t.Fatal(e)
	}
	p.Rev = 2
	pend := a.novoLote([]contratos.Propriedade{p}, nil, nil)
	a.putLote(pend)
	a.importar(pend.LoteUUID)
	if e = a.n.Importador.ExcluirEstudo(ctx, a.estudo.UUID); e != nil {
		t.Fatal(e)
	}
	if a.getLote(pend.LoteUUID).Status != "cancelado" {
		t.Fatal("pendente não cancelado")
	}
	if a.getLote(l.LoteUUID).Recibo == nil {
		t.Fatal("recibo perdido")
	}
	if a.contar(`SELECT count(*) FROM estudos WHERE excluido_em IS NULL`) != 1 {
		t.Fatal("afetou outro estudo")
	}
	if a.contar(`SELECT count(*) FROM estudos WHERE uuid=? AND excluido_em IS NULL`, outro.UUID) != 1 {
		t.Fatal("outro estudo")
	}
	if a.contar(`SELECT count(*) FROM v_propriedades_vigentes`) != 0 {
		t.Fatal("cadastro ainda vigente")
	}
	if _, e = a.n.Importador.Decidir(ctx, pend.LoteUUID, importacao.Decisao{EquipeUUID: a.alfa.UUID}); e == nil {
		t.Fatal("reabriu cancelado")
	}
}

func TestEditarItensEFotosSemPerderArquivo(t *testing.T) {
	a := novoAmb(t)
	a.parear()
	ctx := context.Background()
	p := a.prop("Com foto", 1)
	bytes, hash := fotoAleatoria(t, 1000)
	f := a.foto(p.UUID, 1000, hash)
	l := a.novoLote([]contratos.Propriedade{p}, nil, []contratos.Foto{f})
	a.putLote(l)
	a.enviarFoto(f.UUID, bytes, 0)
	a.importarConfirmado(l.LoteUUID)
	srv := &correcoes.Servico{B: a.n.B}
	d, e := srv.Carregar(ctx, p.UUID)
	if e != nil {
		t.Fatal(e)
	}
	d.Fotos = nil
	d.Itens = []map[string]any{{"codigo_catalogo": "ave.outros", "nome_personalizado": "Codorna", "quantidade": float64(3)}}
	if e = srv.Salvar(ctx, p.UUID, d); e != nil {
		t.Fatal(e)
	}
	if a.contar(`SELECT count(*) FROM v_fotos_vigentes`) != 0 || a.contar(`SELECT count(*) FROM v_itens_vigentes`) != 1 {
		t.Fatal("itens/fotos")
	}
	if _, e = os.Stat(filepath.Join(a.pasta, importacao.CaminhoRelativoFoto(f.UUID))); e != nil {
		t.Fatal("apagou arquivo", e)
	}
	d, _ = srv.Carregar(ctx, p.UUID)
	if d.Itens[0]["origem_instalacao"] != "central" {
		t.Fatal("origem incorreta")
	}
	// Falha no meio da cascata deve desfazer tudo.
	if _, e = a.n.B.Escrita.Exec(`CREATE TRIGGER teste_falha BEFORE UPDATE ON propriedades BEGIN SELECT RAISE(ABORT,'falha simulada'); END`); e != nil {
		t.Fatal(e)
	}
	if e = a.n.Importador.ExcluirEstudo(ctx, a.estudo.UUID); e == nil {
		t.Fatal("falha não propagou")
	}
	if a.contar(`SELECT count(*) FROM estudos WHERE excluido_em IS NULL`) != 1 || a.contar(`SELECT count(*) FROM v_itens_vigentes`) != 1 {
		t.Fatal("cascata parcial")
	}
}
