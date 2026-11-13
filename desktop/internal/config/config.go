// Package config resolve caminhos de dados da Central.
package config

import (
	"os"
	"path/filepath"
)

// PastaDados devolve a pasta de dados: SITPOSSE7_DADOS, se definida; senão
// %LOCALAPPDATA%\SITPOSSE-7 no Windows (os.UserCacheDir) ou equivalente.
func PastaDados() (string, error) {
	if p := os.Getenv("SITPOSSE7_DADOS"); p != "" {
		return p, os.MkdirAll(p, 0o700)
	}
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	p := filepath.Join(base, "SITPOSSE-7")
	return p, os.MkdirAll(p, 0o700)
}
