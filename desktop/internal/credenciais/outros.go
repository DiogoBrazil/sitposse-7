//go:build !windows

package credenciais

var memoria = &Memoria{}

// Nativo: fora do Windows não há cofre do sistema; usa memória (testes no Linux).
func Nativo() Cofre { return memoria }
