package authenticate

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/integration-system/isp-lib/config"
	"github.com/pkg/errors"
	"isp-gate-service/conf"
)

var validateToken token

type token struct{}

func (t token) Admin(token string) error {
	secret := config.GetRemote().(*conf.RemoteConfig).SecretSetting.Admin
	_, err := t.parse(token, secret)
	return err
}

func (t token) Application(token string) (int, error) {
	secret := config.GetRemote().(*conf.RemoteConfig).SecretSetting.Application

	if parsed, err := t.parse(token, secret); err != nil {
		return 0, err
	} else if claims, ok := parsed.Claims.(jwt.MapClaims); !ok {
		return 0, errors.New("token is invalid")
	} else if appId, ok := claims["appId"]; !ok {
		return 0, errors.New("token is invalid")
	} else if applicationId, ok := appId.(float64); !ok {
		return 0, errors.New("token is invalid")
	} else {
		return int(applicationId), nil
	}
}

func (token) parse(token, secret string) (*jwt.Token, error) {
	if parsed, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
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
