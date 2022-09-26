package sql

import (
	"errors"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/vmihailenco/msgpack/v5"
	"gorm.io/gorm"
	"jutkey-server/packages/storage/kv"
)

type Statistics struct {
	Circulations    string `json:"circulations"`
	BlockId         int64  `json:"blockId"`
	EcosystemCount  int64  `json:"ecosystemCount"`
	NftMinerCount   int64  `json:"nftMinerCount"`
	NftMinerStaking string `json:"nftMinerStaking"`
}

func GetStatisticsData() error {
	var data Statistics
	tm, err := GetTotalAmount(1)
	if err != nil {
		return errors.New("Get Circulations failed:" + err.Error())
	}
	data.Circulations = tm.String()

	err = GetDB(nil).Table("info_block").Select("block_id").Take(&data.BlockId).Error
	if err != nil {
		return err
	}

	data.EcosystemCount, err = GetAllSystemCount()
	if err != nil {
		return err
	}

	if NftMinerReady {
		err := GetDB(nil).Table("1_nft_miner_items").Where("merge_status = ? ", 1).Count(&data.NftMinerCount).Error
		if err != nil {
			return err
		}
		err = GetDB(nil).Table("1_nft_miner_staking").Select("coalesce(sum(stake_amount),'0')as nft_miner_staking").
			Where("staking_status = 1").Take(&data.NftMinerStaking).Error
		if err != nil {
			if err != gorm.ErrRecordNotFound {
				return err
			}
		}
	} else {
		data.NftMinerStaking = "0"
	}

	err = data.SetRedis()
	if err != nil {
		return err
	}

	err = ParseChannel(ChannelDashboard, CmdStatistical, &data)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Send Websocket Statistics Data Failed")
	}
	return nil
}

func (m *Statistics) GetRedis() (bool, error) {
	rd := kv.RedisParams{
		Key:   "statistics",
		Value: "",
	}
	err := rd.Get()
	if err != nil {
		if err.Error() == "redis: nil" || err.Error() == "EOF" {
			return false, nil
		}
		return false, err
	}
	err = m.Unmarshal([]byte(rd.Value))
	if err != nil {
		return false, err
	}

	return true, err
}

func (m *Statistics) SetRedis() error {
	val, err := m.Marshal()
	if err != nil {
		return err
	}

	rd := kv.RedisParams{
		Key:   "statistics",
		Value: string(val),
	}
	err = rd.Set()
	if err != nil {
		return err
	}
	return nil
}

func (s *Statistics) Marshal() ([]byte, error) {
	if res, err := msgpack.Marshal(s); err != nil {
		return nil, err
	} else {
		return res, err
	}
}

func (s *Statistics) Unmarshal(bt []byte) error {
	if err := msgpack.Unmarshal(bt, &s); err != nil {
		return err
	}
	return nil
}

// GetKeysCount returns common count of keys
func GetTotalAmount(ecosystem int64) (decimal.Decimal, error) {
	var err error
	type result struct {
		Amount decimal.Decimal
	}
	var (
		res  result
		utxo SumAmount
	)
	err = GetDB(nil).Table("1_keys").
		Select("coalesce(sum(amount),0) as amount").Where("ecosystem = ?", ecosystem).Scan(&res).Error
	if err != nil {
		return decimal.Zero, err
	}
	err = GetDB(nil).Table("spent_info").Select("sum(output_value)").Where("input_tx_hash is NULL AND ecosystem = ?", ecosystem).Take(&utxo).Error
	if err != nil {
		return decimal.Zero, err
	}

	return res.Amount.Add(utxo.Sum), err
}

func GetAllSystemCount() (int64, error) {
	var eco Ecosystem
	var total int64

	if err := GetDB(nil).Table(eco.TableName()).Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func (p *Statistics) SendWebsocket(channel string, cmd string) error {
	return SendDataToWebsocket(channel, cmd, p)
}

func InitGlobalSwitch() {
	NodeReady = CandidateTableExist()
	NftMinerReady = NftMinerTableIsExist()
}
