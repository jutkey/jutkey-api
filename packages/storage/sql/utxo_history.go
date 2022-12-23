package sql

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/IBAX-io/go-ibax/packages/consts"
	"github.com/IBAX-io/go-ibax/packages/transaction"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var initStart = true

type UtxoHistory struct {
	Id               int64  `gorm:"primary_key;not null"`
	Block            int64  `gorm:"column:block;not null;index"`
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
	Hash    []byte
	BlockId int64
	Data    []byte
}

type utxoTxInfo struct {
	UtxoType     string
	TransferType string
	TxTime       int64

	SenderId    int64
	RecipientId int64
	Amount      decimal.Decimal
	Ecosystem   int64
}

const (
	FeesType       = "fees"
	TaxesType      = "taxes"
	StartUpType    = "startUp"
	CombustionType = "combustion"

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
	return GetDB(nil).Where("block >= ?", p.Block).Delete(&UtxoHistory{}).Error
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
		si     TransactionData
		st     TransactionData
		bkDiff int64
	)
	tr := &UtxoHistory{}
	_, err := tr.GetLast()
	if err != nil {
		return fmt.Errorf("[utxo sync]get spent info history last failed:%s", err.Error())
	}
	f, err := si.GetLastByType(formatTxDataType(true))
	if err != nil {
		return fmt.Errorf("[utxo sync]get spent info last failed:%s", err.Error())
	}
	if f {
		if initStart {
			err = tr.RollbackOne()
			if err != nil {
				return err
			} else {
				initStart = false
			}
		}
		if tr.Block >= si.Block {
			utxoTxCheck(si.Block)
			return nil
		}
	}

	f, err = st.GetFirstByType(tr.Block, formatTxDataType(true))
	if err != nil {
		return fmt.Errorf("[utxo sync]get spent info block:%d first failed:%s", tr.Block, err.Error())
	}
	if !f {
		return nil
	}
	txList, err := getSpentInfoHashList(st.Block, st.Block+100)
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
		} else {
			ba := make(map[int64]decimal.Decimal)

			ba[data.Ecosystem] = decimal.Zero
			if _, ok := keysBalance[data.Ecosystem]; !ok {
				keysBalance[data.Ecosystem] = ba
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
			data UtxoHistory
			s1   SpentInfo
		)
		data.Hash = val.Hash
		data.Block = val.BlockId

		if bkDiff != val.BlockId { //block diff update keys balance
			bkDiff = val.BlockId

			if insertData != nil {
				err = createUtxoTxBatches(GetDB(nil), &insertData)
				if err != nil {
					return err
				}
				insertData = nil
			}

			keys, err := s1.GetOutputKeysByBlockId(val.BlockId)
			if err != nil {
				return fmt.Errorf("[utxo sync]get block:%d output keys failed:%s", val.BlockId, err.Error())
			}
			for _, k := range keys {
				var (
					his UtxoHistory
				)
				ba := make(map[int64]decimal.Decimal)

				balance, err := his.GetKeyBalance(k.OutputKeyId, k.Ecosystem)
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

		info, err := val.UnmarshalBlockTransaction()
		if err != nil {
			return fmt.Errorf("[utxo sync]unmarshal utxo transaction failed:%s", err.Error())
		}
		data.CreatedAt = info.TxTime

		if info.UtxoType == UtxoTx {
			var (
				outputList []SpentInfo
				si         SpentInfo
			)
			outputList, err = si.GetOutputs(val.Hash)
			if err != nil {
				return fmt.Errorf("[utxo sync]get out puts failed:%s", err.Error())
			}
			for _, v := range outputList {
				amount, _ := decimal.NewFromString(v.OutputValue)
				recipientId := v.OutputKeyId
				if info.Ecosystem == 1 {
					if v.Ecosystem == 1 {
						switch v.Type {
						case 20: //fees type
							data.Amount = amount.String()
							data.SenderId = info.SenderId
							data.RecipientId = recipientId
							data.Ecosystem = 1
							data.Type = formatSpentInfoHistoryType(FeesType)

							err = addInsertData(data, amount)
							if err != nil {
								return err
							}

						case 21: //taxes type
							data.Amount = amount.String()
							data.SenderId = info.SenderId
							data.RecipientId = recipientId
							data.Ecosystem = 1
							data.Type = formatSpentInfoHistoryType(TaxesType)

							err = addInsertData(data, amount)
							if err != nil {
								return err
							}

						case 26: //utxo tx
							data.Amount = amount.String()
							data.SenderId = info.SenderId
							data.RecipientId = recipientId
							data.Ecosystem = 1
							data.Type = formatSpentInfoHistoryType(UtxoTx)

							err = addInsertData(data, amount)
							if err != nil {
								return err
							}
						}
					}
				} else {
					if v.Ecosystem == 1 {
						switch v.Type {
						case 20: //fees type
							data.Amount = amount.String()
							data.SenderId = info.SenderId
							data.RecipientId = recipientId
							data.Ecosystem = 1
							data.Type = formatSpentInfoHistoryType(FeesType)

							err = addInsertData(data, amount)
							if err != nil {
								return err
							}

						case 21: //taxes type
							data.Amount = amount.String()
							data.SenderId = info.SenderId
							data.RecipientId = recipientId
							data.Ecosystem = 1
							data.Type = formatSpentInfoHistoryType(TaxesType)

							err = addInsertData(data, amount)
							if err != nil {
								return err
							}

						}
					} else {
						switch v.Type {
						case 23: //combustion type
							data.Amount = amount.String()
							data.SenderId = info.SenderId
							data.RecipientId = recipientId
							data.Ecosystem = v.Ecosystem
							data.Type = formatSpentInfoHistoryType(CombustionType)

							err = addInsertData(data, amount)
							if err != nil {
								return err
							}
						case 20: //fees type
							data.Amount = amount.String()
							data.SenderId = info.SenderId
							data.RecipientId = recipientId
							data.Ecosystem = v.Ecosystem
							data.Type = formatSpentInfoHistoryType(FeesType)

							err = addInsertData(data, amount)
							if err != nil {
								return err
							}

						case 21: //taxes type

							data.Amount = amount.String()
							data.SenderId = info.SenderId
							data.RecipientId = recipientId
							data.Ecosystem = v.Ecosystem
							data.Type = formatSpentInfoHistoryType(TaxesType)

							err = addInsertData(data, amount)
							if err != nil {
								return err
							}

						case 26: //utxo tx type
							data.Amount = amount.String()
							data.SenderId = info.SenderId
							data.RecipientId = recipientId
							data.Ecosystem = v.Ecosystem
							data.Type = formatSpentInfoHistoryType(UtxoTx)

							err = addInsertData(data, amount)
							if err != nil {
								return err
							}
						}
					}
				}
			}
		} else if info.UtxoType == StartUpType {
			var lt LogTransaction
			f, err := lt.GetByHash(val.Hash)
			if err != nil {
				return err
			}
			if !f {
				return fmt.Errorf("[utxo sync]get log hash doesn't exist hash:%s", hex.EncodeToString(val.Hash))
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
	}
	if insertData != nil {
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
	return dbTx.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(data, 5000).Error
}

func (si *spentInfoTxData) UnmarshalBlockTransaction() (*utxoTxInfo, error) {
	if si.Data == nil {
		return nil, errors.New("transaction Data is null")
	}

	var (
		result utxoTxInfo
	)
	lt := &LogTransaction{}

	tx, err := transaction.UnmarshallTransaction(bytes.NewBuffer(si.Data), false)
	if err != nil {
		return nil, err
	}

	if tx.IsSmartContract() {
		result.Ecosystem = tx.SmartContract().TxSmart.Header.EcosystemID
		if tx.SmartContract().TxSmart.UTXO != nil {
			result.SenderId = tx.KeyID()
			result.UtxoType = UtxoTx
		} else if tx.SmartContract().TxSmart.TransferSelf != nil {
			result.UtxoType = UtxoTransferSelf
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
		result.TxTime = tx.Timestamp()

		if result.TxTime == 0 {
			f, err := lt.GetTxTime(si.Hash)
			if err == nil && f {
				result.TxTime = lt.Timestamp
			}
			result.TxTime = lt.Timestamp
		}
	} else {
		if si.BlockId == 1 {
			result.UtxoType = StartUpType
			f, err := lt.GetTxTime(si.Hash)
			if err == nil && f {
				result.TxTime = lt.Timestamp
			}
			return &result, nil
		}
		return nil, errors.New("doesn't not Smart Contract")
	}
	return &result, nil
}

func formatSpentInfoHistoryType(utxoType string) int {
	switch utxoType {
	case UtxoTransferSelf:
		return 1
	case UtxoTx:
		return 2
	case FeesType:
		return 3
	case TaxesType:
		return 4
	case StartUpType:
		return 5
	case CombustionType:
		return 6
	}
	return 0
}

func parseSpentInfoHistoryType(utxoType int) string {
	switch utxoType {
	case 1:
		return UtxoTransferSelf
	case 2:
		return UtxoTx
	case 3:
		return FeesType
	case 4:
		return TaxesType
	case 5:
		return StartUpType
	case 6:
		return CombustionType
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
	case 6:
		contractType = 16
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

func (p *UtxoHistory) RollbackOne() error {
	if p.Block > 0 {
		err := p.RollbackTransaction()
		if err != nil {
			log.WithFields(log.Fields{"error": err, "block": p.Block}).Error("[rollback one] rollback Failed")
			return err
		}
	}
	return nil
}
