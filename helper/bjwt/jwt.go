package bjwt

import (
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
)

type Claim struct {
	UserName string
	jwt.RegisteredClaims
}

func MakeHsToken(claims Claim) (string, error) {

	signKey := []byte(viper.GetString("jwtKey"))
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	str, err := token.SignedString(signKey)
	if err != nil {
		return "", err
	}
	return str, nil

}
func ParseHsToken(tokenStr string) (*Claim, error) {

	signKey := []byte(viper.GetString("jwtKey"))

	token, err := jwt.ParseWithClaims(tokenStr, &Claim{}, func(token *jwt.Token) (i interface{}, err error) {
		return signKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claim, ok := token.Claims.(*Claim); ok && token.Valid {
		return claim, nil
	}
	return nil, errors.New("invalid token")
}
