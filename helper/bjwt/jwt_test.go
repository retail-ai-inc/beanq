package bjwt

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

const signkey = "!@$!@$werwWER"

func TestMakeHsToken(t *testing.T) {

	claims := Claim{
		UserName: "testuser",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	}

	tokenStr, err := MakeHsToken(claims, []byte(signkey))
	assert.NoError(t, err)
	assert.NotEmpty(t, tokenStr)
}

func TestParseHsToken(t *testing.T) {

	tests := []struct {
		name     string
		tokenStr string
		key      []byte
		wantUser string
		wantErr  string
	}{
		{
			name:     "Valid Token",
			tokenStr: generateToken([]byte(signkey), "testuser", time.Now().Add(1*time.Hour)),
			key:      []byte(signkey),
			wantUser: "testuser",
			wantErr:  "",
		},
		{
			name:     "Invalid Key",
			tokenStr: generateToken([]byte(signkey), "testuser", time.Now().Add(1*time.Hour)),
			key:      []byte("wrong-key"),
			wantUser: "",
			wantErr:  jwt.ErrSignatureInvalid.Error(),
		},
		{
			name:     "Expired Token",
			tokenStr: generateToken([]byte(signkey), "testuser", time.Now().Add(-1*time.Hour)),
			key:      []byte(signkey),
			wantUser: "",
			wantErr:  "token is expired",
		},
		{
			name:     "Invalid Token Format",
			tokenStr: "invalid.token.format",
			key:      []byte(signkey),
			wantUser: "",
			wantErr:  "invalid token",
		},
		{
			name:     "Empty Token",
			tokenStr: "",
			key:      []byte(signkey),
			wantUser: "",
			wantErr:  "token is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claim, err := ParseHsToken(tt.tokenStr, tt.key)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, claim)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, claim)
				assert.Equal(t, tt.wantUser, claim.UserName)
			}
		})
	}
}

func generateToken(key []byte, username string, expireAt time.Time) string {
	claims := Claim{
		UserName: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireAt),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString(key)
	return tokenStr
}
