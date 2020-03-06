package authenticate

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/integration-system/isp-lib/v2/config"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"isp-gate-service/conf"
)

var verifyToken token

type appClaims struct {
	*jwt.StandardClaims
	AppId int32
}

type userClaims struct {
	*jwt.StandardClaims
	UserId int64
}

type token struct{}

func (t token) Admin(token string) error {
	secret := config.GetRemote().(*conf.RemoteConfig).TokensSetting.AdminSecret
	_, err := t.parse(token, secret, jwt.MapClaims{})
	return err
}

func (t token) Application(token string) (int32, error) {
	secret := config.GetRemote().(*conf.RemoteConfig).TokensSetting.ApplicationSecret

	parsed, err := t.parse(token, secret, &appClaims{StandardClaims: &jwt.StandardClaims{}})
	if err != nil {
		return 0, err
	}
	claims, ok := parsed.Claims.(*appClaims)
	if !ok {
		return 0, errors.New("token is invalid")
	}
	return claims.AppId, nil
}

func (t token) User(token string) (string, error) {
	secret := config.GetRemote().(*conf.RemoteConfig).TokensSetting.UserSecret

	parsed, err := t.parse(token, secret, &userClaims{StandardClaims: &jwt.StandardClaims{}})
	if err != nil {
		return "", err
	}
	claims, ok := parsed.Claims.(*userClaims)
	if !ok {
		return "", errors.New("token is invalid")
	}

	return cast.ToString(claims.UserId), nil
}

func (token) parse(token, secret string, claims jwt.Claims) (*jwt.Token, error) {
	parsed, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		_, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			return nil, errors.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !parsed.Valid {
		return nil, errors.New("token is invalid")
	}
	return parsed, nil
}
