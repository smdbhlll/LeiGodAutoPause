//go:build !windows

package startup

import "errors"

func SetEnabled(bool) error { return errors.New("startup registration is supported on Windows only") }
