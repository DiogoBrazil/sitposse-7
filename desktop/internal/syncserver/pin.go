package syncserver

import (
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"errors"
)

// ErrPinDivergente indica que o servidor não é a Central pareada.
var ErrPinDivergente = errors.New("certificado da Central não corresponde ao pareamento (pin SPKI divergente)")

// ConfigClienteComPin devolve uma configuração TLS de cliente que aceita SOMENTE
// o servidor cuja chave pública tem o pin informado. A validação por cadeia de CA
// é substituída pela verificação explícita do pin (ADR-0007): não se trata de
// desabilitar a verificação, e sim de usar outra âncora de confiança.
// Usada pelos testes e por ferramentas de diagnóstico; o Android implementa o mesmo.
func ConfigClienteComPin(pin string) *tls.Config {
	return &tls.Config{
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: true, // substituído por VerifyConnection abaixo
		VerifyConnection: func(cs tls.ConnectionState) error {
			if len(cs.PeerCertificates) == 0 {
				return ErrPinDivergente
			}
			return verificarPin(cs.PeerCertificates[0], pin)
		},
	}
}

func verificarPin(cert *x509.Certificate, pin string) error {
	obtido := PinDaChavePublica(cert.PublicKey)
	if obtido == "" || subtle.ConstantTimeCompare([]byte(obtido), []byte(pin)) != 1 {
		return ErrPinDivergente
	}
	return nil
}
