// Package credenciais guarda segredos de serviço (ex.: o cliente OAuth do Copernicus,
// ADR-0022) no Gerenciador de Credenciais do Windows: nunca em arquivo de texto, no
// código ou nos logs. Fora do Windows (testes), guarda só em memória.
package credenciais

import "errors"

// ErrNaoEncontrada: não há credencial com esse nome.
var ErrNaoEncontrada = errors.New("credencial não encontrada")

// Credencial genérica: usuário (ex.: client_id) e segredo (ex.: client_secret).
type Credencial struct {
	Usuario string
	Segredo string
}

// Cofre lê, grava e apaga credenciais por nome de destino.
type Cofre interface {
	Ler(alvo string) (Credencial, error)
	Gravar(alvo string, c Credencial) error
	Apagar(alvo string) error
}

// Memoria é um cofre só em memória (testes e sistemas sem cofre nativo).
type Memoria struct{ m map[string]Credencial }

func (c *Memoria) Ler(alvo string) (Credencial, error) {
	if v, ok := c.m[alvo]; ok {
		return v, nil
	}
	return Credencial{}, ErrNaoEncontrada
}

func (c *Memoria) Gravar(alvo string, v Credencial) error {
	if c.m == nil {
		c.m = map[string]Credencial{}
	}
	c.m[alvo] = v
	return nil
}

func (c *Memoria) Apagar(alvo string) error { delete(c.m, alvo); return nil }
