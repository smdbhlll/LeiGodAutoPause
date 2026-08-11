//go:build windows

package startup

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const (
	hkeyCurrentUser = 0x80000001
	keySetValue     = 0x0002
	regSZ           = 1
	fileNotFound    = 2
	valueName       = "LeiGodAutoPause"
)

var (
	advapi32       = syscall.NewLazyDLL("advapi32.dll")
	regCreateKeyEx = advapi32.NewProc("RegCreateKeyExW")
	regSetValueEx  = advapi32.NewProc("RegSetValueExW")
	regDeleteValue = advapi32.NewProc("RegDeleteValueW")
	regCloseKey    = advapi32.NewProc("RegCloseKey")
)

func SetEnabled(enabled bool) error {
	path, err := os.Executable()
	if err != nil {
		return err
	}
	subKey, _ := syscall.UTF16PtrFromString(`Software\Microsoft\Windows\CurrentVersion\Run`)
	var key uintptr
	result, _, _ := regCreateKeyEx.Call(hkeyCurrentUser, uintptr(unsafe.Pointer(subKey)), 0, 0, 0, keySetValue, 0, uintptr(unsafe.Pointer(&key)), 0)
	if result != 0 {
		return fmt.Errorf("无法打开开机启动注册表项: %d", result)
	}
	defer regCloseKey.Call(key)
	name, _ := syscall.UTF16PtrFromString(valueName)
	if !enabled {
		result, _, _ = regDeleteValue.Call(key, uintptr(unsafe.Pointer(name)))
		if result != 0 && result != fileNotFound {
			return fmt.Errorf("无法关闭开机自启: %d", result)
		}
		return nil
	}
	command := `"` + path + `" -autostart`
	data, _ := syscall.UTF16FromString(command)
	result, _, _ = regSetValueEx.Call(key, uintptr(unsafe.Pointer(name)), 0, regSZ, uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)*2))
	if result != 0 {
		return fmt.Errorf("无法设置开机自启: %d", result)
	}
	return nil
}
