//go:build !windows

package secret

import "encoding/base64"

func Protect(value string) (string, error) {
	return base64.StdEncoding.EncodeToString([]byte(value)), nil
}
func Unprotect(value string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(value)
	return string(b), err
}
