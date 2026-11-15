// central-teste sobe a Central completa (banco, pareamento, importação e servidor
// HTTPS), sem interface, para os testes cruzados do Android (mobile/core/sync).
// Uso exclusivo em testes.
//
// Escuta em 127.0.0.1 numa porta livre, cria um estudo de teste (jornada 1 com as
// equipes "Alfa" e "Bravo"), gera um token de pareamento e escreve uma linha JSON
// {"porta","pin","token","central_id","pasta","equipes":[{"uuid","nome","codigo"}]}
// na saída padrão. Com -confirmar (padrão), aceita sozinho os pedidos de pareamento.
// Com -destino=auto (padrão), confirma sozinho o destino dos lotes cujo código existe
// (ou cuja coleta já tem destino); os de código inexistente ficam aguardando, como na
// Central real. Com -destino=manual, todos ficam aguardando os comandos abaixo.
// Encerra quando a entrada padrão é fechada.
//
// Comandos pela entrada padrão (uma linha cada), para os testes:
//
//	token                    → {"token":"..."} com um token novo
//	pendentes                → pedidos de pareamento esperando o operador
//	aceitar | recusar        → decide todos os pedidos de pareamento
//	aguardando               → {"aguardando":[{"lote","codigo","travado","encontrado"}]}
//	confirmar LOTE [CODIGO]  → importa o lote para a equipe do CODIGO (padrão: a do código recebido
//	                           ou a travada); {"status":...} ou {"erro":...}
//	cancelar LOTE            → cancela o lote
//	estado                   → contagens das visões vigentes e dos lotes
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"sitposse7/desktop/internal/estudos"
	"sitposse7/desktop/internal/importacao"
	"sitposse7/desktop/internal/nucleo"
)

// decidir confirma (para a equipe do código dado, ou a sugerida) ou cancela um lote aguardando.
func decidir(ctx context.Context, n *nucleo.Nucleo, lote, codigo string, cancelar bool) (string, error) {
	dec := importacao.Decisao{Cancelar: cancelar}
	if !cancelar {
		ls, err := n.Importador.Aguardando(ctx)
		if err != nil {
			return "", err
		}
		for _, a := range ls {
			if a.LoteUUID != lote {
				continue
			}
			switch {
			case codigo != "":
				d, err := estudos.DestinoPorCodigo(ctx, n.B.Leitura, codigo)
				if err != nil || d == nil {
					return "", fmt.Errorf("código %s não existe", codigo)
				}
				dec.EquipeUUID = d.EquipeUUID
			case a.Travado:
				dec.EquipeUUID = a.DestinoTravado.EquipeUUID
			case a.Destino != nil:
				dec.EquipeUUID = a.Destino.EquipeUUID
			default:
				return "", fmt.Errorf("código recebido não existe; informe o código do destino")
			}
		}
	}
	est, err := n.Importador.Decidir(ctx, lote, dec)
	if err != nil {
		return "", err
	}
	return est.Status, nil
}

func main() {
	dir := flag.String("dir", "", "pasta de dados (padrão: temporária, apagada ao sair)")
	confirmar := flag.Bool("confirmar", true, "confirmar automaticamente os pedidos de pareamento")
	espera := flag.Duration("espera", 10*time.Second, "tempo máximo aguardando o operador confirmar o pareamento")
	destino := flag.String("destino", "auto", "auto: confirma sozinho o destino dos lotes com código existente; manual: só pelos comandos")
	flag.Parse()

	pasta := *dir
	if pasta == "" {
		tmp, err := os.MkdirTemp("", "central-teste-*")
		if err != nil {
			log.Fatal(err)
		}
		defer os.RemoveAll(tmp)
		pasta = tmp
	}
	ctx := context.Background()
	n, err := nucleo.Abrir(ctx, pasta, []net.IP{net.IPv4(127, 0, 0, 1)})
	if err != nil {
		log.Fatal(err)
	}
	defer n.Fechar()
	n.Pareamento.Espera = *espera
	type equipeSaida struct {
		UUID   string `json:"uuid"`
		Nome   string `json:"nome"`
		Codigo string `json:"codigo"`
	}
	var equipes []equipeSaida
	lista, err := n.Estudos.Listar(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if len(lista) == 0 {
		if _, err := n.Estudos.CriarEstudo(ctx, "Estudo de teste", "2026-10-10", []string{"Alfa", "Bravo"}); err != nil {
			log.Fatal(err)
		}
		if lista, err = n.Estudos.Listar(ctx); err != nil {
			log.Fatal(err)
		}
	}
	det, err := n.Estudos.Detalhe(ctx, lista[0].UUID)
	if err != nil {
		log.Fatal(err)
	}
	for _, q := range det.ListaJornadas[0].Equipes {
		equipes = append(equipes, equipeSaida{q.UUID, q.Nome, q.Codigo})
	}
	token, _, err := n.Pareamento.GerarToken(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if err := n.Servidor.Iniciar("127.0.0.1:0"); err != nil {
		log.Fatal(err)
	}
	if *confirmar {
		go func() {
			for {
				for _, p := range n.Pareamento.Pendentes() {
					_ = n.Pareamento.Decidir(p.ID, true)
				}
				time.Sleep(20 * time.Millisecond)
			}
		}()
	}
	if *destino == "auto" {
		go func() {
			for {
				ls, _ := n.Importador.Aguardando(ctx)
				for _, a := range ls {
					if a.Travado || a.Destino != nil {
						_, _ = decidir(ctx, n, a.LoteUUID, "", false)
					}
				}
				time.Sleep(20 * time.Millisecond)
			}
		}()
	}
	saida := json.NewEncoder(os.Stdout)
	if err := saida.Encode(map[string]any{"porta": n.Servidor.Porta(), "pin": n.Servidor.Identidade.Pin(), "token": token,
		"central_id": n.CentralID, "pasta": pasta, "equipes": equipes}); err != nil {
		log.Fatal(err)
	}

	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() { // termina quando o processo de teste fecha a entrada
		campos := strings.Fields(sc.Text())
		if len(campos) == 0 {
			continue
		}
		switch campos[0] {
		case "aguardando":
			type item struct {
				Lote       string `json:"lote"`
				Codigo     string `json:"codigo"`
				Travado    bool   `json:"travado"`
				Encontrado bool   `json:"encontrado"`
			}
			out := []item{}
			ls, _ := n.Importador.Aguardando(ctx)
			for _, a := range ls {
				out = append(out, item{a.LoteUUID, a.CodigoInformado, a.Travado, a.Destino != nil})
			}
			_ = saida.Encode(map[string]any{"aguardando": out})
		case "confirmar", "cancelar":
			if len(campos) < 2 {
				_ = saida.Encode(map[string]string{"erro": "informe o lote"})
				continue
			}
			codigo := ""
			if len(campos) > 2 {
				codigo = campos[2]
			}
			st, err := decidir(ctx, n, campos[1], codigo, campos[0] == "cancelar")
			if err != nil {
				_ = saida.Encode(map[string]string{"erro": err.Error()})
				continue
			}
			_ = saida.Encode(map[string]string{"status": st})
		case "token":
			t, _, err := n.Pareamento.GerarToken(ctx)
			if err != nil {
				log.Fatal(err)
			}
			_ = saida.Encode(map[string]string{"token": t})
		case "pendentes": // pedidos de pareamento esperando o operador
			_ = saida.Encode(map[string]int{"pendentes": len(n.Pareamento.Pendentes())})
		case "aceitar", "recusar": // decide todos os pedidos pendentes, como o operador
			d := 0
			for _, p := range n.Pareamento.Pendentes() {
				if n.Pareamento.Decidir(p.ID, campos[0] == "aceitar") == nil {
					d++
				}
			}
			_ = saida.Encode(map[string]int{"decididos": d})
		case "estado":
			est := map[string]int{}
			for nome, q := range map[string]string{
				"propriedades": "SELECT count(*) FROM v_propriedades_vigentes",
				"itens":        "SELECT count(*) FROM v_itens_vigentes",
				"fotos":        "SELECT count(*) FROM v_fotos_vigentes",
				"lotes":        "SELECT count(*) FROM lotes_recebidos WHERE status = 'importado'",
				"aguardando":   "SELECT count(*) FROM lotes_recebidos WHERE status = 'aguardando_confirmacao'",
				"cancelados":   "SELECT count(*) FROM lotes_recebidos WHERE status = 'cancelado'",
				"coletas":      "SELECT count(*) FROM coletas WHERE equipe_uuid IS NOT NULL",
				"aplicacoes":   "SELECT count(*) FROM aplicacoes",
			} {
				var v int
				_ = n.B.Leitura.QueryRow(q).Scan(&v)
				est[nome] = v
			}
			_ = saida.Encode(est)
		}
	}
}
