package authenticate

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/integration-system/isp-lib/config"
	"github.com/pkg/errors"
	"isp-gate-service/conf"
)

var verifyToken token

type appClaims struct {
	*jwt.StandardClaims
	AppId int32
}

type token struct{}

func (t token) Admin(token string) error {
	secret := config.GetRemote().(*conf.RemoteConfig).Secrets.Admin
	_, err := t.parse(token, secret, jwt.MapClaims{})
	return err
}

func (t token) Application(token string) (int32, error) {
	secret := config.GetRemote().(*conf.RemoteConfig).Secrets.Application

	if parsed, err := t.parse(token, secret, &appClaims{StandardClaims: &jwt.StandardClaims{}}); err != nil {
		return 0, err
	} else if claims, ok := parsed.Claims.(*appClaims); !ok {
		return 0, errors.New("token is invalid")
	} else {
		return claims.AppId, nil
	}
}

func (token) parse(token, secret string, claims jwt.Claims) (*jwt.Token, error) {
	if parsed, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	}); err != nil {
		return nil, err
	} else if !parsed.Valid {
		return nil, errors.New("token is invalid")
	} else {
		return parsed, nil
	}
}
