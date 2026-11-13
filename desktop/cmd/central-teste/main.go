// central-teste sobe só o servidor de sincronização da Central, sem interface,
// para os testes cruzados do Android (mobile/core/sync). Uso exclusivo em testes.
//
// Escuta em 127.0.0.1 numa porta livre, escreve uma linha JSON {"porta":N,"pin":"..."}
// na saída padrão e encerra quando a entrada padrão é fechada.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net"
	"os"
	"time"

	"sitposse7/desktop/internal/syncserver"
)

func main() {
	dir := flag.String("dir", "", "pasta da identidade TLS (padrão: temporária, apagada ao sair)")
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

	id, err := syncserver.CarregarOuCriarIdentidade(pasta, []net.IP{net.IPv4(127, 0, 0, 1)})
	if err != nil {
		log.Fatal(err)
	}
	s := &syncserver.Servidor{
		Identidade: id,
		CentralID:  "central-teste",
		Operacao: func() *syncserver.Operacao {
			return &syncserver.Operacao{UUID: "9b2d4f61-0c3e-4a5b-8f7d-1e2c3b4a5d6e", Nome: "Operação de teste"}
		},
	}
	if err := s.Iniciar("127.0.0.1:0"); err != nil {
		log.Fatal(err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(map[string]any{"porta": s.Porta(), "pin": id.Pin()}); err != nil {
		log.Fatal(err)
	}

	_, _ = io.Copy(io.Discard, os.Stdin) // bloqueia até o processo de teste fechar a entrada
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = s.Parar(ctx)
}
