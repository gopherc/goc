// Generated by the GopherC bind tool.
// 2019-04-04 21:04:36.8765417 +0200 CEST m=+0.004002401

// +build goc

package bind

func gocPutc(ch int32) int32

// Putc prints a character on the screen.
func Putc(ch int) int {
	_ch := int32(ch)
	_r := gocPutc(_ch)
	return int(_r)
}

