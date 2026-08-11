//go:build windows

package secret

import (
	"encoding/base64"
	"errors"
	"syscall"
	"unsafe"
)

type dataBlob struct {
	cbData uint32
	pbData *byte
}

var (
	crypt32          = syscall.NewLazyDLL("crypt32.dll")
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	cryptProtectData = crypt32.NewProc("CryptProtectData")
	cryptUnprotect   = crypt32.NewProc("CryptUnprotectData")
	localFree        = kernel32.NewProc("LocalFree")
)

func Protect(value string) (string, error) {
	data := []byte(value)
	in := blob(data)
	var out dataBlob
	ok, _, callErr := cryptProtectData.Call(uintptr(unsafe.Pointer(&in)), 0, 0, 0, 0, 1, uintptr(unsafe.Pointer(&out)))
	if ok == 0 {
		return "", callErr
	}
	defer localFree.Call(uintptr(unsafe.Pointer(out.pbData)))
	protected := unsafe.Slice(out.pbData, out.cbData)
	return base64.StdEncoding.EncodeToString(protected), nil
}

func Unprotect(value string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", nil
	}
	in := blob(data)
	var out dataBlob
	ok, _, callErr := cryptUnprotect.Call(uintptr(unsafe.Pointer(&in)), 0, 0, 0, 0, 1, uintptr(unsafe.Pointer(&out)))
	if ok == 0 {
		return "", callErr
	}
	if out.pbData == nil {
		return "", errors.New("DPAPI returned empty data")
	}
	defer localFree.Call(uintptr(unsafe.Pointer(out.pbData)))
	plain := unsafe.Slice(out.pbData, out.cbData)
	return string(plain), nil
}

func blob(data []byte) dataBlob {
	if len(data) == 0 {
		return dataBlob{}
	}
	return dataBlob{cbData: uint32(len(data)), pbData: &data[0]}
}
