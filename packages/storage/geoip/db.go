package geoip

import (
	"github.com/oschwald/geoip2-golang"
	log "github.com/sirupsen/logrus"
	"jutkey-server/conf"
	"path/filepath"
)

var DB *geoip2.Reader

func InitGeoIpDB() error {
	var err error
	dbFile := filepath.Join(conf.GetEnvConf().ConfigPath, "geoip_db", "GeoLite2-City.mmdb")

	DB, err = geoip2.Open(dbFile)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Geo Ip Database Init open err")
		return err
	}
	return nil
}

func CloseGeoIp() {
	if DB != nil {
		err := DB.Close()
		if err != nil {
			log.WithFields(log.Fields{"error": err}).Error("Geo Ip Close Failed")
		}
	}
}
