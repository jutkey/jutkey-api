package services

import (
	"errors"
	"github.com/golang-jwt/jwt/v4"
	log "github.com/sirupsen/logrus"
	"jutkey-server/conf"
	"strconv"
	"time"
)

var centrifugoTimeout = time.Second * 5

const (
	CryptoError = "Crypto"
)

type CentJWT struct {
	Sub string
	jwt.StandardClaims
}

type CentJWTToken struct {
	Token string `json:"token"`
	Url   string `json:"url"`
}

func GetJWTCentToken(userID, expire int64) (*CentJWTToken, error) {
	if conf.GetEnvConf().Centrifugo.Enable {
		var ret CentJWTToken
		centJWT := CentJWT{
			Sub: strconv.FormatInt(userID, 10),
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: time.Now().Add(time.Second * time.Duration(expire)).Unix(),
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, centJWT)
		result, err := token.SignedString([]byte(conf.GetEnvConf().Centrifugo.Secret))

		if err != nil {
			log.WithFields(log.Fields{"type": CryptoError, "error": err}).Error("JWT centrifugo error")
			return &ret, err
		}
		ret.Token = result
		ret.Url = conf.GetEnvConf().Centrifugo.Socket
		return &ret, nil
	} else {
		var ret CentJWTToken
		return &ret, errors.New("centrifugo not enable")
	}
}
