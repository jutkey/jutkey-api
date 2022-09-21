package crontab

import (
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
	"jutkey-server/conf"
	"jutkey-server/packages/storage/sql"
)

func CreateCrontab() {
	crontabInfo := conf.GetEnvConf().Crontab
	initCrontabTask()
	if crontabInfo != nil {
		go createCrontabFromRealTime(crontabInfo.RealTime)
		go createCrontabFromDelay(crontabInfo.Delay)
	}

}

func newWithSecond() *cron.Cron {
	secondParser := cron.NewParser(cron.Second | cron.Minute |
		cron.Hour | cron.Dom | cron.Month | cron.DowOptional | cron.Descriptor)
	return cron.New(cron.WithParser(secondParser), cron.WithChain())
}

func createCrontabFromRealTime(timeSet string) {
	c := newWithSecond()
	_, err := c.AddFunc(timeSet, func() {
		realTimeDataTask()
	})
	if err != nil {
		log.WithFields(log.Fields{"error": err, "time set": timeSet}).Error("create Crontab From real Time Add Function Failed")
	}
	c.Start()
}

func createCrontabFromDelay(timeSet string) {
	c := newWithSecond()
	_, err := c.AddFunc(timeSet, func() {
		delayDataTask()
	})
	if err != nil {
		log.WithFields(log.Fields{"error": err, "time set": timeSet}).Error("create Crontab From delay Add Function Failed")
	}
	c.Start()
}

func initCrontabTask() {
	realTimeDataTask()
	delayDataTask()
}

func realTimeDataTask() {
	go sql.InitPledgeAmount()
	go sql.UpdateHonorNodeInfo()
	go initGlobalSwitch()
	go sql.SendStatisticsSignal()
}

func delayDataTask() {
	go sql.GetHonorNode()
}

func initGlobalSwitch() {
	sql.NodeReady = sql.CandidateTableExist()
	sql.NftMinerReady = sql.NftMinerTableIsExist()
}
