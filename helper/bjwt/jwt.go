package bjwt

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

type Claim struct {
	UserName string
	jwt.RegisteredClaims
}

func MakeHsToken(claims Claim, key []byte) (string, error) {

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	str, err := token.SignedString(key)
	if err != nil {
		return "", err
	}
	return str, nil

}
func ParseHsToken(tokenStr string, key []byte) (*Claim, error) {

	token, err := jwt.ParseWithClaims(tokenStr, &Claim{}, func(token *jwt.Token) (i interface{}, err error) {
		return key, nil
	})

	if err != nil {
		return nil, err
	}

	if claim, ok := token.Claims.(*Claim); ok && token.Valid {
		return claim, nil
	}
	return nil, errors.New("invalid token")
}
