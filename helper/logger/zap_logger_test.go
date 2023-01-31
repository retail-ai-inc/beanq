package logger

import "testing"

func TestComposeParametere(t *testing.T) {
	info := composeParameter(WithInfoFile("aa"), WithErrFile("bb"), WithMaxSize(300))
	t.Fatalf("%+v \n", info)
}
