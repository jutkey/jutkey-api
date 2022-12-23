package sql

import (
	"encoding/json"
	"github.com/IBAX-io/go-ibax/packages/storage/sqldb"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"strconv"
	"time"
)

type AirdropInfo struct {
	Id            int64 `gorm:"primary_key;not_null"`
	Account       string
	BalanceAmount decimal.Decimal
	DateCreated   int64
	Detail        string `gorm:"type:jsonb"`
	DirectAmount  decimal.Decimal
	LatestAt      int64
	LockAmount    decimal.Decimal
	PeriodCount   int64
	Priority      int64
	StakeAmount   decimal.Decimal
	Surplus       int64
	TotalAmount   decimal.Decimal
}

type detailInfo struct {
	End    string `json:"end"`
	Start  string `json:"start"`
	Amount string `json:"amount"`
	Format string `json:"format"`
	Period string `json:"period"`
	Status int    `json:"status"`
}

var (
	AirdropReady bool
)

func (m AirdropInfo) TableName() string {
	return `1_airdrop_info`
}

func AirdropTableExist() bool {
	var p AirdropInfo
	if !HasTableOrView(p.TableName()) {
		return false
	}
	return true
}

func (p *AirdropInfo) Get(account string) (bool, error) {
	return isFound(GetDB(nil).Where("account = ?", account).Take(p))
}

func (p *AirdropInfo) GetAirdropInfo() (*AirdropInfoResponse, error) {
	rets := &AirdropInfoResponse{}
	f, err := p.Get(p.Account)
	if err != nil {
		return rets, err
	}
	if !f {
		return rets, nil
	}

	rets.Total = p.TotalAmount.String()
	rets.IsGet = p.TotalAmount.Sub(p.BalanceAmount).String()
	var speed int64 = 1
	switch p.Priority {
	case 2:
		speed = 5
	case 3:
		speed = 10
	case 5:
		speed = 20
	}

	now := time.Now()
	rets.X5Lock = p.LockAmount.Mul(decimal.NewFromInt(2)).String()
	rets.X10Lock = p.LockAmount.Mul(decimal.NewFromInt(3)).String()
	rets.X20Lock = p.LockAmount.Mul(decimal.NewFromInt(5)).String()

	unlocking := p.LatestAt > now.Unix()
	rets.CanSpeedUp = (p.Priority == 0) && unlocking
	rets.NowSpeedUp = speed
	rets.UnLockAll = !unlocking

	if rets.CanSpeedUp || unlocking {
		rets.Lock = p.BalanceAmount.String()

		var dList []detailInfo
		err = json.Unmarshal([]byte(p.Detail), &dList)
		if err != nil {
			return rets, err
		}

		var (
			findOut bool
		)
		for _, v := range dList {
			end, _ := strconv.ParseInt(v.End, 10, 64)
			st, _ := strconv.ParseInt(v.Start, 10, 64)
			if findOut {
				rets.NextPeriod = int64(time.Unix(st, 0).Sub(now).Hours() / float64(24))
				break
			}
			if now.Unix() >= st && now.Unix() < end && v.Status == 1 {
				findOut = true
			}
		}

		getSpeedGet := func(unLockPeriod int64) decimal.Decimal {
			var (
				getAmount decimal.Decimal
				number    int64 = 1
			)
			for _, v := range dList {
				end, _ := strconv.ParseInt(v.End, 10, 64)
				if now.Unix() < end && number <= unLockPeriod {
					amount, _ := decimal.NewFromString(v.Amount)
					getAmount = getAmount.Add(amount)
					number += 1
				}
			}
			return getAmount
		}

		var surplus int64
		var unLock int64
		for _, v := range dList {
			end, _ := strconv.ParseInt(v.End, 10, 64)
			if now.Unix() >= end {
				unLock += 1
			}
		}
		surplus = p.PeriodCount - unLock //lock period

		var nowUnLockPeriod int64 = 1
		switch p.Priority {
		case 2:
			nowUnLockPeriod = 5
		case 3:
			nowUnLockPeriod = 10
		case 5:
			nowUnLockPeriod = 20
		}
		rets.PerGet = getSpeedGet(nowUnLockPeriod).String()

		x5lock := decimal.NewFromInt(surplus).Div(decimal.NewFromInt(5))
		rets.X5Period = x5lock.Ceil().IntPart()
		rets.X5Get = getSpeedGet(5).String()

		x10lock := decimal.NewFromInt(surplus).Div(decimal.NewFromInt(10))
		rets.X10Period = x10lock.Ceil().IntPart()
		rets.X10Get = getSpeedGet(10).String()

		x20lock := decimal.NewFromInt(surplus).Div(decimal.NewFromInt(20))
		rets.X20Period = x20lock.Ceil().IntPart()
		rets.X20Get = getSpeedGet(20).String()

		rets.Surplus = surplus
		switch p.Priority {
		case 2:
			rets.Surplus = rets.X5Period
		case 3:
			rets.Surplus = rets.X10Period
		case 5:
			rets.Surplus = rets.X20Period
		}

	} else {
		zeroStr := decimal.Zero.String()
		rets.Lock = zeroStr
		rets.PerGet = zeroStr
		rets.X5Get = zeroStr
		rets.X10Get = zeroStr
		rets.X20Get = zeroStr
	}

	return rets, nil
}

func (p *AirdropInfo) GetAirdropBalance() (*AirdropBalanceResponse, error) {
	rets := &AirdropBalanceResponse{}
	f, err := p.Get(p.Account)
	if err != nil {
		return rets, err
	}
	if !f {
		return rets, nil
	}
	var dList []detailInfo
	err = json.Unmarshal([]byte(p.Detail), &dList)
	if err != nil {
		return rets, err
	}
	now := time.Now().Unix()
	rets.Lock = p.BalanceAmount
	if p.BalanceAmount.GreaterThan(decimal.Zero) {
		rets.Show = true
	}
	var amount decimal.Decimal
	for _, v := range dList {
		end, _ := strconv.ParseInt(v.End, 10, 64)
		if now >= end && v.Status == 1 {
			get, _ := decimal.NewFromString(v.Amount)
			amount = amount.Add(get)
		}
	}
	rets.Amount = amount

	return rets, nil
}

func getGenerateBlockTime() {
	var (
		p1 = &sqldb.PlatformParameter{}
		p2 = &sqldb.PlatformParameter{}
	)
	f, err := p1.Get(nil, "max_block_generation_time")
	if err != nil {
		log.Info("Get max block generation time failed:", err.Error())
		return
	}
	if f {
		maxGenerationTime, _ := strconv.ParseInt(p1.Value, 10, 64)
		if maxGenerationTime > 0 {
			maxGenerationTime = maxGenerationTime / 1000
		} else {
			log.Info("max block generation time invalid:", p1.Value)
			return
		}
		f, err = p2.Get(nil, "gap_between_blocks")
		if err != nil {
			log.Info("Get gap between blocks failed:", err.Error())
			return
		}
		if f {
			betweenTime, _ := strconv.ParseInt(p2.Value, 10, 64)
			if betweenTime == 0 {
				log.Info("gap between blocks time invalid:", p2.Value)
				return
			}
			//generateBlockTime = betweenTime + maxGenerationTime
		}
	}

}
