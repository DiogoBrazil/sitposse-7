package dominio

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// NormalizarNome gera a chave de agrupamento de itens "Outros" (RN-35).
// O texto original nunca é alterado; esta chave serve só para estatística.
// Retorna ok=false quando o nome fica vazio.
func NormalizarNome(original string) (normalizado string, ok bool) {
	s := strings.Join(strings.Fields(original), " ")
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range norm.NFD.String(s) {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		b.WriteRune(r)
	}
	n := b.String()
	return n, n != ""
}
