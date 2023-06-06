package common

import "unsafe"

// Str2B zero allocation string convertion
// to byte slice
func Str2B(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(&s))
}
