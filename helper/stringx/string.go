package stringx

import "unsafe"

// REFERENCE:
// https://github.com/kubernetes/apiserver/blob/706a6d89cf35950281e095bb1eeed5e3211d6272/pkg/authentication/token/cache/cached_token_authenticator.go#L263-L271

func StringToByte(str string) []byte {
	return *(*[]byte)(unsafe.Pointer(&str))
}

func ByteToString(data []byte) string {
	return *(*string)(unsafe.Pointer(&data))
}
