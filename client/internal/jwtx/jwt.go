package jwtx

import (
	"os"
	"path"
	"path/filepath"
	"runtime"

	"github.com/golang-jwt/jwt/v5"
)

type Claim struct {
	UserName string
	jwt.Claims
}

func MakeRsaToken(claims Claim) (string, error) {

	_, p, _, ok := runtime.Caller(0)
	if ok {
		p = filepath.Dir(p)
		p = path.Join(p, "cert", "private.pem")
	} else {
		p = "./private.pem"
	}

	b, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}

	signKey, err := jwt.ParseRSAPrivateKeyFromPEM(b)
	if err != nil {
		return "", err
	}

	token := jwt.NewWithClaims(jwt.GetSigningMethod("RS256"), claims)
	str, err := token.SignedString(signKey)
	if err != nil {
		return "", err
	}
	return str, nil
}
func ParseRsaToken(tokenStr string) (*jwt.Token, error) {

	_, p, _, ok := runtime.Caller(0)
	if ok {
		p = filepath.Dir(p)
		p = path.Join(p, "cert", "public.pem")
	} else {
		p = "./public.pem"
	}

	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	verifyKey, err := jwt.ParseRSAPublicKeyFromPEM(b)
	if err != nil {
		return nil, err
	}

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return verifyKey, nil
	})
	if err != nil {
		return nil, err
	}
	return token, nil
}
