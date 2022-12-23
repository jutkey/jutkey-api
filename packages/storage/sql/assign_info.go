package sql

import (
	"encoding/json"
	"errors"
	"github.com/shopspring/decimal"
	"strconv"
	"time"
	//"time"
)

// AssignInfo is model
type AssignInfo struct {
	ID            int64           `gorm:"primary_key;not null"`
	Type          int64           `gorm:"not null"`
	Account       string          `gorm:"not null"`
	TotalAmount   decimal.Decimal `gorm:"not null"`
	BalanceAmount decimal.Decimal `gorm:"not null"`
	Detail        string          `gorm:"not null;type:jsonb"`
	Deleted       int64           `gorm:"not null"`
	DateDeleted   int64           `gorm:"not null"`
	DateUpdated   int64           `gorm:"not null"`
	DateCreated   int64           `gorm:"not null"`
}

// TableName returns name of table
func (m AssignInfo) TableName() string {
	return `1_assign_info`
}

// GetId is retrieving model from database
func (m *AssignInfo) GetBalance(db *DbTransaction, account string) (bool, decimal.Decimal, decimal.Decimal, error) {

	var mps []AssignInfo
	var amount, balance decimal.Decimal
	amount = decimal.NewFromFloat(0)
	balance = decimal.NewFromFloat(0)
	if !HasTable(m) {
		return false, amount, balance, nil
	}
	err := GetDB(nil).Table(m.TableName()).
		Where("account =? AND deleted =? AND balance_amount > 0", account, 0).
		Find(&mps).Error
	if err != nil {
		return false, amount, balance, err
	}
	if len(mps) == 0 {
		return false, amount, balance, nil
	}

	//genesis time
	block := &Block{}
	genesisAt, err := block.GetSystemTime()
	if err != nil {
		return false, amount, balance, err
	}

	now := time.Now()
	for _, t := range mps {
		list, err := getAssignDetail(t.Detail, t.Type)
		if err != nil {
			return false, amount, balance, err
		}

		for _, v := range list {
			st, _ := strconv.ParseInt(v.StartAt, 10, 64)
			if st >= genesisAt && st <= now.Unix() && v.Status == 1 {
				am, _ := decimal.NewFromString(v.Amount)
				amount = amount.Add(am)
			}
		}
		balance = balance.Add(t.BalanceAmount)
	}
	return true, amount, balance, err
}

type assignDetail struct {
	Amount  string `json:"amount"`
	Status  int    `json:"status"`
	StartAt string `json:"startAt"`
	ClaimAt string `json:"claimAt"`
}

func getAssignDetail(detail string, assignType int64) ([]assignDetail, error) {
	var list []assignDetail
	switch assignType {
	case 1, 2, 3, 4, 5, 6:
		err := json.Unmarshal([]byte(detail), &list)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("not support assign type")
	}
	return list, nil

}
