package syncserver

import "testing"

// A porta padrão precisa ficar fora das faixas que o Windows costuma reservar:
// portas dinâmicas e exclusões do Hyper-V (>= 49152) e a faixa efêmera do Linux
// que a rede espelhada do WSL reserva no Windows (44620–48715). Achado da Fase 0:
// a porta anterior (47807) ficava inacessível em notebooks com WSL espelhado.
func TestPortaPadraoForaDeFaixasReservadas(t *testing.T) {
	if PortaPadrao < 1024 {
		t.Fatalf("porta %d exige privilégio", PortaPadrao)
	}
	if PortaPadrao >= 49152 {
		t.Fatalf("porta %d na faixa dinâmica do Windows", PortaPadrao)
	}
	if PortaPadrao >= 44620 && PortaPadrao <= 48715 {
		t.Fatalf("porta %d na faixa reservada pelo WSL espelhado", PortaPadrao)
	}
}
