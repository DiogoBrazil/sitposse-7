// central-teste sobe a Central completa (banco, pareamento, importação e servidor
// HTTPS), sem interface, para os testes cruzados do Android (mobile/core/sync).
// Uso exclusivo em testes.
//
// Escuta em 127.0.0.1 numa porta livre, cria uma operação, gera um token de
// pareamento e escreve uma linha JSON {"porta","pin","token","central_id","pasta"}
// na saída padrão. Com -confirmar (padrão), aceita sozinho os pedidos de pareamento,
// como se o operador clicasse em "Confirmar". Encerra quando a entrada padrão é fechada.
//
// Comandos pela entrada padrão (uma linha cada), para os testes:
//
//	token  → escreve {"token":"..."} com um token novo
//	estado → escreve contagens das visões vigentes
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"log"
	"net"
	"os"
	"time"

	"sitposse7/desktop/internal/nucleo"
)

func main() {
	dir := flag.String("dir", "", "pasta de dados (padrão: temporária, apagada ao sair)")
	confirmar := flag.Bool("confirmar", true, "confirmar automaticamente os pedidos de pareamento")
	espera := flag.Duration("espera", 10*time.Second, "tempo máximo aguardando o operador confirmar o pareamento")
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
	if op, err := n.Pareamento.OperacaoAtiva(ctx); err != nil {
		log.Fatal(err)
	} else if op == nil {
		if _, err := n.Pareamento.CriarOperacao(ctx, "Operação de teste", ""); err != nil {
			log.Fatal(err)
		}
	}
	token, _, _, err := n.Pareamento.GerarToken(ctx)
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
	saida := json.NewEncoder(os.Stdout)
	if err := saida.Encode(map[string]any{"porta": n.Servidor.Porta(), "pin": n.Servidor.Identidade.Pin(), "token": token, "central_id": n.CentralID, "pasta": pasta}); err != nil {
		log.Fatal(err)
	}

	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() { // termina quando o processo de teste fecha a entrada
		switch sc.Text() {
		case "token":
			t, _, _, err := n.Pareamento.GerarToken(ctx)
			if err != nil {
				log.Fatal(err)
			}
			_ = saida.Encode(map[string]string{"token": t})
		case "pendentes": // pedidos de pareamento esperando o operador
			_ = saida.Encode(map[string]int{"pendentes": len(n.Pareamento.Pendentes())})
		case "aceitar", "recusar": // decide todos os pedidos pendentes, como o operador
			d := 0
			for _, p := range n.Pareamento.Pendentes() {
				if n.Pareamento.Decidir(p.ID, sc.Text() == "aceitar") == nil {
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
