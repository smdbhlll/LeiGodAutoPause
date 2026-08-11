//go:build !windows

package processes

import "errors"

type nativeScanner struct{}

func (nativeScanner) List() ([]Process, error) {
	return nil, errors.New("process scanning is supported on Windows only")
}
