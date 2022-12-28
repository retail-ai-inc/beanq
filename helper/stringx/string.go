package stringx

import "unsafe"

// REFERENCE:
// https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/authentication/token/cache/cached_token_authenticator.go

func StringToByte(str string) []byte {
	return *(*[]byte)(unsafe.Pointer(&str))
}
func ByteToString(data []byte) string {
	return *(*string)(unsafe.Pointer(&data))
}
