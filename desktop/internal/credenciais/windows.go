//go:build windows

package credenciais

import (
	"errors"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Gerenciador de Credenciais do Windows (advapi32: CredWriteW, CredReadW, CredDeleteW).
// Credencial genérica, persistida para o usuário do Windows nesta máquina.

const (
	credTypeGeneric         = 1
	credPersistLocalMachine = 2
	errorNotFound           = windows.Errno(1168) // ERROR_NOT_FOUND
)

type credentialW struct {
	Flags              uint32
	Type               uint32
	TargetName         *uint16
	Comment            *uint16
	LastWritten        windows.Filetime
	CredentialBlobSize uint32
	CredentialBlob     *byte
	Persist            uint32
	AttributeCount     uint32
	Attributes         uintptr
	TargetAlias        *uint16
	UserName           *uint16
}

var (
	advapi32       = windows.NewLazySystemDLL("advapi32.dll")
	procCredWriteW = advapi32.NewProc("CredWriteW")
	procCredReadW  = advapi32.NewProc("CredReadW")
	procCredDelete = advapi32.NewProc("CredDeleteW")
	procCredFree   = advapi32.NewProc("CredFree")
)

// Windows usa o Gerenciador de Credenciais do Windows.
type Windows struct{}

// Nativo devolve o cofre do sistema.
func Nativo() Cofre { return Windows{} }

func (Windows) Gravar(alvo string, c Credencial) error {
	t, err := windows.UTF16PtrFromString(alvo)
	if err != nil {
		return err
	}
	u, err := windows.UTF16PtrFromString(c.Usuario)
	if err != nil {
		return err
	}
	coment, _ := windows.UTF16PtrFromString("SITPOSSE-7 Central")
	blob := []byte(c.Segredo)
	cred := credentialW{Type: credTypeGeneric, TargetName: t, UserName: u, Comment: coment, Persist: credPersistLocalMachine, CredentialBlobSize: uint32(len(blob))}
	if len(blob) > 0 {
		cred.CredentialBlob = &blob[0]
	}
	r, _, e := procCredWriteW.Call(uintptr(unsafe.Pointer(&cred)), 0)
	if r == 0 {
		return e
	}
	return nil
}

func (Windows) Ler(alvo string) (Credencial, error) {
	t, err := windows.UTF16PtrFromString(alvo)
	if err != nil {
		return Credencial{}, err
	}
	var p *credentialW
	r, _, e := procCredReadW.Call(uintptr(unsafe.Pointer(t)), credTypeGeneric, 0, uintptr(unsafe.Pointer(&p)))
	if r == 0 {
		if errors.Is(e, errorNotFound) {
			return Credencial{}, ErrNaoEncontrada
		}
		return Credencial{}, e
	}
	defer procCredFree.Call(uintptr(unsafe.Pointer(p)))
	segredo := string(unsafe.Slice(p.CredentialBlob, p.CredentialBlobSize))
	return Credencial{Usuario: windows.UTF16PtrToString(p.UserName), Segredo: segredo}, nil
}

func (Windows) Apagar(alvo string) error {
	t, err := windows.UTF16PtrFromString(alvo)
	if err != nil {
		return err
	}
	r, _, e := procCredDelete.Call(uintptr(unsafe.Pointer(t)), credTypeGeneric, 0)
	if r == 0 && !errors.Is(e, errorNotFound) {
		return e
	}
	return nil
}
