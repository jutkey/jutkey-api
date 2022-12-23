package sql

import (
	"errors"
	"github.com/IBAX-io/go-ibax/packages/storage/sqldb"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"strconv"
)

// SpentInfo is model
type SpentInfo struct {
	InputTxHash  []byte `gorm:"default:(-)"`
	InputIndex   int32
	OutputTxHash []byte `gorm:"not null"`
	OutputIndex  int32  `gorm:"not null"`
	OutputKeyId  int64  `gorm:"not null"`
	OutputValue  string `gorm:"not null"`
	Ecosystem    int64
	BlockId      int64
	Type         int64
}

const (
	UtxoTx           = "UTXO_Tx"
	UtxoTransferSelf = "UTXO_Transfer_Self"
)

// TableName returns name of table
func (si *SpentInfo) TableName() string {
	return "spent_info"
}

func (si *SpentInfo) GetAmountByKeyId(keyId int64, ecosystem int64) (decimal.Decimal, error) {
	var utxoAmount SumAmount
	f, err := isFound(GetDB(nil).Table(si.TableName()).
		Select("coalesce(sum(output_value),'0') as sum").Where("input_tx_hash is NULL AND output_key_id = ? AND ecosystem = ?", keyId, ecosystem).
		Take(&utxoAmount))
	if err != nil {
		return decimal.Zero, err
	}
	if !f {
		return decimal.Zero, nil
	}
	return utxoAmount.Sum, nil
}

func GetUtxoInput(keyId int64, search []int64) (*[]UtxoInputResponse, error) {
	var (
		rets []UtxoInputResponse
		si   SpentInfo
		err  error
	)

	if len(search) > 10 {
		return nil, errors.New("search max len 10")
	}

	err = GetDB(nil).Table(si.TableName()).Select("count(1) AS input,ecosystem").
		Where("input_tx_hash is NULL AND output_key_id = ? AND ecosystem IN(?)", keyId, search).Group("ecosystem").Find(&rets).Error
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Warn("Get Utxo Input Failed")
		return nil, err
	}
	for _, v1 := range search {
		var findOut bool
		for _, v2 := range rets {
			if v2.Ecosystem == v1 {
				findOut = true
			}
		}
		if !findOut {
			rets = append(rets, UtxoInputResponse{Ecosystem: v1, Input: 0})
		}
	}

	fuels := GetFuelRate()
	for k, val := range rets {
		if val.Ecosystem != 1 {
			state := &sqldb.StateParameter{}
			state.SetTablePrefix(strconv.FormatInt(val.Ecosystem, 10))
			f, _ := state.Get(nil, "utxo_fee")
			if !f {
				val.FuelRate = decimal.Zero.String()
				rets[k] = val
				continue
			} else {
				if state.Value != "1" {
					val.FuelRate = decimal.Zero.String()
					rets[k] = val
					continue
				}
			}
		}
		if _, ok := fuels[val.Ecosystem]; ok {
			val.FuelRate = fuels[val.Ecosystem].String()
		} else {
			val.FuelRate = decimal.Zero.String()
		}
		rets[k] = val
	}

	return &rets, nil
}

func (si *SpentInfo) GetOutputs(txHash []byte) (list []SpentInfo, err error) {
	err = GetDB(nil).Table(si.TableName()).
		Where("output_tx_hash = ?", txHash).Order("ecosystem ASC").Find(&list).Error
	return
}

func (si *SpentInfo) GetLast() (bool, error) {
	return isFound(GetDB(nil).Order("block_id desc").Take(si))
}

func (si *SpentInfo) GetFirst(blockId int64) (bool, error) {
	return isFound(GetDB(nil).Order("block_id asc").Where("block_id > ?", blockId).Take(si))
}

func getSpentInfoHashList(stId, endId int64) (*[]spentInfoTxData, error) {
	var (
		err error
	)
	var rlt []spentInfoTxData

	err = GetDB(nil).Raw(`
SELECT v1.block_id,v1.hash,v2.data FROM(
	SELECT output_tx_hash AS hash,block_id FROM spent_info WHERE block_id >= ? AND block_id < ? GROUP BY output_tx_hash,block_id
		UNION
	SELECT input_tx_hash AS hash,lg.block AS block_id FROM spent_info AS s1 LEFT JOIN log_transactions AS lg ON(lg.hash = s1.input_tx_hash) 
	WHERE block >= ? AND block < ? AND input_tx_hash IS NOT NULL GROUP BY input_tx_hash,block
)AS v1
LEFT JOIN(
	SELECT hash,data FROM tx_data WHERE block >= ? AND block < ?
)AS v2 ON(v2.hash = v1.hash)
ORDER BY block_id asc
`, stId, endId, stId, endId, stId, endId).Find(&rlt).Error
	if err != nil {
		return nil, err
	}

	return &rlt, nil
}

func (p *SpentInfo) GetOutputKeysByBlockId(blockId int64) (outputKeys []SpentInfo, err error) {
	err = GetDB(nil).Raw(`
SELECT output_key_id,ecosystem FROM "spent_info" WHERE block_id = ? GROUP BY output_key_id,ecosystem
UNION
SELECT v2.address AS output_key_id,v2.ecosystem_id AS ecosystem 
FROM tx_data AS v1 LEFT JOIN log_transactions AS v2 ON(v2.hash = v1.hash) 
WHERE v1.block = ? AND v1.type = 1 GROUP BY output_key_id,ecosystem
`, blockId, blockId).Find(&outputKeys).Error
	return
}
