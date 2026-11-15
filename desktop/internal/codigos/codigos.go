// Package codigos gera, normaliza e confere os códigos públicos de estudo e de equipe
// (RN-02, ADR-0023): alfabeto Crockford Base32, 7 caracteres aleatórios e 1 dígito
// verificador Luhn mod 32. Os casos de referência estão em
// shared/test-fixtures/codigos/codigos.v1.json (os mesmos do app Campo).
package codigos

import (
	"crypto/rand"
	"errors"
	"io"
	"strings"
)

// Alfabeto Crockford Base32 (sem I, L, O e U).
const Alfabeto = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// Prefixos dos códigos.
const (
	PrefixoEquipe = "EQ"
	PrefixoEstudo = "SITPOSSE7"
)

// TamanhoCorpo é o número de caracteres depois do prefixo (7 aleatórios + dígito).
const TamanhoCorpo = 8

// Erros de normalização (a mensagem é para o usuário).
var (
	ErrVazio     = errors.New("informe o código")
	ErrTamanho   = errors.New("o código tem 8 caracteres depois do prefixo")
	ErrCaractere = errors.New("o código tem um caractere que não existe nos códigos")
	ErrDigito    = errors.New("confira os caracteres do código")
)

func indice(c byte) int { return strings.IndexByte(Alfabeto, c) }

// DigitoVerificador calcula o dígito Luhn mod 32 de um corpo de 7 caracteres do alfabeto.
func DigitoVerificador(corpo string) (byte, error) {
	fator, soma := 2, 0
	for i := len(corpo) - 1; i >= 0; i-- {
		v := indice(corpo[i])
		if v < 0 {
			return 0, ErrCaractere
		}
		a := fator * v
		if fator == 2 {
			fator = 1
		} else {
			fator = 2
		}
		soma += a/32 + a%32
	}
	return Alfabeto[(32-soma%32)%32], nil
}

// digitoConfere valida um corpo completo (com o dígito no fim).
func digitoConfere(corpo string) bool {
	fator, soma := 1, 0
	for i := len(corpo) - 1; i >= 0; i-- {
		v := indice(corpo[i])
		if v < 0 {
			return false
		}
		a := fator * v
		if fator == 1 {
			fator = 2
		} else {
			fator = 1
		}
		soma += a/32 + a%32
	}
	return soma%32 == 0
}

// Gerar cria um código novo com o prefixo dado, usando a fonte aleatória (nil = crypto/rand).
func Gerar(prefixo string, aleatorio io.Reader) (string, error) {
	if aleatorio == nil {
		aleatorio = rand.Reader
	}
	var b [7]byte
	if _, err := io.ReadFull(aleatorio, b[:]); err != nil {
		return "", err
	}
	corpo := make([]byte, 0, TamanhoCorpo)
	for _, x := range b {
		corpo = append(corpo, Alfabeto[int(x)%32]) // 256 é múltiplo de 32: distribuição uniforme
	}
	d, _ := DigitoVerificador(string(corpo))
	return prefixo + string(append(corpo, d)), nil
}

func ehSeparador(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\r', '-', '.', '_', '/':
		return true
	}
	return false
}

// Normalizar converte o que foi digitado (com ou sem prefixo, hífens, espaços,
// minúsculas, O no lugar de 0, I/L no lugar de 1) para a forma guardada:
// prefixo + 8 caracteres. Confere alfabeto e dígito verificador.
func Normalizar(entrada, prefixo string) (string, error) {
	var b strings.Builder
	for _, r := range strings.ToUpper(entrada) {
		if !ehSeparador(r) {
			b.WriteRune(r)
		}
	}
	s := b.String()
	if s == "" {
		return "", ErrVazio
	}
	var corpo string
	switch {
	case len(s) == len(prefixo)+TamanhoCorpo && strings.HasPrefix(s, prefixo):
		corpo = s[len(prefixo):]
	case len(s) == TamanhoCorpo:
		corpo = s
	default:
		return "", ErrTamanho
	}
	c := []byte(corpo)
	for i, x := range c {
		switch x {
		case 'O':
			c[i] = '0'
		case 'I', 'L':
			c[i] = '1'
		}
		if indice(c[i]) < 0 {
			return "", ErrCaractere
		}
	}
	if !digitoConfere(string(c)) {
		return "", ErrDigito
	}
	return prefixo + string(c), nil
}

// Valido diz se o código já está na forma normalizada e com dígito correto.
func Valido(codigo, prefixo string) bool {
	n, err := Normalizar(codigo, prefixo)
	return err == nil && n == codigo
}

// Exibir formata o código guardado para leitura: EQ-7K3M-9QX1.
func Exibir(codigo string) string {
	if len(codigo) < TamanhoCorpo {
		return codigo
	}
	p := len(codigo) - TamanhoCorpo
	c := codigo[p:]
	return codigo[:p] + "-" + c[:4] + "-" + c[4:]
}
