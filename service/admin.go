package service

import (
	"github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"
)

type adminClaims struct {
	*jwt.RegisteredClaims
	Id int64
}

type Admin struct {
	secret string
}

func NewAdmin(secret string) Admin {
	return Admin{
		secret: secret,
	}
}

func (s Admin) Authenticate(token string) (int, error) {
	parsed, err := s.parseToken(token, &adminClaims{RegisteredClaims: &jwt.RegisteredClaims{}})
	if err != nil {
		return 0, errors.WithMessage(err, "parse token")
	}

	claims, ok := parsed.Claims.(*adminClaims)
	if !ok {
		return 0, errors.New("invalid token")
	}

	return int(claims.Id), nil
}

func (s Admin) parseToken(token string, claims jwt.Claims) (*jwt.Token, error) {
	parsed, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		_, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			return nil, errors.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(s.secret), nil
	})
	if err != nil {
		return nil, errors.WithMessage(err, "jwt parse with claims")
	}
	if !parsed.Valid {
		return nil, errors.New("invalid token")
	}

	return parsed, nil
}
