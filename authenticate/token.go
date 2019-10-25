package authenticate

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/integration-system/isp-lib/config"
	"github.com/pkg/errors"
	"isp-gate-service/conf"
)

var token tokenHelper

type tokenHelper struct{}

func (tokenHelper) Check(token string) error {
	secret := config.GetRemote().(*conf.RemoteConfig).SecretKey

	if parsed, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	}); err != nil {
		return err
	} else if !parsed.Valid {
		return errors.New("token is invalid")
	}

	return nil
}
