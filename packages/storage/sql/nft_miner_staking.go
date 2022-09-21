package sql

import (
	"encoding/json"
	"errors"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"reflect"
	"time"
)

type NftMinerStaking struct {
	ID            int64  `gorm:"primary_key;not null"`         //ID
	TokenId       int64  `gorm:"column:token_id;not null"`     //NFT Miner ID
	StakeAmount   string `gorm:"column:stake_amount;not null"` //starking
	EnergyPower   int64  `gorm:"column:energy_power;not null"`
	EnergyPoint   int64  `gorm:"column:energy_point;not null"`
	Source        int64  `gorm:"column:source;not null"`      //source
	StartDated    int64  `gorm:"column:start_dated;not null"` //start time
	EndDated      int64  `gorm:"column:end_dated;not null"`   //end time
	Staker        string `gorm:"column:staker;not null"`      //owner account
	StakingStatus int64  `gorm:"column:staking_status;not null"`
	WithdrawDate  int64  `gorm:"column:withdraw_date;not null"` //withdraw time

}

func (p *NftMinerStaking) TableName() string {
	return "1_nft_miner_staking"
}

func (p *NftMinerStaking) GetNftMinerStakeInfo(search any, page, limit int, order, wallet string) (*GeneralResponse, error) {
	var (
		rets  []NftMinerStakeInfoResponse
		total int64
		ret   GeneralResponse
		nftId int64
	)
	if order == "" {
		order = "id desc"
	}

	switch reflect.TypeOf(search).String() {
	case "string":
		var item NftMinerItems
		f, err := item.GetByTokenHash(search.(string))
		if err != nil {
			log.WithFields(log.Fields{"search type": reflect.TypeOf(search).String()}).Warn("Get Nft Miner Stake Info  Failed")
			return nil, err
		}
		if !f {
			return nil, errors.New("NFT Miner Doesn't Not Exist")
		}
		nftId = item.ID
	case "json.Number":
		tokenId, err := search.(json.Number).Int64()
		if err != nil {
			return nil, err
		}
		nftId = tokenId
	default:
		log.WithFields(log.Fields{"search type": reflect.TypeOf(search).String()}).Warn("Get Nft Miner Stake Info Search Failed")
		return nil, errors.New("request params invalid")
	}

	err := GetDB(nil).Table(p.TableName()).Where("token_id = ? AND staker = ?", nftId, wallet).Count(&total).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Info("get nft Miner stake info total err:", err.Error(), " nftId:", nftId)
		}
		return nil, err
	}

	err = GetDB(nil).Raw(`SELECT id,token_id AS nft_miner_id,start_dated,end_dated,stake_amount,date_part('day',cast(to_char(to_timestamp(end_dated),'yyyy-MM-dd') as TIMESTAMP)-cast(to_char(to_timestamp(start_dated),'yyyy-MM-dd') as TIMESTAMP)) 
	AS cycle FROM "1_nft_miner_staking" WHERE token_id = ? AND staker = ?`+"ORDER BY "+order+` offset ? limit ?`, nftId, wallet, (page-1)*limit, limit).Find(&rets).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Info("get nft Miner stake info nftStaking err:", err.Error(), " nft Miner Id:", nftId)
		}
		return nil, err
	}

	nowTime := time.Now().Unix()
	for i := 0; i < len(rets); i++ {
		if nowTime >= rets[i].StartDated && nowTime <= rets[i].EndDated {
			rets[i].StakeStatus = true
		}
	}

	ret.Total = total
	ret.Page = page
	ret.Limit = limit
	ret.List = rets

	return &ret, nil
}
