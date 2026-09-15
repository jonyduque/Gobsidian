//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

// contexto descreve o token, o job e o pacote do processo. Sao as tres coisas
// que separavam o processo do Claude Desktop dos processos de teste em
// 2026-09-14: integridade Medium, job com LimitFlags=0x3C00, sem identidade de
// pacote.
func contexto() string {
	return fmt.Sprintf("%s | %s | pacote: %s", token(), job(), pacote())
}

func token() string {
	tk, err := syscall.OpenCurrentProcessToken()
	if err != nil {
		return "token: " + errno(err)
	}
	defer func() { _ = tk.Close() }()

	var n uint32
	_ = syscall.GetTokenInformation(tk, syscall.TokenIntegrityLevel, nil, 0, &n)
	if n == 0 {
		return "token: tamanho de TokenIntegrityLevel indisponivel"
	}
	buf := make([]byte, n)
	if err := syscall.GetTokenInformation(tk, syscall.TokenIntegrityLevel, &buf[0], n, &n); err != nil {
		return "token: " + errno(err)
	}
	// TOKEN_MANDATORY_LABEL tem o mesmo leiaute de TOKEN_USER: um
	// SID_AND_ATTRIBUTES.
	rotulo := (*syscall.Tokenuser)(unsafe.Pointer(&buf[0]))
	sid, err := rotulo.User.Sid.String()
	if err != nil {
		sid = "?"
	}

	var elevado, m uint32
	_ = syscall.GetTokenInformation(tk, syscall.TokenElevation, (*byte)(unsafe.Pointer(&elevado)), 4, &m)
	return fmt.Sprintf("integridade=%s elevado=%d", sid, elevado)
}

func job() string {
	k := syscall.NewLazyDLL("kernel32.dll")
	proc, err := syscall.GetCurrentProcess()
	if err != nil {
		return "job: " + errno(err)
	}
	var dentro int32
	if r, _, e := k.NewProc("IsProcessInJob").Call(uintptr(proc), 0, uintptr(unsafe.Pointer(&dentro))); r == 0 {
		return "job: IsProcessInJob " + e.Error()
	}
	if dentro == 0 {
		return "fora de job"
	}
	// JOBOBJECT_EXTENDED_LIMIT_INFORMATION (classe 9): LimitFlags no offset 16
	// em amd64.
	ext := make([]byte, 144)
	var ret uint32
	r, _, e := k.NewProc("QueryInformationJobObject").Call(0, 9, uintptr(unsafe.Pointer(&ext[0])), uintptr(len(ext)), uintptr(unsafe.Pointer(&ret)))
	if r == 0 {
		return "em job; LimitFlags ilegivel: " + e.Error()
	}
	return fmt.Sprintf("em job; LimitFlags=0x%X", *(*uint32)(unsafe.Pointer(&ext[16])))
}

func pacote() string {
	k := syscall.NewLazyDLL("kernel32.dll")
	var n uint32 = 256
	buf := make([]uint16, n)
	r, _, _ := k.NewProc("GetCurrentPackageFullName").Call(uintptr(unsafe.Pointer(&n)), uintptr(unsafe.Pointer(&buf[0])))
	switch r {
	case 0:
		return syscall.UTF16ToString(buf)
	case 15700: // APPMODEL_ERROR_NO_PACKAGE
		return "sem identidade de pacote"
	default:
		return fmt.Sprintf("erro %d", r)
	}
}
