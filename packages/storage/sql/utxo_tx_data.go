package sql

import (
	"bytes"
	"encoding/hex"
	"github.com/IBAX-io/go-ibax/packages/transaction"
	"github.com/IBAX-io/go-ibax/packages/types"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TransactionData struct {
	Hash   []byte `gorm:"primary_key;not null;index"`
	Block  int64  `gorm:"not null;index"`
	Data   []byte `gorm:"not null"`
	TxTime int64  `gorm:"not null"`
	Type   int    `gorm:"not null"` //type 1:utxo transaction 0:contract transaction
}

var (
	getTransactionData chan bool
	txDataStart        bool = true
)

func (p *TransactionData) TableName() string {
	return "tx_data"
}

func (p *TransactionData) CreateTable() (err error) {
	err = nil
	if !HasTableOrView(p.TableName()) {
		if err = GetDB(nil).Migrator().CreateTable(p); err != nil {
			return err
		}
	}
	return err
}

func InitTransactionData() error {
	var p TransactionData
	err := p.CreateTable()
	if err != nil {
		return err
	}
	go TxDataSyncSignalReceive()

	return nil
}

func (p *TransactionData) GetByHash(hash []byte) (bool, error) {
	return isFound(GetDB(nil).Where("hash = ?", hash).First(p))
}

func (p *TransactionData) GetTxDataByHash(hash []byte) (bool, error) {
	return isFound(GetDB(nil).Select("tx_data,block").Where("hash = ?", hash).First(p))
}

func (p *TransactionData) GetLast() (bool, error) {
	return isFound(GetDB(nil).Order("tx_time desc").Take(p))
}

func (p *TransactionData) RollbackTransaction() error {
	return GetDB(nil).Where("block >= ?", p.Block).Delete(&TransactionData{}).Error
}

func (p *TransactionData) RollbackOne() error {
	if p.Block > 0 {
		err := p.RollbackTransaction()
		if err != nil {
			log.WithFields(log.Fields{"error": err, "block": p.Block}).Error("[transaction data] rollback one Failed")
			return err
		}
	}
	return nil
}

func TxDataSyncSignalReceive() {
	if getTransactionData == nil {
		getTransactionData = make(chan bool)
	}
	for {
		select {
		case <-getTransactionData:
			if err := transactionDataSync(); err != nil {
				log.WithFields(log.Fields{"error": err}).Error("Transaction Data Sync Failed")
			}
		}
	}
}

func SendTxDataSyncSignal() {
	select {
	case getTransactionData <- true:
	default:
		//fmt.Printf("Get Transaction Data len:%d,cap:%d\n", len(getTransactionData), cap(getTransactionData))
	}
}

func transactionDataSync() error {
	var insertData []TransactionData
	var b1 Block
begin:
	tr := &TransactionData{}
	_, err := tr.GetLast()
	if err != nil {
		return err
	}
	f, err := b1.GetMaxBlock()
	if err != nil {
		return err
	}
	if f {
		if txDataStart {
			err = tr.RollbackOne()
			if err != nil {
				return err
			} else {
				txDataStart = false
				goto begin
			}
		}
		if tr.Block >= b1.ID {
			transactionDataCheck(b1.ID)
			return nil
		}
	}

	bkList, err := GetBlockData(tr.Block, tr.Block+100, "asc")
	if err != nil {
		return err
	}
	if bkList == nil {
		return nil
	}
	for _, val := range *bkList {
		txList, err := UnmarshallBlockTxData(bytes.NewBuffer(val.Data), val.ID)
		if err != nil {
			return err
		}
		for _, data := range txList {
			if data.TxTime == 0 {
				var lg LogTransaction
				f, err = lg.GetTxTime(data.Hash)
				if err == nil && f {
					data.TxTime = lg.Timestamp
				} else {
					data.TxTime = val.Time * 1000
				}
			}
			insertData = append(insertData, data)
		}
	}
	err = createTransactionDataBatches(GetDB(nil), &insertData)
	if err != nil {
		return err
	}

	return transactionDataSync()
}

func transactionDataCheck(lastBlockId int64) {
	tran := &TransactionData{}
	f, err := tran.GetLast()
	if err == nil && f {
		logTran := &LogTransaction{}
		f, err = logTran.GetByHash(tran.Hash)
		if err == nil {
			if !f {
				if tran.Block > lastBlockId {
					tran.Block = lastBlockId
				}
				if tran.Block > 0 {
					log.WithFields(log.Fields{"log hash doesn't exist": hex.EncodeToString(tran.Hash), "block": tran.Block}).Info("rollback transaction data")
					err = tran.RollbackTransaction()
					if err == nil {
						transactionDataCheck(tran.Block)
					} else {
						log.WithFields(log.Fields{"error": err, "block": tran.Block}).Error("transaction Data rollback Failed")
					}
				}
			}
		} else {
			log.WithFields(log.Fields{"error": err, "hash": hex.EncodeToString(tran.Hash)}).Error("get log transaction failed")
		}
	}
}

func createTransactionDataBatches(dbTx *gorm.DB, data *[]TransactionData) error {
	if data == nil {
		return nil
	}
	return dbTx.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(data, 1000).Error
}

func UnmarshallBlockTxData(blockBuffer *bytes.Buffer, blockId int64) (map[string]TransactionData, error) {
	var (
		block = &types.BlockData{}
	)
	if err := block.UnmarshallBlock(blockBuffer.Bytes()); err != nil {
		return nil, err
	}

	txList := make(map[string]TransactionData)
	for i := 0; i < len(block.TxFullData); i++ {
		var info TransactionData

		tx, err := transaction.UnmarshallTransaction(bytes.NewBuffer(block.TxFullData[i]), false)
		if err != nil {
			return nil, err
		}
		info.Data = block.TxFullData[i]
		info.Hash = tx.Hash()
		info.Block = blockId

		if tx.IsSmartContract() {
			if tx.SmartContract().TxSmart.UTXO != nil || tx.SmartContract().TxSmart.TransferSelf != nil {
				info.Type = 1
			}
			info.TxTime = tx.Timestamp()
		} else {
			if blockId == 1 {
				info.Type = 1
			}
		}
		txList[hex.EncodeToString(tx.Hash())] = info
	}
	return txList, nil
}

func formatTxDataType(isUtxo bool) int {
	if isUtxo {
		return 1
	}
	return 0
}

func (p *TransactionData) GetFirstByType(blockId int64, txType int) (bool, error) {
	return isFound(GetDB(nil).Order("block asc").Where("type = ? AND block > ?", txType, blockId).Take(p))
}

func (p *TransactionData) GetLastByType(txType int) (bool, error) {
	return isFound(GetDB(nil).Where("type = ?", txType).Order("block desc,tx_time desc").Take(p))
}
