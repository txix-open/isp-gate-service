package service

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
)

type AdminMethodStore interface {
	IsInnerMethod(path string) bool
}

type adminClaims struct {
	*jwt.StandardClaims
	Id int64
}

type Admin struct {
	methodStore AdminMethodStore
	secret      string
}

func NewAdmin(secret string, methodStore AdminMethodStore) Admin {
	return Admin{
		methodStore: methodStore,
		secret:      secret,
	}
}

func (s Admin) Authorize(id int, path string) error {
	innerMethod := s.methodStore.IsInnerMethod(path)
	if !innerMethod {
		return nil
	}

	if id > 0 {
		return nil
	}

	return errors.New("not authorized method")
}

func (s Admin) Authenticate(token string) (int, error) {
	if token == "" {
		return -1, nil
	}

	parsed, err := s.parseToken(token, &adminClaims{StandardClaims: &jwt.StandardClaims{}})
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
