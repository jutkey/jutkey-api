package crontab

import (
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
	"jutkey-server/conf"
	"time"
)

var (
	cr crontab
)

func CreateCrontab() {
	crontabInfo := conf.GetEnvConf().Crontab

	go cr.crontabMain()
	time.Sleep(1 * time.Second)
	cr.sendCrontabCmd(realTime)
	cr.sendCrontabCmd(delay)

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
		cr.sendCrontabCmd(realTime)
	})
	if err != nil {
		log.WithFields(log.Fields{"error": err, "time set": timeSet}).Error("create Crontab From real Time Add Function Failed")
	}
	c.Start()
}

func createCrontabFromDelay(timeSet string) {
	c := newWithSecond()
	_, err := c.AddFunc(timeSet, func() {
		cr.sendCrontabCmd(delay)
	})
	if err != nil {
		log.WithFields(log.Fields{"error": err, "time set": timeSet}).Error("create Crontab From delay Add Function Failed")
	}
	c.Start()
}
