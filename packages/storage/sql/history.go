package sql

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/IBAX-io/go-ibax/packages/converter"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
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
	InAmount    decimal.Decimal `json:"inamount"`
	OutAmount   decimal.Decimal `json:"outamount"`
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
	var rets WalletMonthDetailResponse
	rets.Limit = req.Limit
	rets.Page = req.Page
	kid := converter.StringToAddress(req.Wallet)
	stTime := time.Unix(req.Time, 0)
	edTime := stTime.AddDate(0, 1, 0)

	err := GetDB(nil).Raw(`
SELECT count(1) FROM(
	SELECT block_id AS block,txhash AS hash,sender_id,recipient_id,type,created_at,amount,false AS isutxo,sender_balance,recipient_balance FROM "1_history"
		WHERE ecosystem = ? AND (sender_id = ? or recipient_id = ?) and created_at >= ? and created_at < ? AND type <> 24
				union all
	SELECT block,hash,sender_id,recipient_id,type,created_at,amount,true AS isutxo,sender_balance,recipient_balance FROM utxo_history 
	WHERE type <> 1 AND ecosystem = ? AND (sender_id = ? or recipient_id = ?) and created_at >= ? and created_at < ?
)AS v1
`, req.Ecosystem, kid, kid, stTime.UnixMilli(), edTime.UnixMilli(),
		req.Ecosystem, kid, kid, stTime.UnixMilli(), edTime.UnixMilli()).Take(&rets.Total).Error
	if err != nil {
		return nil, err
	}

	var list []historyMonthRet
	err = GetDB(nil).Raw(`
SELECT v1.block,v1.hash,v1.sender_id,v1.recipient_id,v1.type,v1.created_at,v1.amount,v1.isutxo,
	CASE WHEN v1.isutxo = FALSE THEN
		v1.sender_balance+COALESCE((
			SELECT CASE WHEN sender_id = v1.sender_id THEN
				sender_balance
			ELSE
				recipient_balance
			END AS balance
			FROM utxo_history 
			WHERE(recipient_id = v1.sender_id OR sender_id = v1.sender_id) AND ecosystem = ? AND block <= v1.block ORDER BY id DESC LIMIT 1
		),0)
	ELSE
		v1.sender_balance+COALESCE((
			SELECT CASE WHEN sender_id = v1.sender_id THEN
				sender_balance
			ELSE
				recipient_balance
			END AS balance
			FROM "1_history" 
			WHERE(recipient_id = v1.sender_id OR sender_id = v1.sender_id) AND ecosystem = ? AND block_id <= v1.block ORDER BY id DESC LIMIT 1
		),0)
	END AS sender_balance,

	CASE WHEN v1.isutxo = FALSE THEN
		v1.recipient_balance+COALESCE((
			SELECT CASE WHEN sender_id = v1.recipient_id THEN
				sender_balance
			ELSE
				recipient_balance
			END AS balance
			FROM utxo_history 
			WHERE(recipient_id = v1.recipient_id OR sender_id = v1.recipient_id) AND ecosystem = ? AND block <= v1.block ORDER BY id DESC LIMIT 1
		),0)
	ELSE
		v1.recipient_balance+COALESCE((
			SELECT CASE WHEN sender_id = v1.recipient_id THEN
				sender_balance
			ELSE
				recipient_balance
			END AS balance
			FROM "1_history" 
			WHERE(recipient_id = v1.recipient_id OR sender_id = v1.recipient_id) AND ecosystem = ? AND block_id <= v1.block ORDER BY id DESC LIMIT 1
		),0)
	END AS recipient_balance 
FROM(
	SELECT block_id AS block,id,txhash AS hash,sender_id,recipient_id,type,created_at,amount,false AS isutxo,sender_balance,recipient_balance FROM "1_history"
	WHERE ecosystem = ? AND (sender_id = ? or recipient_id = ?) and created_at >= ? and created_at < ? AND type <> 24
			union all
	SELECT block,id,hash,sender_id,recipient_id,type,created_at,amount,true AS isutxo,sender_balance,recipient_balance FROM utxo_history 
	WHERE type <> 1 AND ecosystem = ? AND (sender_id = ? or recipient_id = ?) and created_at >= ? and created_at < ?
	ORDER BY block DESC,id DESC

	OFFSET ? LIMIT ?
)AS v1
`, req.Ecosystem, req.Ecosystem, req.Ecosystem, req.Ecosystem,
		req.Ecosystem, kid, kid, stTime.UnixMilli(), edTime.UnixMilli(),
		req.Ecosystem, kid, kid, stTime.UnixMilli(), edTime.UnixMilli(),
		(req.Page-1)*req.Limit, req.Limit).Find(&list).Error
	if err != nil {
		return nil, err
	}

	rets.List = *th.ChangeMonthResults(&list, kid)
	rets.TokenSymbol = Tokens.Get(req.Ecosystem)

	return &rets, nil
}

func (th *History) ChangeMonthResults(vers *[]historyMonthRet, kid int64) *[]MonthHistoryResponse {
	var dats []MonthHistoryResponse
	for k, t := range *vers {
		s := t.ChangeMonthResult(kid)
		s.ID = int64(k) + 1
		dats = append(dats, *s)
	}
	return &dats
}

func (th *historyMonthRet) ChangeMonthResult(kid int64) *MonthHistoryResponse {
	var balance string
	if kid == th.SenderId {
		balance = th.SenderBalance
	} else if kid == th.RecipientId {
		balance = th.RecipientBalance
	}
	var txTime int64
	txTime = MsToSeconds(th.CreatedAt)
	s := MonthHistoryResponse{
		Balance: balance,
		Amount:  th.Amount,
		Time:    txTime,
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

func (th *History) GetDBDayNftInComeinfo(kid, t1, t2 int64) (decimal.Decimal, error) {
	type allAmount struct {
		Amounts decimal.Decimal `gorm:"column:amounts"`
	}
	var at allAmount
	err := GetDB(nil).Table(th.TableName()).Select("sum(amount) amounts").Where(`type = ? and recipient_id = ? and created_at >= ? and created_at < ?`, 12, kid, t1, t2).Take(&at).Error
	//.Row().Scan(&allAmount)

	if err != nil {
		return decimal.Zero, err
	}
	return at.Amounts, err
}

func (th *History) GetList(c *params.MineHistoryRequest) (*GeneralResponse, error) {
	var (
		rets   GeneralResponse
		txList []AccountTxHistory
	)
	type accountHistory struct {
		Block        int64
		Hash         []byte
		Address      int64
		SenderId     int64
		RecipientId  int64
		Type         int
		CreatedAt    int64
		Amount       string
		Isutxo       bool
		ContractName string
	}
	var list []accountHistory

	kid := converter.StringToAddress(c.Wallet)
	if kid == 0 && c.Wallet != "0000-0000-0000-0000-0000" {
		return nil, errors.New("account invalid")
	}
	var whereSql *gorm.DB
	switch c.Opt {
	case "send":
		whereSql = GetDB(nil).Where("ecosystem = ? AND sender_id = ?", c.Ecosystem, kid)
	case "recipient":
		whereSql = GetDB(nil).Where("ecosystem = ? AND recipient_id = ?", c.Ecosystem, kid)
	case "all":
		whereSql = GetDB(nil).Where("ecosystem = ?", c.Ecosystem).Where(GetDB(nil).Where("recipient_id = ?", kid).Or("sender_id = ?", kid))
	}

	err := GetDB(nil).Raw("SELECT count(1) FROM(? UNION ALL ?)AS v1",
		GetDB(nil).Select("FALSE AS isutxo").Where(whereSql).Where("type <> 24").Table("1_history"),
		GetDB(nil).Select("TRUE AS isutxo").Where(whereSql).Where("type <> 1").Table("utxo_history"),
	).Take(&rets.Total).Error
	if err != nil {
		return nil, err
	}

	err = GetDB(nil).Raw(
		`SELECT v1.*,v2.contract_name,v2.address FROM(
				SELECT * FROM(? UNION ALL ?) as v1 ORDER BY block DESC,created_at DESC OFFSET ? LIMIT ?
			)AS v1
			LEFT JOIN (SELECT contract_name,hash,address FROM log_transactions)AS v2 ON(v2.hash = v1.hash)
	`,
		GetDB(nil).Select("block_id AS block,txhash AS hash,sender_id,recipient_id,type,created_at,amount,false AS isutxo").
			Where(whereSql).Where("type <> 24").Table("1_history"),

		GetDB(nil).Select("block,hash,sender_id,recipient_id,type,created_at,amount,true AS isutxo").Where(whereSql).
			Where("type <> 1").Table("utxo_history"),
		(c.Page-1)*c.Limit,
		c.Limit,
	).Find(&list).Error
	if err != nil {
		return nil, err
	}

	tokenSymbol := Tokens.Get(c.Ecosystem)

	for k, val := range list {
		var rlt AccountTxHistory
		rlt.Address = converter.AddressToString(val.Address)
		rlt.Sender = converter.AddressToString(val.SenderId)
		rlt.Recipient = converter.AddressToString(val.RecipientId)
		rlt.BlockId = val.Block
		rlt.Hash = hex.EncodeToString(val.Hash)
		rlt.TokenSymbol = tokenSymbol
		rlt.Amount = val.Amount
		if val.Isutxo {
			rlt.Type = compatibleContractAccountType(val.Type)
			rlt.Contract = parseSpentInfoHistoryType(val.Type)
		} else {
			rlt.Type = val.Type
			rlt.Contract = val.ContractName
		}
		rlt.CreatedAt = MsToSeconds(val.CreatedAt)
		rlt.Id = k + 1

		txList = append(txList, rlt)
	}
	rets.List = txList
	rets.Page = c.Page
	rets.Limit = c.Limit
	return &rets, nil
}

func (th *History) GetWalletMonthHistory(eid, keyId, t1, t2 int64) (*WalletMonthHistory, error) {
	var (
		ret    WalletMonthHistory
		sCount int64
		rCount int64
		in     string
		out    string
		err    error
	)

	rCount, sCount, err = getAccountTxCount(eid, keyId, t1, t2)
	if err != nil {
		return nil, err
	}

	if rCount > 0 {
		err = GetDB(nil).Raw(`
SELECT COALESCE(sum(amount),0)+
	(SELECT COALESCE(sum(amount),0) FROM utxo_history WHERE 
	recipient_id = ? AND ecosystem = ? AND type <> 1 AND created_at >= ? AND created_at < ?)AS in_amount 
FROM "1_history" WHERE recipient_id = ? AND ecosystem = ? AND created_at >= ? AND created_at < ? AND type <> 24
`, keyId, eid, t1, t2, keyId, eid, t1, t2).Row().Scan(&in)
		if err != nil {
			return nil, err
		}

	} else {
		in = "0"
	}

	if sCount > 0 {

		err = GetDB(nil).Raw(`
SELECT COALESCE(sum(amount),0)+
	(SELECT COALESCE(sum(amount),0) FROM utxo_history 
	WHERE sender_id = ? AND ecosystem = ? AND type <> 1 AND created_at >= ? AND created_at < ?)AS out_amount 
FROM "1_history" WHERE sender_id = ? AND ecosystem = ? AND created_at >= ? AND created_at < ? AND type <> 24
`, keyId, eid, t1, t2, keyId, eid, t1, t2).Row().Scan(&out)
		if err != nil {
			return nil, err
		}
	} else {
		out = "0"
	}

	dIn, err := decimal.NewFromString(in)
	if err != nil {
		return &ret, err
	}
	dOut, err := decimal.NewFromString(out)
	if err != nil {
		return &ret, err
	}

	ret.OutCount = sCount
	ret.OutAmount = dOut
	ret.InCount = rCount
	ret.InAmount = dIn

	return &ret, err
}

func (th *History) GetAccountHistoryTotals(id int64, keyId int64) (*WalletHistoryHex, error) {
	var (
		ret    WalletHistoryHex
		sCount int64
		rCount int64
		in     = "0"
		out    = "0"
		err    error
	)

	rCount, sCount, err = getAccountTxCount(id, keyId, 0, 0)
	if err != nil {
		return nil, err
	}

	//in amount
	if rCount > 0 {
		err = GetDB(nil).Raw(`
SELECT COALESCE(sum(amount),0)+
	(SELECT COALESCE(sum(amount),0) FROM utxo_history WHERE recipient_id = ? AND ecosystem = ? AND type <> 1)AS in_amount 
FROM "1_history" WHERE recipient_id = ? AND ecosystem = ? AND type <> 24
`, keyId, id, keyId, id).Row().Scan(&in)
		if err != nil {
			return &ret, err
		}
	}

	//out amount
	if sCount > 0 {
		err = GetDB(nil).Raw(`
SELECT COALESCE(sum(amount),0)+
	(SELECT COALESCE(sum(amount),0) FROM utxo_history WHERE sender_id = ? AND ecosystem = ? AND type <> 1)AS out_amount 
FROM "1_history" WHERE sender_id = ? AND ecosystem = ? AND type <> 24
`, keyId, id, keyId, id).Row().Scan(&out)
		if err != nil {
			return &ret, err
		}
	}

	inAmount, _ := decimal.NewFromString(in)
	outAmount, _ := decimal.NewFromString(out)

	ret.InTx = rCount
	ret.OutTx = sCount

	ret.Transaction = ret.InTx + ret.OutTx
	ret.InAmount = inAmount
	ret.OutAmount = outAmount

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
	rets.InAmount = dh.InAmount.String()
	rets.OutAmount = dh.OutAmount.String()
	rets.AllAmount = dh.InAmount.Add(dh.OutAmount).String()
	rets.InTx = dh.InTx
	rets.OutTx = dh.OutTx
	rets.AllTx = dh.Transaction
	rets.TokenSymbol = Tokens.Get(ecosystem)
	rets.Ecosystem = ecosystem

	return &rets, nil
}

func getAccountTxCount(ecosystem int64, keyId int64, st, ed int64) (inTx, outTx int64, err error) {
	var (
		inSqlQuery  string
		outSqlQuery string
		q1          *gorm.DB
		q2          *gorm.DB
	)

	if ecosystem > 0 {
		//in out tx
		inSqlQuery = `
SELECT count(1) FROM(
	SELECT block_id AS block,txhash AS hash,sender_id,recipient_id,type,created_at,amount,false AS isutxo FROM "1_history" WHERE ecosystem = ? AND type <> 24
		union all
	SELECT block,hash,sender_id,recipient_id,type,created_at,amount,true AS isutxo FROM utxo_history WHERE type <> 1 AND ecosystem = ?
)AS v1
WHERE recipient_id = ?
`
		if st > 0 {
			inSqlQuery = `
SELECT count(1) FROM(
	SELECT block_id AS block,txhash AS hash,sender_id,recipient_id,type,created_at,amount,false AS isutxo FROM "1_history"
	WHERE ecosystem = ? AND created_at >= ? AND created_at < ? AND type <> 24
		union all
	SELECT block,hash,sender_id,recipient_id,type,created_at,amount,true AS isutxo FROM utxo_history 
	WHERE type <> 1 AND ecosystem = ? AND created_at >= ? AND created_at < ?
)AS v1
WHERE recipient_id = ?
`
		}

		outSqlQuery = `
SELECT count(1) FROM(
	SELECT block_id AS block,txhash AS hash,sender_id,recipient_id,type,created_at,amount,false AS isutxo FROM "1_history" 
	WHERE ecosystem = ? AND type <> 24
		union all
	SELECT block,hash,sender_id,recipient_id,type,created_at,amount,true AS isutxo FROM utxo_history WHERE type <> 1 AND ecosystem = ?
)AS v1
WHERE sender_id = ?
`
		if st > 0 {
			outSqlQuery = `
SELECT count(1) FROM(
	SELECT block_id AS block,txhash AS hash,sender_id,recipient_id,type,created_at,amount,false AS isutxo FROM "1_history" 
	WHERE ecosystem = ? AND created_at >= ? AND created_at < ? AND type <> 24
		union all
	SELECT block,hash,sender_id,recipient_id,type,created_at,amount,true AS isutxo FROM utxo_history 
	WHERE type <> 1 AND ecosystem = ? AND created_at >= ? AND created_at < ?
)AS v1
WHERE sender_id = ?
`
		}
		if st > 0 {
			q1 = GetDB(nil).Raw(inSqlQuery, ecosystem, st, ed, ecosystem, st, ed, keyId)
			q2 = GetDB(nil).Raw(outSqlQuery, ecosystem, st, ed, ecosystem, st, ed, keyId)
		} else {
			q1 = GetDB(nil).Raw(inSqlQuery, ecosystem, ecosystem, keyId)
			q2 = GetDB(nil).Raw(outSqlQuery, ecosystem, ecosystem, keyId)
		}
	} else {
		inSqlQuery = `
SELECT count(1) FROM(
	SELECT block_id AS block,txhash AS hash,sender_id,recipient_id,type,created_at,amount,false AS isutxo FROM "1_history" WHERE type <> 24
		union all
	SELECT block,hash,sender_id,recipient_id,type,created_at,amount,true AS isutxo FROM utxo_history WHERE type <> 1
)AS v1
WHERE recipient_id = ?
`
		outSqlQuery = `
SELECT count(1) FROM(
	SELECT block_id AS block,txhash AS hash,sender_id,recipient_id,type,created_at,amount,false AS isutxo FROM "1_history" WHERE type <> 24
		union all
	SELECT block,hash,sender_id,recipient_id,type,created_at,amount,true AS isutxo FROM utxo_history WHERE type <> 1
)AS v1
WHERE sender_id = ?
`
		q1 = GetDB(nil).Raw(inSqlQuery, keyId)
		q2 = GetDB(nil).Raw(outSqlQuery, keyId)
	}
	err = q1.Row().Scan(&inTx)
	if err != nil {
		return
	}

	err = q2.Row().Scan(&outTx)
	if err != nil {
		return
	}
	return
}

func (p *History) GetKeyBalance(keyId int64, ecosystem int64) (balance decimal.Decimal, err error) {
	err = GetDB(nil).Raw(`
SELECT CASE WHEN v2.sender_id = v1.key_id THEN
	COALESCE(v2.sender_balance,0)
ELSE
	COALESCE(v2.recipient_balance,0)
END AS balance
FROM(
	SELECT ? AS key_id
)AS v1 LEFT JOIN "1_history" AS v2 
ON((v2.recipient_id = v1.key_id OR v2.sender_id = v1.key_id) AND v2.ecosystem = ?)ORDER BY id DESC LIMIT 1`, keyId, ecosystem).Take(&balance).Error
	return
}

func getNftMinerBurstCount(recipientId int64, nftMinerId int64) int64 {
	nftId := strconv.FormatInt(nftMinerId, 10)
	var total int64
	var p History
	err := GetDB(nil).Table(p.TableName()).
		Where(fmt.Sprintf("type = 12 AND comment = 'NFT Miner #%s' AND recipient_id = ?", nftId), recipientId).Count(&total).Error
	if err != nil {
		log.WithFields(log.Fields{"error": err, "recipientId": recipientId, "nftMinerId": nftMinerId}).Error("get nft miner burst count failed")
		return 0
	}
	return total
}
