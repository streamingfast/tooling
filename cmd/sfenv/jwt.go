package main

import (
	"github.com/golang-jwt/jwt/v5"
)

func init() {
	jwt.RegisterSigningMethod("KMSES256", func() jwt.SigningMethod {
		return acceptAllSigningMethod{}
	})
}

// ParseJWTUnverified parses a JWT token without verifying the signature.
func ParseJWTUnverified(tokenString string) (jwt.MapClaims, error) {
	claims := jwt.MapClaims{}
	_, _, err := jwt.NewParser().ParseUnverified(tokenString, claims)
	if err != nil {
		return nil, err
	}

	return claims, nil
}

var _ jwt.SigningMethod = acceptAllSigningMethod{}

type acceptAllSigningMethod struct {
}

func (a acceptAllSigningMethod) Alg() string { return "KMSES256" }
func (a acceptAllSigningMethod) Sign(signingString string, key interface{}) ([]byte, error) {
	panic("unimplemented")
}
func (a acceptAllSigningMethod) Verify(signingString string, sig []byte, key interface{}) error {
	return nil
}
