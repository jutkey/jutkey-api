package sql

import (
	"errors"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

// SpentInfo is model
type SpentInfo struct {
	InputTxHash  []byte `gorm:"default:(-)"`
	InputIndex   int32
	OutputTxHash []byte `gorm:"not null"`
	OutputIndex  int32  `gorm:"not null"`
	OutputKeyId  int64  `gorm:"not null"`
	OutputValue  string `gorm:"not null"`
	Scene        string
	Ecosystem    int64
	Contract     string
	BlockId      int64
	Asset        string
	Action       string `gorm:"-"` // UTXO operation control : change
}

const (
	UtxoTx       = "UTXO_Tx"
	UtxoTransfer = "UTXO_Transfer"
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
		return nil, errors.New("search len must be less than 11")
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
		Where("output_tx_hash = ?", txHash).Order("output_index ASC").Find(&list).Error
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
SELECT v1.block_id,v1.output_tx_hash,v2.time,v2.data FROM(
	SELECT output_tx_hash,block_id FROM spent_info WHERE block_id >= ? AND block_id < ? GROUP BY output_tx_hash,block_id ORDER BY block_id asc
)AS v1
LEFT JOIN(
	SELECT id,time,data FROM block_chain
)AS v2 ON(v2.id = v1.block_id)
 `, stId, endId).Find(&rlt).Error
	if err != nil {
		return nil, err
	}

	return &rlt, nil
}

func (p *SpentInfo) GetOutputKeysByBlockId(blockId int64) (outputKeys []SpentInfo, err error) {
	err = GetDB(nil).Select("output_key_id,ecosystem").Table(p.TableName()).
		Where("block_id = ?", blockId).Group("output_key_id,ecosystem").Find(&outputKeys).Error
	return
}
