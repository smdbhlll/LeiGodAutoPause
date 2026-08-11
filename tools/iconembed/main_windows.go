//go:build windows

package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const (
	rtIcon      = 3
	rtGroupIcon = 14
)

var (
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	beginUpdateResource = kernel32.NewProc("BeginUpdateResourceW")
	updateResource      = kernel32.NewProc("UpdateResourceW")
	endUpdateResource   = kernel32.NewProc("EndUpdateResourceW")
)

type iconEntry struct {
	width      byte
	height     byte
	colorCount byte
	reserved   byte
	planes     uint16
	bitCount   uint16
	data       []byte
}

func main() {
	exePath := flag.String("exe", "", "path to the executable")
	icoPath := flag.String("ico", "", "path to the icon")
	flag.Parse()
	if *exePath == "" || *icoPath == "" {
		fatal(errors.New("both -exe and -ico are required"))
	}
	iconData, err := os.ReadFile(*icoPath)
	if err != nil {
		fatal(err)
	}
	entries, err := parseICO(iconData)
	if err != nil {
		fatal(err)
	}
	if err := embedIcon(*exePath, entries); err != nil {
		fatal(err)
	}
}

func parseICO(data []byte) ([]iconEntry, error) {
	if len(data) < 6 || binary.LittleEndian.Uint16(data[0:2]) != 0 || binary.LittleEndian.Uint16(data[2:4]) != 1 {
		return nil, errors.New("invalid ICO header")
	}
	count := int(binary.LittleEndian.Uint16(data[4:6]))
	if count == 0 || len(data) < 6+count*16 {
		return nil, errors.New("ICO contains no complete entries")
	}
	entries := make([]iconEntry, 0, count)
	for index := range count {
		offset := 6 + index*16
		size := int(binary.LittleEndian.Uint32(data[offset+8 : offset+12]))
		imageOffset := int(binary.LittleEndian.Uint32(data[offset+12 : offset+16]))
		if size <= 0 || imageOffset < 0 || imageOffset > len(data)-size {
			return nil, fmt.Errorf("invalid ICO entry %d", index)
		}
		entries = append(entries, iconEntry{
			width:      data[offset],
			height:     data[offset+1],
			colorCount: data[offset+2],
			reserved:   data[offset+3],
			planes:     binary.LittleEndian.Uint16(data[offset+4 : offset+6]),
			bitCount:   binary.LittleEndian.Uint16(data[offset+6 : offset+8]),
			data:       data[imageOffset : imageOffset+size],
		})
	}
	return entries, nil
}

func embedIcon(exePath string, entries []iconEntry) error {
	path, err := syscall.UTF16PtrFromString(exePath)
	if err != nil {
		return err
	}
	handle, _, callErr := beginUpdateResource.Call(uintptr(unsafe.Pointer(path)), 0)
	if handle == 0 {
		return fmt.Errorf("BeginUpdateResourceW failed: %w", callErr)
	}
	committed := false
	defer func() {
		if !committed {
			endUpdateResource.Call(handle, 1)
		}
	}()

	for index, entry := range entries {
		if err := writeResource(handle, rtIcon, uintptr(index+1), entry.data); err != nil {
			return err
		}
	}
	group := buildGroupIcon(entries)
	if err := writeResource(handle, rtGroupIcon, 1, group); err != nil {
		return err
	}
	if result, _, callErr := endUpdateResource.Call(handle, 0); result == 0 {
		return fmt.Errorf("EndUpdateResourceW failed: %w", callErr)
	}
	committed = true
	return nil
}

func writeResource(handle, resourceType, name uintptr, data []byte) error {
	result, _, callErr := updateResource.Call(handle, resourceType, name, 0, uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)))
	if result == 0 {
		return fmt.Errorf("UpdateResourceW type %d name %d failed: %w", resourceType, name, callErr)
	}
	return nil
}

func buildGroupIcon(entries []iconEntry) []byte {
	var group bytes.Buffer
	_ = binary.Write(&group, binary.LittleEndian, uint16(0))
	_ = binary.Write(&group, binary.LittleEndian, uint16(1))
	_ = binary.Write(&group, binary.LittleEndian, uint16(len(entries)))
	for index, entry := range entries {
		group.WriteByte(entry.width)
		group.WriteByte(entry.height)
		group.WriteByte(entry.colorCount)
		group.WriteByte(entry.reserved)
		_ = binary.Write(&group, binary.LittleEndian, entry.planes)
		_ = binary.Write(&group, binary.LittleEndian, entry.bitCount)
		_ = binary.Write(&group, binary.LittleEndian, uint32(len(entry.data)))
		_ = binary.Write(&group, binary.LittleEndian, uint16(index+1))
	}
	return group.Bytes()
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "iconembed:", err)
	os.Exit(1)
}
