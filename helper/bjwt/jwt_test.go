package bjwt

import (
	"fmt"
	"log"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestMakeHsToken(t *testing.T) {

	claim := Claim{
		UserName:         "username",
		RegisteredClaims: jwt.RegisteredClaims{},
	}
	str, _ := MakeHsToken(claim)
	fmt.Println(str)
}

func TestParseHsToken(t *testing.T) {
	str := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJVc2VyTmFtZSI6InRyaWFsIiwiaXNzIjoiVHJpYWwgQ2hpbmEiLCJzdWIiOiJiZWFucSBtb25pdG9yIHVpIiwiZXhwIjoxNzA0NjkwMzcyfQ.dXX_TxFi-ASQWnHbXJrnc2ZteGHEzxWLffuSw6GRLgo "
	token, err := ParseHsToken(str)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println(token)
}
