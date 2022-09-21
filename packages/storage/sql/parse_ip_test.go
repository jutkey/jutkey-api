package sql

import (
	"fmt"
	"jutkey-server/conf"
	"jutkey-server/packages/storage/geoip"
	"jutkey-server/packages/storage/locator"
	"testing"
)

func TestGetLocator(t *testing.T) {
	conf.GetEnvConf().ConfigPath = "E:\\workspace\\work\\project\\IBAX\\jutkey-server\\conf"
	conf.LoadConfig(conf.GetEnvConf().ConfigPath)
	err := locator.InitCountryLocator()
	if err != nil {
		fmt.Printf("Init Country Locator failed:%s\n", err.Error())
		return
	}
	err = geoip.InitGeoIpDB()
	if err != nil {
		fmt.Printf("geoip database init failed:%s\n", err.Error())
		return
	}
	ipstr := "180.214.232.11"
	got, err := GetLocator(ipstr)
	if err != nil {
		fmt.Printf("get locator failed:%s\n", err.Error())
		return
	}
	fmt.Printf("got+%v\n", got)
}
