package ui

import "time"

type Ui struct {
	Stmt struct {
		Host     string `json:"host"`
		Port     string `json:"port"`
		User     string `json:"user"`
		Password string `json:"password"`
	}
	GoogleAuth struct {
		ClientId     string
		ClientSecret string
		CallbackUrl  string
	}
	SendGrid struct {
		Key         string
		FromName    string
		FromAddress string
	}
	Root struct {
		UserName string `json:"username"`
		Password string `json:"password"`
	} `json:"root"`
	On        bool          `json:"on"`
	Issuer    string        `json:"issuer"`
	Subject   string        `json:"subject"`
	JwtKey    string        `json:"jwtKey"`
	Port      string        `json:"port"`
	ExpiresAt time.Duration `json:"expiresAt"`
}
