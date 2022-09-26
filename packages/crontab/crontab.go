package crontab

import (
	"github.com/IBAX-io/go-ibax/packages/smart"
	log "github.com/sirupsen/logrus"
	"jutkey-server/packages/storage/sql"
	"sync"
)

type task struct {
	cmd         byte
	getDataOver bool
	sync.RWMutex
}

type crontab struct {
	signal chan byte
}

const (
	realTime = iota
	delay
)

//realtime
const (
	initPledgeAmount = iota
	updateHonorNodeInfo
	initGlobalSwitch
	getStatisticsData
	syncSpentInfoHistory
	syncEcosystemInfo
)

//delay
const (
	getHonorNode = iota
	loadContracts
)

func (p *crontab) crontabMain() {
	p.signal = make(chan byte, 3)
	var (
		r1Task = &task{cmd: initPledgeAmount, getDataOver: true}
		r2Task = &task{cmd: updateHonorNodeInfo, getDataOver: true}
		r3Task = &task{cmd: initGlobalSwitch, getDataOver: true}
		r4Task = &task{cmd: getStatisticsData, getDataOver: true}
		r5Task = &task{cmd: syncSpentInfoHistory, getDataOver: true}
		r6Task = &task{cmd: syncEcosystemInfo, getDataOver: true}

		d1Task = &task{cmd: getHonorNode, getDataOver: true}
		d2Task = &task{cmd: loadContracts, getDataOver: true}
	)
	for {
		select {
		case cmd := <-p.signal:
			switch cmd {
			case realTime:
				go r1Task.startUpRealTimeTask()
				go r2Task.startUpRealTimeTask()
				go r3Task.startUpRealTimeTask()
				go r4Task.startUpRealTimeTask()
				go r5Task.startUpRealTimeTask()
				go r6Task.startUpRealTimeTask()
			case delay:
				go d1Task.startUpDelayTask()
				go d2Task.startUpDelayTask()
			}

		}

	}
}

func (p *crontab) sendCrontabCmd(cmd byte) {
	if len(p.signal) < cap(p.signal) {
		p.signal <- cmd
	}
}

func (rk *task) startUpRealTimeTask() {
	rk.Lock()
	defer rk.Unlock()

	if !rk.getDataOver {
		return
	}

	rk.getDataOver = false
	defer func() {
		rk.getDataOver = true
	}()
	switch rk.cmd {
	case initPledgeAmount:
		sql.InitPledgeAmount()
	case updateHonorNodeInfo:
		sql.UpdateHonorNodeInfo()
	case initGlobalSwitch:
		sql.InitGlobalSwitch()
	case getStatisticsData:
		err := sql.GetStatisticsData()
		if err != nil {
			log.WithFields(log.Fields{"err:": err}).Error("Get Statistics Data Failed")
		}
	case syncSpentInfoHistory:
		err := sql.SpentInfoHistorySync()
		if err != nil {
			log.WithFields(log.Fields{"err:": err}).Error("Spent Info History Sync Failed")
		}
	case syncEcosystemInfo:
		sql.SyncEcosystemInfo()
	}
}

func (rk *task) startUpDelayTask() {
	if !rk.getDataOver {
		return
	}
	rk.getDataOver = false
	defer func() {
		rk.getDataOver = true
	}()

	switch rk.cmd {
	case getHonorNode:
		sql.GetHonorNode()
	case loadContracts:
		err := smart.LoadContracts()
		if err != nil {
			log.WithFields(log.Fields{"err:": err}).Error("Load Contracts Failed")
		}
	}
}
