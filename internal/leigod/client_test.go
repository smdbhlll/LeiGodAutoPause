package leigod

import "testing"

func TestPasswordMD5(t *testing.T) {
	if got := PasswordMD5("123456"); got != "e10adc3949ba59abbe56e057f20f883e" {
		t.Fatalf("unexpected md5: %s", got)
	}
}
