package sql

import (
	"encoding/hex"
	"errors"
	"github.com/IBAX-io/go-ibax/packages/consts"
	"github.com/IBAX-io/go-ibax/packages/converter"
	"github.com/IBAX-io/go-ibax/packages/storage/sqldb"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"jutkey-server/conf"
	"jutkey-server/packages/params"
	"strconv"
	"time"
)

// History represent record of history table
type History struct {
	ID               int64           `gorm:"primary_key;not null"`
	SenderId         int64           `gorm:"column:sender_id;not null"`
	RecipientId      int64           `gorm:"column:recipient_id;not null"`
	SenderBalance    decimal.Decimal `gorm:"column:sender_balance;not null"`
	RecipientBalance decimal.Decimal `gorm:"column:recipient_balance;not null"`
	Amount           decimal.Decimal `gorm:"column:amount;not null"`
	ValueDetail      string          `gorm:"column:value_detail;not null"`
	Comment          string          `gorm:"column:comment;not null"`
	BlockId          int64           `gorm:"column:block_id;not null"`
	Txhash           []byte          `gorm:"column:txhash;not null"`
	CreatedAt        int64           `gorm:"column:created_at;not null"`
	Ecosystem        int64           `gorm:"not null"`
	Type             int64           `gorm:"not null"`
	Status           int32           `gorm:"not null"`
}

type WalletHistoryHex struct {
	Transaction int64           `json:"transaction"`
	InTx        int64           `json:"in_tx"`
	OutTx       int64           `json:"out_tx"`
	Inamount    decimal.Decimal `json:"inamount"`
	Outamount   decimal.Decimal `json:"outamount"`
	Amount      decimal.Decimal `json:"amount,omitempty"`
}

// TableName returns name of table
func (th *History) TableName() string {
	return "1_history"
}

func (th *History) GetMonthFind(req *params.HistoryFindForm) (*WalletMonthDetailResponse, error) {
	if req.Time <= 0 {
		return nil, errors.New("invalid request parameter time")
	}
	rs := new([]History)
	var rets WalletMonthDetailResponse
	rets.Limit = req.Limit
	rets.Page = req.Page
	kid := converter.StringToAddress(req.Wallet)
	stTime := req.Time
	te := time.Unix(req.Time, 0).AddDate(0, 1, 0).Unix()
	if err := GetDB(nil).Table(th.TableName()).
		Where("(sender_id = ? or recipient_id = ? ) and ecosystem = ? and created_at >= ? and created_at < ?",
			kid, kid, req.Ecosystem, time.Unix(stTime, 0).UnixMilli(), time.Unix(te, 0).UnixMilli()).
		Count(&rets.Total).Error; err != nil {
		return nil, err
	}
	if err := GetDB(nil).Where("(sender_id = ? or recipient_id = ? ) and ecosystem = ? and created_at >= ? and created_at < ?",
		kid, kid, req.Ecosystem, time.Unix(stTime, 0).UnixMilli(), time.Unix(te, 0).UnixMilli()).
		Order(req.Order).
		Offset((req.Page - 1) * req.Limit).
		Limit(req.Limit).
		Find(rs).Error; err != nil {
		return nil, err
	}

	rets.List = *th.ChangeMonthResults(rs, kid)
	rets.TokenSymbol, _ = GetEcosystemTokenSymbol(req.Ecosystem)

	return &rets, nil
}

func (th *History) ChangeMonthResults(vers *[]History, kid int64) *[]HistoryMonthRet {
	var dats []HistoryMonthRet
	for _, t := range *vers {
		s := t.ChangeMonthResult(kid)
		dats = append(dats, *s)
	}
	return &dats
}

func (th *History) ChangeMonthResult(kid int64) *HistoryMonthRet {
	var balance string
	if kid == th.SenderId {
		balance = th.SenderBalance.String()
	} else if kid == th.RecipientId {
		balance = th.RecipientBalance.String()
	}
	s := HistoryMonthRet{
		Ecosystem: th.Ecosystem,
		ID:        th.ID,
		Sender:    converter.AddressToString(th.SenderId),
		Recipient: converter.AddressToString(th.RecipientId),
		Balance:   balance,
		Amount:    th.Amount.String(),
		Comment:   th.Comment,
		BlockId:   th.BlockId,
		TxHash:    hex.EncodeToString(th.Txhash),
		Time:      MsToSeconds(th.CreatedAt),
		Type:      th.Type,
	}
	return &s
}

func (th *History) GetWalletMonthHistoryTotals(eid, kid int64, month int) ([]WalletMonthHistory, error) {
	var (
		list []WalletMonthHistory
		err  error
	)
	now := time.Now()
	t := GetZeroTime(now.AddDate(0, 0, -now.Day()+1))
	//t := GetFirstDateOfMonth(-z, time.Now())
	for i := 0; i < month; i++ {
		te := t
		te = te.AddDate(0, 1, 0)
		if dat, err := th.GetWalletMonthHistory(eid, kid, t.UnixMilli(), te.UnixMilli()); err == nil {
			dat.Month = t.Month().String()
			dat.Time = t.Unix()
			list = append(list, *dat)
		} else {
			return list, err
		}
		t = t.AddDate(0, -1, 0)
	}

	return list, err
}

func (th *History) GetWalletMonthHistory(eid, keyid, t1, t2 int64) (*WalletMonthHistory, error) {
	var (
		ret    WalletMonthHistory
		scount int64
		rcount int64
		in     string
		out    string
		err    error
	)

	err = GetDB(nil).Table("1_history").
		Where("recipient_id = ? and ecosystem = ? and type != ? and created_at >= ? and created_at < ?", keyid, eid, 13, t1, t2).
		Count(&rcount).Error
	if err != nil {
		return &ret, err
	}
	if rcount > 0 {
		err = GetDB(nil).Table("1_history").Select("sum(amount)").Where("recipient_id = ? and ecosystem = ? and type != ? and created_at >= ? and created_at < ?", keyid, eid, 13, t1, t2).Row().Scan(&in)
		if err != nil {
			return &ret, err
		}
	} else {
		in = "0"
	}

	err = GetDB(nil).Table("1_history").
		Where("sender_id = ? and ecosystem = ? and type != ? and created_at >= ? and created_at < ?", keyid, eid, 13, t1, t2).
		Count(&scount).Error
	if err != nil {
		return &ret, err
	}
	if scount > 0 {
		err = GetDB(nil).Table("1_history").Select("sum(amount)").Where("sender_id = ? and ecosystem = ? and type != ? and created_at >= ? and created_at < ?", keyid, eid, 13, t1, t2).Row().Scan(&out)
		if err != nil {
			return &ret, err
		}
	} else {
		out = "0"
	}

	din, err := decimal.NewFromString(in)
	if err != nil {
		return &ret, err
	}
	dout, err := decimal.NewFromString(out)
	if err != nil {
		return &ret, err
	}

	ret.OutCount = scount
	ret.OutAmount = dout
	ret.InCount = rcount
	ret.InAmount = din

	return &ret, err
}

func (th *History) GetDBDayNftInComeinfo(kid, t1, t2 int64) (string, error) {
	type allAmount struct {
		Amounts string `gorm:"column:amounts"`
	}
	var at allAmount
	err := GetDB(nil).Table(th.TableName()).Select("sum(amount) amounts").Where(`type = ? and recipient_id = ? and created_at >= ? and created_at < ?`, 12, kid, t1, t2).Take(&at).Error
	//.Row().Scan(&allAmount)

	if err != nil {
		return "", err
	}
	return at.Amounts, err
}

func (th *History) GetList(c *params.MineHistoryRequest) (*GeneralResponse, error) {
	var rets GeneralResponse
	var list []map[string]string
	kid := converter.StringToAddress(c.Wallet)
	q := GetDB(nil).Table(th.TableName()).Select("id,sender_id,recipient_id,created_at,status,block_id,type,ecosystem,txhash,amount").Where("ecosystem = ?", c.Ecosystem)
	switch c.Opt {
	case "send":
		q = q.Where("sender_id = ?", kid)
	case "recipient":
		q = q.Where("recipient_id = ?", kid)
	case "all":
		q = q.Where("recipient_id = ? OR sender_id = ? ", kid, kid)
	}
	err := q.Count(&rets.Total).Error
	if err != nil {
		return nil, err
	}
	q = q.Order(c.Order)
	rows, err := q.Offset((c.Page - 1) * c.Limit).Limit(c.Limit).Rows()
	if err != nil {
		return nil, err
	}
	list, err = sqldb.GetNodeResult(rows)
	if err != nil {
		return nil, err
	}
	tokenSymbol, _ := GetEcosystemTokenSymbol(c.Ecosystem)

	for key, val := range list {
		list[key]["token_symbol"] = tokenSymbol
		if createdAt, ok := val["created_at"]; ok {
			t1, _ := strconv.ParseInt(createdAt, 10, 64)
			t2 := strconv.FormatInt(MsToSeconds(t1), 10)
			list[key]["created_at"] = t2
		}
		if keyIdStr, ok := val["sender_id"]; ok {
			keyId, _ := strconv.ParseInt(keyIdStr, 10, 64)
			wallet := converter.AddressToString(keyId)
			delete(list[key], "sender_id")
			list[key]["sender"] = wallet
			if wallet != c.Wallet {
				list[key]["address"] = wallet
			}
		}
		if keyIdStr, ok := val["recipient_id"]; ok {
			keyId, _ := strconv.ParseInt(keyIdStr, 10, 64)
			wallet := converter.AddressToString(keyId)
			delete(list[key], "recipient_id")
			list[key]["recipient"] = wallet
			if wallet != c.Wallet {
				list[key]["address"] = wallet
			}
		}
		if _, ok := list[key]["address"]; !ok {
			list[key]["address"] = c.Wallet
		}

		if hashStr, ok := val["txhash"]; ok {
			delete(list[key], "txhash")
			hash, _ := hex.DecodeString(hashStr)
			list[key]["contract"] = GetContractName(hash)
		}
	}
	rets.List = list
	rets.Page = c.Page
	rets.Limit = c.Limit
	return &rets, nil
}

func (th *History) GetAccountHistoryTotals(id int64, keyId int64) (*WalletHistoryHex, error) {
	var (
		ret    WalletHistoryHex
		scount int64
		rcount int64
		in     string
		out    string
		err    error
	)

	err = conf.GetDbConn().Conn().Table("1_history").
		Where("recipient_id = ? and ecosystem = ?", keyId, id).
		Count(&rcount).Error
	if err != nil {
		return &ret, err
	}
	if rcount > 0 {
		err = conf.GetDbConn().Conn().Table("1_history").Select("sum(amount)").Where("recipient_id = ? and ecosystem = ?", keyId, id).Row().Scan(&in)
		if err != nil {
			return &ret, err
		}
	} else {
		in = "0"
	}

	err = conf.GetDbConn().Conn().Table("1_history").
		Where("sender_id = ? and ecosystem = ?", keyId, id).
		Count(&scount).Error
	if err != nil {
		return &ret, err
	}
	if scount > 0 {
		err = conf.GetDbConn().Conn().Table("1_history").Select("sum(amount)").Where("sender_id = ? and ecosystem = ?", keyId, id).Row().Scan(&out)
		if err != nil {
			return &ret, err
		}
	} else {
		out = "0"
	}

	din, err := decimal.NewFromString(in)
	if err != nil {
		return &ret, err
	}
	dout, err := decimal.NewFromString(out)
	if err != nil {
		return &ret, err
	}
	ret.InTx = rcount
	ret.OutTx = scount
	ret.Transaction = scount + rcount
	ret.Inamount = din
	ret.Outamount = dout

	return &ret, err
}

func GetAccountHistoryTotal(account string, ecosystem int64) (*AccountHistoryTotal, error) {
	var rets AccountHistoryTotal
	kid := converter.StringToAddress(account)
	ts := &History{}
	dh, err := ts.GetAccountHistoryTotals(ecosystem, kid)
	if err != nil {
		return nil, err
	}
	rets.InAmount = dh.Inamount.String()
	rets.OutAmount = dh.Outamount.String()
	rets.AllAmount = dh.Inamount.Add(dh.Outamount).String()
	rets.InTx = dh.InTx
	rets.OutTx = dh.OutTx
	rets.AllTx = dh.Transaction
	tokenSymbol, _ := GetEcosystemTokenSymbol(ecosystem)
	rets.TokenSymbol = tokenSymbol
	rets.Ecosystem = ecosystem

	return &rets, nil
}

func GetContractName(txHash []byte) string {
	var tx LogTransaction
	err := GetDB(nil).Select("contract_name").Where("hash = ?", txHash).Take(&tx).Error
	if err != nil {
		log.WithFields(log.Fields{"type": consts.ConversionError, "hash": txHash}).Error("Get Contract Name Failed")
		return ""
	}
	return tx.ContractName
}
