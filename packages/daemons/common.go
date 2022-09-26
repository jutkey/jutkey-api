package daemons

import (
	"context"
	"fmt"
	"jutkey-server/packages/crontab"
	"jutkey-server/packages/storage/geoip"
	"jutkey-server/packages/storage/locator"
	"jutkey-server/packages/storage/sql"
)

var ExitCh chan error

func StartDaemons(ctx context.Context) {
	//...

	err := locator.InitCountryLocator()
	if err != nil {
		ExitCh <- fmt.Errorf("Init Country Locator err:%s\n", err.Error())
	}

	err = geoip.InitGeoIpDB()
	if err != nil {
		ExitCh <- fmt.Errorf("GeoIp Database Init err:%s\n", err.Error())
	}

	var node sql.HonorNodeInfo
	err = node.CreateTable()
	if err != nil {
		ExitCh <- fmt.Errorf("Create honer node table err:%s\n", err.Error())
	}
	err = sql.InitSpentInfoHistory()
	if err != nil {
		ExitCh <- fmt.Errorf("Init Spent Info History err:%s\n", err.Error())
	}
	sql.InitEcosystemInfo()

	go crontab.CreateCrontab()
}
