// Package fontes embute a família Archivo (OFL-1.1, OFL.txt), usada nos relatórios
// e nas cartas-imagem. Nada é baixado em tempo de execução.
package fontes

import _ "embed"

var (
	//go:embed Archivo-Regular.ttf
	Regular []byte
	//go:embed Archivo-SemiBold.ttf
	SemiBold []byte
	//go:embed Archivo-Bold.ttf
	Bold []byte
	//go:embed ArchivoCondensed-Bold.ttf
	CondensedBold []byte
)
