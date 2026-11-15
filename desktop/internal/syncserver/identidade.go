// Package syncserver implementa o servidor de sincronização local da Central
// (docs/synchronization/protocolo.md). Fase 0: identidade TLS, pinning e /v1/status.
package syncserver

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"slices"
	"time"
)

// Identidade é o par de chaves da Central. A chave é gerada uma única vez e
// preservada (entra no backup); o certificado pode ser reemitido quando os IPs
// mudam sem alterar o pin, que é calculado sobre a chave pública (SPKI). ADR-0007.
type Identidade struct {
	Chave       *ecdsa.PrivateKey
	Certificado tls.Certificate
	IPs         []net.IP
}

const (
	arquivoChave = "central.key.pem"
	arquivoCert  = "central.crt.pem"
)

// CarregarOuCriarIdentidade lê a chave de dir (criando se não existir) e garante
// um certificado cujo SAN contenha ips.
func CarregarOuCriarIdentidade(dir string, ips []net.IP) (*Identidade, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	chave, err := lerChave(filepath.Join(dir, arquivoChave))
	if errors.Is(err, os.ErrNotExist) {
		chave, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, err
		}
		der, err := x509.MarshalECPrivateKey(chave)
		if err != nil {
			return nil, err
		}
		if err := gravarAtomico(filepath.Join(dir, arquivoChave), pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der}), 0o600); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	certPath := filepath.Join(dir, arquivoCert)
	if cert, err := lerCert(certPath, chave); err == nil && cobreIPs(cert.Leaf, ips) && time.Until(cert.Leaf.NotAfter) > 365*24*time.Hour {
		// IPs anunciados (tela e QR) são os atuais, não o SAN do certificado: ele
		// sempre traz o loopback e pode trazer IPs de redes antigas.
		return &Identidade{Chave: chave, Certificado: cert, IPs: ips}, nil
	}
	cert, err := emitirCert(chave, ips)
	if err != nil {
		return nil, err
	}
	if err := gravarAtomico(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]}), 0o644); err != nil {
		return nil, err
	}
	return &Identidade{Chave: chave, Certificado: cert, IPs: ips}, nil
}

// Pin devolve o SHA-256 do SPKI da Central em base64url sem padding (vai no QR).
func (id *Identidade) Pin() string { return PinDaChavePublica(&id.Chave.PublicKey) }

// PinDaChavePublica calcula o pin SPKI de uma chave pública.
func PinDaChavePublica(pub any) string {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return ""
	}
	h := sha256.Sum256(der)
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func emitirCert(chave *ecdsa.PrivateKey, ips []net.IP) (tls.Certificate, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 127))
	if err != nil {
		return tls.Certificate{}, err
	}
	modelo := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "SITPOSSE-7 Central", Organization: []string{"7º BPM/PMRO"}},
		NotBefore:    time.Now().Add(-24 * time.Hour),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  append([]net.IP{net.IPv4(127, 0, 0, 1)}, ips...),
		DNSNames:     []string{"localhost"},
	}
	der, err := x509.CreateCertificate(rand.Reader, modelo, modelo, &chave.PublicKey, chave)
	if err != nil {
		return tls.Certificate{}, err
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: chave, Leaf: leaf}, nil
}

func lerChave(p string) (*ecdsa.PrivateKey, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	blk, _ := pem.Decode(b)
	if blk == nil {
		return nil, fmt.Errorf("%s: PEM inválido", p)
	}
	return x509.ParseECPrivateKey(blk.Bytes)
}

func lerCert(p string, chave *ecdsa.PrivateKey) (tls.Certificate, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return tls.Certificate{}, err
	}
	blk, _ := pem.Decode(b)
	if blk == nil {
		return tls.Certificate{}, errors.New("certificado PEM inválido")
	}
	leaf, err := x509.ParseCertificate(blk.Bytes)
	if err != nil {
		return tls.Certificate{}, err
	}
	if PinDaChavePublica(leaf.PublicKey) != PinDaChavePublica(&chave.PublicKey) {
		return tls.Certificate{}, errors.New("certificado não corresponde à chave")
	}
	return tls.Certificate{Certificate: [][]byte{blk.Bytes}, PrivateKey: chave, Leaf: leaf}, nil
}

func cobreIPs(leaf *x509.Certificate, ips []net.IP) bool {
	for _, ip := range ips {
		if !slices.ContainsFunc(leaf.IPAddresses, ip.Equal) {
			return false
		}
	}
	return true
}

// gravarAtomico grava em arquivo temporário, faz fsync e renomeia.
func gravarAtomico(p string, dados []byte, perm os.FileMode) error {
	tmp := p + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := f.Write(dados); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}
