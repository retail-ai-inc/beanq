package jwtx

import (
	"fmt"
	"log"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestMakeToken(t *testing.T) {

	claim := jwt.MapClaims{"name": "bb", "age": 10}
	str, _ := MakeToken(claim)
	fmt.Println(str)
}

func TestParseToken(t *testing.T) {
	str := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOiIyMDIzLTA4LTIzVDE4OjM4OjEyLjg5NjM4NzkzNiswODowMCIsInVzZXJuYW1lIjoiYWFhIn0.tro2PBRb792DgyT3EPcNUBTEAt7nuzUnosogBIAh4_K7LGvEjKhxtRTsqE52t28TLJOkuLIV7FdFK4DGnLrUXoQ4tdyfRxtCmXLBLrKzTXKs-upZodLzfgJ8Ti4KMGkGWFTekohhZv01J8QDF7BsTibbNHN3EqtfY7pfbZqmoIgIEvWfE-1UlKLbFxFpOPIv42M2YyBj3L-V-UIfM1Kf3GNnDCm-GiXKuJjDjKB807T_UxJ_IScyb7GvmGGnVFWhzYmVWqykjY4SZQg6kD2ZAH3E7qkfuS5gAyt9OQcAwk-W9I0xXCAWnwYQIvwQLuBRpRCsH4FcjPuNjeIGFuZDxw"
	token, err := ParseToken(str)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println(token.Claims)
}
