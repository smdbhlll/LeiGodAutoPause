//go:build windows

package processes

import (
	"errors"
	"sort"
	"syscall"
	"unsafe"
)

const (
	th32csSnapProcess               = 0x00000002
	processQueryLimitedInfo         = 0x1000
	invalidHandleValue      uintptr = ^uintptr(0)
)

type processEntry32 struct {
	Size              uint32
	Usage             uint32
	ProcessID         uint32
	DefaultHeapID     uintptr
	ModuleID          uint32
	Threads           uint32
	ParentProcessID   uint32
	PriorityClassBase int32
	Flags             uint32
	ExeFile           [260]uint16
}

type nativeScanner struct{}

var (
	kernel                    = syscall.NewLazyDLL("kernel32.dll")
	createToolhelp32Snapshot  = kernel.NewProc("CreateToolhelp32Snapshot")
	process32First            = kernel.NewProc("Process32FirstW")
	process32Next             = kernel.NewProc("Process32NextW")
	openProcess               = kernel.NewProc("OpenProcess")
	queryFullProcessImageName = kernel.NewProc("QueryFullProcessImageNameW")
	closeHandle               = kernel.NewProc("CloseHandle")
)

func (nativeScanner) List() ([]Process, error) {
	handle, _, callErr := createToolhelp32Snapshot.Call(th32csSnapProcess, 0)
	if handle == invalidHandleValue {
		return nil, callErr
	}
	defer closeHandle.Call(handle)

	entry := processEntry32{Size: uint32(unsafe.Sizeof(processEntry32{}))}
	ok, _, _ := process32First.Call(handle, uintptr(unsafe.Pointer(&entry)))
	if ok == 0 {
		return nil, errors.New("cannot enumerate Windows processes")
	}
	items := make([]Process, 0, 128)
	for {
		name := syscall.UTF16ToString(entry.ExeFile[:])
		items = append(items, Process{PID: entry.ProcessID, Name: name, Path: processPath(entry.ProcessID)})
		entry.Size = uint32(unsafe.Sizeof(processEntry32{}))
		next, _, _ := process32Next.Call(handle, uintptr(unsafe.Pointer(&entry)))
		if next == 0 {
			break
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}

func processPath(pid uint32) string {
	handle, _, _ := openProcess.Call(processQueryLimitedInfo, 0, uintptr(pid))
	if handle == 0 {
		return ""
	}
	defer closeHandle.Call(handle)
	buffer := make([]uint16, 32768)
	size := uint32(len(buffer))
	ok, _, _ := queryFullProcessImageName.Call(handle, 0, uintptr(unsafe.Pointer(&buffer[0])), uintptr(unsafe.Pointer(&size)))
	if ok == 0 {
		return ""
	}
	return syscall.UTF16ToString(buffer[:size])
}
