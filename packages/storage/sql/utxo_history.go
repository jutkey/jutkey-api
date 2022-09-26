package sql

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/IBAX-io/go-ibax/packages/consts"
	"github.com/IBAX-io/go-ibax/packages/transaction"
	"github.com/IBAX-io/go-ibax/packages/types"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UtxoHistory struct {
	Id               int64  `gorm:"primary_key;not null"`
	Block            int64  `gorm:"column:block;not null"`
	Hash             []byte `gorm:"column:hash;not null"`
	SenderId         int64  `gorm:"column:sender_id;not null"`
	RecipientId      int64  `gorm:"column:recipient_id;not null"`
	Amount           string `gorm:"column:amount;type:decimal(40);default:'0';not null"`
	CreatedAt        int64  `gorm:"column:created_at;not null"`
	Ecosystem        int64  `gorm:"not null"`
	Type             int    `gorm:"not null"` //1:UTXO_Transfer 2:UTXO_Tx
	SubType          int    `gorm:"not null"` //type is 1 then 1:AccountUTXO 2:UTXO-Account
	SenderBalance    string `gorm:"type:decimal(50);default:'0';not null"`
	RecipientBalance string `gorm:"type:decimal(50);default:'0';not null"`
}

type spentInfoTxData struct {
	OutputTxHash []byte
	BlockId      int64
	Time         int64
	Data         []byte
}

type utxoTxInfo struct {
	UtxoType     string
	TransferType string

	SenderId    int64
	RecipientId int64
	Amount      decimal.Decimal
	Ecosystem   int64
}

const (
	FeesType    = "fees"
	TaxesType   = "taxes"
	StartUpType = "startUp"

	AccountUTXO = "Account-UTXO"
	UTXOAccount = "UTXO-Account"
)

func (p *UtxoHistory) TableName() string {
	return "utxo_history"
}

func (p *UtxoHistory) CreateTable() (err error) {
	err = nil
	if !HasTableOrView(p.TableName()) {
		if err = GetDB(nil).Migrator().CreateTable(p); err != nil {
			return err
		}
	}
	return err
}

func (p *UtxoHistory) GetLast() (bool, error) {
	return isFound(GetDB(nil).Last(p))
}

func (p *UtxoHistory) RollbackTransaction() error {
	return GetDB(nil).Where("block > ?", p.Block).Delete(&UtxoHistory{}).Error
}

func InitSpentInfoHistory() error {
	var p UtxoHistory
	err := p.CreateTable()
	if err != nil {
		return err
	}

	return nil
}

func SpentInfoHistorySync() error {
	var insertData []UtxoHistory
	var (
		si     SpentInfo
		st     SpentInfo
		bkDiff int64
	)

	tr := &UtxoHistory{}
	_, err := tr.GetLast()
	if err != nil {
		return fmt.Errorf("[utxo sync]get spent info history last failed:%s", err.Error())
	}
	f, err := si.GetLast()
	if err != nil {
		return fmt.Errorf("[utxo sync]get spent info last failed:%s", err.Error())
	}
	if f {
		if tr.Block >= si.BlockId {
			utxoTxCheck(si.BlockId)
			return nil
		}
	}

	f, err = st.GetFirst(tr.Block)
	if err != nil {
		return fmt.Errorf("[utxo sync]get spent info block:%d first failed:%s", tr.Block, err.Error())
	}
	if !f {
		return nil
	}
	txList, err := getSpentInfoHashList(st.BlockId, st.BlockId+100)
	if err != nil {
		return fmt.Errorf("[utxo sync]get spent info hash list failed:%s", err.Error())
	}
	if txList == nil {
		return nil
	}

	//					map[ecosystem]map[key_id]balance
	keysBalance := make(map[int64]map[int64]decimal.Decimal)

	addInsertData := func(data UtxoHistory, amount decimal.Decimal) error {

		if data.Type != 5 {
			if _, ok := keysBalance[data.Ecosystem][data.SenderId]; !ok {
				return fmt.Errorf("[utxo sync] ecosystem:%d, senderId:%d balance doen't not exist", data.Ecosystem, data.SenderId)
			}
			if _, ok := keysBalance[data.Ecosystem][data.RecipientId]; !ok {
				return fmt.Errorf("[utxo sync] ecosystem:%d, recipientId:%d balance doen't not exist", data.Ecosystem, data.RecipientId)
			}

			if data.Type == 1 {
				if data.SubType == 2 {
					keysBalance[data.Ecosystem][data.SenderId] = keysBalance[data.Ecosystem][data.SenderId].Sub(amount)
					data.SenderBalance = keysBalance[data.Ecosystem][data.SenderId].String()
					data.RecipientBalance = keysBalance[data.Ecosystem][data.RecipientId].String()

					insertData = append(insertData, data)
					return nil
				}
			} else {
				keysBalance[data.Ecosystem][data.SenderId] = keysBalance[data.Ecosystem][data.SenderId].Sub(amount)
			}
		}

		keysBalance[data.Ecosystem][data.RecipientId] = keysBalance[data.Ecosystem][data.RecipientId].Add(amount)
		data.SenderBalance = keysBalance[data.Ecosystem][data.SenderId].String()
		data.RecipientBalance = keysBalance[data.Ecosystem][data.RecipientId].String()

		insertData = append(insertData, data)
		return nil
	}

	for _, val := range *txList {
		var (
			data       UtxoHistory
			outputList []SpentInfo
			si         SpentInfo
		)
		info, err := val.UnmarshalBlockTransaction()
		if err != nil {
			return fmt.Errorf("[utxo sync]unmarshal utxo transaction failed:%s", err.Error())
		}
		data.CreatedAt = val.Time
		data.Hash = val.OutputTxHash
		data.Block = val.BlockId

		if bkDiff != val.BlockId { //block diff update keys balance
			bkDiff = val.BlockId
			keys, err := si.GetOutputKeysByBlockId(val.BlockId)
			if err != nil {
				return fmt.Errorf("[utxo sync]get block:%d output keys failed:%s", val.BlockId, err.Error())
			}
			for _, k := range keys {
				var (
					s1 UtxoHistory
				)
				ba := make(map[int64]decimal.Decimal)

				balance, err := s1.GetKeyBalance(k.OutputKeyId, k.Ecosystem)
				if err != nil {
					return fmt.Errorf("[utxo sync]get ecosystem:%d output keys:%d balance failed:%s", k.Ecosystem, k.OutputKeyId, err.Error())
				}
				//fmt.Printf("[blockid]%d,ecosystem:%d,key_id:%d,balance:%s\n", val.BlockId, k.Ecosystem, k.OutputKeyId, balance.String())

				ba[k.OutputKeyId] = balance
				if _, ok := keysBalance[k.Ecosystem]; !ok {
					keysBalance[k.Ecosystem] = ba
				} else {
					keysBalance[k.Ecosystem][k.OutputKeyId] = balance
				}
			}
		}

		outputList, err = si.GetOutputs(val.OutputTxHash)
		if err != nil {
			return fmt.Errorf("[utxo sync]get out puts failed:%s", err.Error())
		}

		if info.UtxoType == UtxoTx {
			var (
				index       int
				indexSet    bool
				ecoCount    int
				ecoGasExist bool
			)

			for _, v := range outputList {
				if v.Ecosystem != 1 {
					ecoCount += 1
				}
			}
			if ecoCount >= 3 {
				ecoGasExist = true
			}

			for _, v := range outputList {
				amount, _ := decimal.NewFromString(v.OutputValue)
				recipientId := v.OutputKeyId
				if info.Ecosystem == 1 {
					if v.Ecosystem == 1 {
						switch index {
						case 0:
							data.Amount = amount.String()
							data.SenderId = info.SenderId
							data.RecipientId = recipientId
							data.Ecosystem = 1
							data.Type = formatSpentInfoHistoryType(FeesType)

							err = addInsertData(data, amount)
							if err != nil {
								return err
							}

							index += 1
						case 1:
							data.Amount = amount.String()
							data.SenderId = info.SenderId
							data.RecipientId = recipientId
							data.Ecosystem = 1
							data.Type = formatSpentInfoHistoryType(TaxesType)

							err = addInsertData(data, amount)
							if err != nil {
								return err
							}

							index += 1
						case 2:
							data.Amount = amount.String()
							data.SenderId = info.SenderId
							data.RecipientId = recipientId
							data.Ecosystem = 1
							data.Type = formatSpentInfoHistoryType(UtxoTx)

							err = addInsertData(data, amount)
							if err != nil {
								return err
							}

							index += 1
						case 3:
						}
					}
				} else {
					if v.Ecosystem == 1 {
						switch index {
						case 0:
							data.Amount = amount.String()
							data.SenderId = info.SenderId
							data.RecipientId = recipientId
							data.Ecosystem = 1
							data.Type = formatSpentInfoHistoryType(FeesType)

							err = addInsertData(data, amount)
							if err != nil {
								return err
							}

							index += 1
						case 1:
							data.Amount = amount.String()
							data.SenderId = info.SenderId
							data.RecipientId = recipientId
							data.Ecosystem = 1
							data.Type = formatSpentInfoHistoryType(TaxesType)

							err = addInsertData(data, amount)
							if err != nil {
								return err
							}

							index += 1
						case 2:
						}
					} else {
						if !indexSet {
							if ecoGasExist {
								index = 0
							} else {
								index = 2
							}
							indexSet = true
						}
						switch index {
						case 0:
							data.Amount = amount.String()
							data.SenderId = info.SenderId
							data.RecipientId = recipientId
							data.Ecosystem = v.Ecosystem
							data.Type = formatSpentInfoHistoryType(FeesType)

							err = addInsertData(data, amount)
							if err != nil {
								return err
							}

							index += 1
						case 1:

							data.Amount = amount.String()
							data.SenderId = info.SenderId
							data.RecipientId = recipientId
							data.Ecosystem = v.Ecosystem
							data.Type = formatSpentInfoHistoryType(TaxesType)

							err = addInsertData(data, amount)
							if err != nil {
								return err
							}

							index += 1
						case 2:
							data.Amount = amount.String()
							data.SenderId = info.SenderId
							data.RecipientId = recipientId
							data.Ecosystem = v.Ecosystem
							data.Type = formatSpentInfoHistoryType(UtxoTx)

							err = addInsertData(data, amount)
							if err != nil {
								return err
							}

							index += 1
						case 3:
						}
					}
				}
			}
		} else if info.UtxoType == StartUpType {
			var lt LogTransaction
			f, err := lt.GetByHash(val.OutputTxHash)
			if err != nil {
				return err
			}
			if !f {
				return fmt.Errorf("[utxo sync]get log hash doesn't exist hash:%s", hex.EncodeToString(val.OutputTxHash))
			}

			amount := decimal.New(consts.FounderAmount, int32(consts.MoneyDigits))
			data.Type = formatSpentInfoHistoryType(info.UtxoType)
			data.SenderId = 5555
			data.RecipientId = lt.Address
			data.Amount = amount.String()
			data.Ecosystem = lt.EcosystemID

			err = addInsertData(data, amount)
			if err != nil {
				return err
			}
		} else {
			data.Type = formatSpentInfoHistoryType(info.UtxoType)
			data.SubType = formatSpentInfoHistorySubType(info.TransferType)
			data.SenderId = info.SenderId
			data.RecipientId = info.RecipientId
			data.Amount = info.Amount.String()
			data.Ecosystem = info.Ecosystem

			err = addInsertData(data, info.Amount)
			if err != nil {
				return err
			}
		}

		err = createUtxoTxBatches(GetDB(nil), &insertData)
		if err != nil {
			return err
		}
		insertData = nil
	}

	return SpentInfoHistorySync()
}

func createUtxoTxBatches(dbTx *gorm.DB, data *[]UtxoHistory) error {
	if data == nil {
		return nil
	}
	return dbTx.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(data, 1000).Error
}

func (si *spentInfoTxData) UnmarshalBlockTransaction() (*utxoTxInfo, error) {
	var (
		block = &types.BlockData{}
	)
	blockBuffer := bytes.NewBuffer(si.Data)
	if err := block.UnmarshallBlock(blockBuffer.Bytes()); err != nil {
		return nil, err
	}
	for i := 0; i < len(block.TxFullData); i++ {
		tx, err := transaction.UnmarshallTransaction(bytes.NewBuffer(block.TxFullData[i]))
		if err != nil {
			return nil, err
		}
		if hex.EncodeToString(tx.Hash()) == hex.EncodeToString(si.OutputTxHash) {

			var result utxoTxInfo
			if tx.IsSmartContract() {
				result.Ecosystem = tx.SmartContract().TxSmart.Header.EcosystemID
				if tx.SmartContract().TxSmart.UTXO != nil {
					result.SenderId = tx.KeyID()
					result.UtxoType = UtxoTx
				} else if tx.SmartContract().TxSmart.TransferSelf != nil {
					result.UtxoType = UtxoTransfer
					result.SenderId = tx.KeyID()
					result.RecipientId = tx.KeyID()
					result.Amount, _ = decimal.NewFromString(tx.SmartContract().TxSmart.TransferSelf.Value)
					if tx.SmartContract().TxSmart.TransferSelf.Source == "Account" && tx.SmartContract().TxSmart.TransferSelf.Target == "UTXO" {
						result.TransferType = AccountUTXO
					} else {
						result.TransferType = UTXOAccount
					}
				} else {
					return &result, errors.New("doesn't not UTXO transaction")
				}
			} else {
				if si.BlockId == 1 {
					result.UtxoType = StartUpType
					return &result, nil
				}
				return nil, errors.New("doesn't not Smart Contract")
			}
			return &result, nil
		}
	}
	return nil, fmt.Errorf("doesn't not UTXO transaction from block id:%d,hash:%s\n", si.BlockId, hex.EncodeToString(si.OutputTxHash))
}

func formatSpentInfoHistoryType(utxoType string) int {
	switch utxoType {
	case UtxoTransfer:
		return 1
	case UtxoTx:
		return 2
	case FeesType:
		return 3
	case TaxesType:
		return 4
	case StartUpType:
		return 5
	}
	return 0
}

func parseSpentInfoHistoryType(utxoType int) string {
	switch utxoType {
	case 1:
		return UtxoTransfer
	case 2:
		return UtxoTx
	case 3:
		return FeesType
	case 4:
		return TaxesType
	case 5:
		return StartUpType
	}
	return ""
}

func formatSpentInfoHistorySubType(subType string) int {
	switch subType {
	case AccountUTXO:
		return 1
	case UTXOAccount:
		return 2
	}
	return 0
}

func parseSpentInfoHistorySubType(subType int) string {
	switch subType {
	case 1:
		return AccountUTXO
	case 2:
		return UTXOAccount
	}
	return ""
}

func compatibleContractAccountType(utxoType int) (contractType int) {
	switch utxoType {
	case 1:
	case 2:
		contractType = 3
	case 3:
		contractType = 1
	case 4:
		contractType = 2
	case 5:
	}
	return
}

func utxoTxCheck(lastBlockId int64) {
	tx := &UtxoHistory{}
	f, err := tx.GetLast()
	if err == nil && f {
		logTran := &LogTransaction{}
		f, err = logTran.GetByHash(tx.Hash)
		if err == nil {
			if !f {
				if tx.Block > lastBlockId {
					tx.Block = lastBlockId
				}
				if tx.Block > 0 {
					log.WithFields(log.Fields{"log hash doesn't exist": hex.EncodeToString(tx.Hash), "block": tx.Block}).Info("[utxo tx check] rollback data")
					tx.Block -= 1
					err = tx.RollbackTransaction()
					if err == nil {
						utxoTxCheck(tx.Block)
					} else {
						log.WithFields(log.Fields{"error": err, "block": tx.Block}).Error("[utxo tx check] rollback Failed")
					}
				}
			}
		} else {
			log.WithFields(log.Fields{"error": err, "hash": hex.EncodeToString(tx.Hash)}).Error("[utxo tx check] get log transaction failed")
		}
	}
}

func (p *UtxoHistory) GetKeyBalance(keyId int64, ecosystem int64) (balance decimal.Decimal, err error) {
	var f bool
	f, err = isFound(GetDB(nil).Raw(`
SELECT CASE WHEN sender_id = ? THEN
	sender_balance
ELSE
	recipient_balance
END AS balance
FROM utxo_history 
WHERE(recipient_id = ? OR sender_id = ?) AND ecosystem = ? ORDER BY id DESC LIMIT 1`, keyId, keyId, keyId, ecosystem).Take(&balance))
	if err != nil {
		return
	}
	if !f {
		return decimal.Zero, nil
	}

	return
}
