package sql

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/IBAX-io/go-ibax/packages/converter"
	"github.com/IBAX-io/go-ibax/packages/smart"
	"github.com/IBAX-io/go-ibax/packages/storage/sqldb"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"reflect"
	"strconv"
	"time"
)

var NftMinerReady bool

const SvgTimeFormat = "15:04:05 02-01-2006 (UTC)"

type NftMinerItems struct {
	ID          int64  `gorm:"primary_key;not null"`         //NFT ID
	EnergyPoint int    `gorm:"column:energy_point;not null"` //power
	Owner       string `gorm:"column:owner;not null"`        //owner account
	Creator     string `gorm:"column:creator;not null"`      //create account
	MergeCount  int64  `gorm:"column:merge_count;not null"`  //merage count
	MergeStatus int64  `gorm:"column:merge_status;not null"` //1:un merge(valid) 0:merge(invalid)
	Attributes  string `gorm:"column:attributes;not null"`   //SVG Params
	TempId      int64  `gorm:"column:temp_id;not null"`
	DateCreated int64  `gorm:"column:date_created;not null"` //create time
	Coordinates string `gorm:"column:coordinates;type:jsonb"`
	TokenHash   []byte `gorm:"column:token_hash"`
	TxHash      []byte `gorm:"column:tx_hash"`
}

type SvgParams struct {
	Owner       string `json:"owner"`
	Point       string `json:"point"`
	Number      int64  `json:"number"`
	Location    string `json:"location"`
	DateCreated int64  `json:"date_created"` //milliseconds
}

type StrReplaceStruct struct {
	CapitalLetter    int `json:"capital_letter"`
	LowercaseLetters int `json:"lowercase_letters"`
	Number           int `json:"number"`
	OtherString      int `json:"other_string"`
}

func (p *NftMinerItems) TableName() string {
	return "1_nft_miner_items"
}

func (p *NftMinerItems) GetById(id int64) (bool, error) {
	return isFound(GetDB(nil).Where("id = ? ", id).First(p))
}

func (p *NftMinerItems) GetByTokenHash(tokenHash string) (bool, error) {
	hash, _ := hex.DecodeString(tokenHash)
	return isFound(GetDB(nil).Where("token_hash = ? ", hash).First(p))
}

func (p *NftMinerItems) GetUserNftMinerSummary(keyid string) (NftMinerSummaryResponse, error) {
	var ret NftMinerSummaryResponse
	var total int64
	if !HasTableOrView(p.TableName()) {
		return ret, nil
	}
	if err := GetDB(nil).Table(p.TableName()).Where("owner = ? AND merge_status = 1", keyid).Count(&total).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return ret, err
		}

	}

	var sk NftMinerStaking
	var list []NftMinerStaking
	if err := GetDB(nil).Table(sk.TableName()).Select("stake_amount,energy_power,start_dated,end_dated").Where("staker = ? AND staking_status = 1", keyid).Find(&list).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return ret, err
		}
	}

	var ins SumAmount
	var his History
	kid := converter.StringToAddress(keyid)
	_, err := isFound(GetDB(nil).Table(his.TableName()).Select("sum(amount)").Where("recipient_id = ? AND type = 12", kid).Take(&ins))
	if err != nil {
		return ret, err
	}

	stakingAmount := decimal.Zero
	nowTime := time.Now().Unix()
	for _, v := range list {
		if !(v.StartDated <= nowTime && v.EndDated >= nowTime) {
			v.EnergyPower = 0
		}
		ret.EnergyPower += v.EnergyPower
		staking, _ := decimal.NewFromString(v.StakeAmount)
		stakingAmount = stakingAmount.Add(staking)
	}

	ret.NftMinerCount = total
	ret.NftMinerIns = ins.Sum.String()
	ret.StakeAmount = stakingAmount.String()

	return ret, nil
}

func (p *NftMinerItems) GetUserNftFifteenDayOverview(day int, wallet string) (*[]NftMinerOverviewResponse, error) {
	var ret []NftMinerOverviewResponse
	var his History
	var bk Block
	kid := converter.StringToAddress(wallet)
	ts, err := bk.GetSystemTime()
	if err != nil {
		return &ret, err
	}
	tz := GetZoneTimes()
	t := tz.AddDate(0, 0, -day+1)
	if ts > t.Unix() {
		d := time.Unix(ts, 0)
		t = d.In(time.UTC)
	}
	for i := 0; i < day; i++ {
		te := t
		te = te.AddDate(0, 0, 1)

		if amount, err := his.GetDBDayNftInComeinfo(kid, t.UnixMilli(), te.UnixMilli()); err == nil {
			var dm NftMinerOverviewResponse
			dm.Amount = amount.String()
			dm.Time = t.Unix()
			ret = append(ret, dm)
		} else {
			return &ret, err
		}

		if te.After(tz) {
			break
		}
		t = t.AddDate(0, 0, 1)
	}

	return &ret, nil
}

func (p *NftMinerItems) GetNftMinerKeyInfo(account string) (*CommonResult, error) {
	kid := converter.StringToAddress(account)
	if kid == 0 {
		return nil, fmt.Errorf("account invalid:%s", account)
	}

	var list []nftMinerInfo

	q := GetDB(nil).Raw(`
SELECT v1.id,encode(v1.token_hash,'hex')token_hash,v1.energy_point,COALESCE(v2.stake_amount,'0') stake_amount,
	COALESCE(v2.energy_power,0)energy_power,(SELECT COALESCE(count(1),0)burst
FROM "1_history" WHERE type = 12 AND comment = 'NFT Miner #'||v1.id AND recipient_id = ?),v2.start_dated,v2.end_dated FROM(
	SELECT id,token_hash,owner,energy_point FROM "1_nft_miner_items" WHERE owner = ? AND merge_status = 1
)AS v1
LEFT JOIN(
	SELECT stake_amount,energy_power,token_id,start_dated,end_dated FROM "1_nft_miner_staking" WHERE staker = ? AND staking_status = 1
)AS v2 ON (v2.token_id = v1.id)
`, kid, account, account)

	var res CommonResult

	if err := GetDB(nil).Table(p.TableName()).Where("owner = ?", account).Count(&res.Total).Error; err != nil {
		return nil, err
	}
	f, err := isFound(GetDB(nil).Table(p.TableName()).Where("creator = ?", account).Take(p))
	if err != nil {
		return nil, err
	}
	if f {
		res.IsCreate = true
	}

	err = q.Find(&list).Error
	if err != nil {
		return nil, err
	}

	nowTime := time.Now().Unix()
	for k, v := range list {
		if !(v.StartDated <= nowTime && v.EndDated >= nowTime) {
			v.EnergyPower = 0
		}
		list[k] = v
	}

	res.Rets = list
	return &res, nil
}

func (p *NftMinerItems) GetNftMinerRewardHistory(search any, page, limit int) (*GeneralResponse, error) {
	var account string
	switch reflect.TypeOf(search).String() {
	case "string":
		account = search.(string)
	default:
		log.WithFields(log.Fields{"search type": reflect.TypeOf(search).String()}).Info("get Nft Miner Reward History Failed")
		return nil, errors.New("request params invalid")
	}
	if account == "" {
		return nil, fmt.Errorf("account can not be empty")
	}

	kid := converter.StringToAddress(account)
	if kid == 0 {
		return nil, fmt.Errorf("account invalid:%s", account)
	}
	q := GetDB(nil).Raw(`
SELECT v2.token_hash,v1.amount,v1.created_at FROM (
	SELECT CAST(substr(comment,12,length(comment)-length('NFT Miner #')) AS numeric) nft_id,amount,created_at/1000 as created_at FROM "1_history" 
	WHERE type = 12 AND recipient_id = ? ORDER BY id DESC OFFSET ? LIMIT ?
)AS v1
LEFT JOIN(
	SELECT encode(token_hash,'hex')token_hash,id FROM "1_nft_miner_items"
)AS v2 ON(v2.id = v1.nft_id)
ORDER BY created_at DESC
`, kid, (page-1)*limit, limit)

	rets := &GeneralResponse{}
	var his History

	err := GetDB(nil).Table(his.TableName()).Where("type = 12 AND recipient_id = ?", kid).Count(&rets.Total).Error
	if err != nil {
		return nil, err
	}

	rows, err := q.Rows()
	if err != nil {
		return nil, err
	}
	list, err := sqldb.GetNodeResult(rows)
	if err != nil {
		return nil, err
	}

	rets.List = list
	rets.Page = page
	rets.Limit = limit
	return rets, nil
}

func NftMinerTableIsExist() bool {
	var p NftMinerItems
	if !HasTableOrView(p.TableName()) {
		return false
	}
	return true
}

func (p *NftMinerItems) GetNftMinerDetailBySearch(search any, wallet string) (*NftMinerInfoResponse, error) {
	var (
		rets NftMinerInfoResponse
		f    bool
		err  error
	)
	switch reflect.TypeOf(search).String() {
	case "string":
		minerHash := search.(string)
		if minerHash == "" {
			return nil, errors.New("request params invalid")
		}
		f, err = p.GetByTokenHash(minerHash)
	case "json.Number":
		minerId, err := search.(json.Number).Int64()
		if err != nil {
			return nil, err
		}
		if minerId <= 0 {
			return nil, errors.New("request params invalid")
		}
		f, err = p.GetById(minerId)
	default:
		log.WithFields(log.Fields{"search type": reflect.TypeOf(search).String()}).Info("Get Nft Miner Detail By Search Failed")
		return nil, errors.New("request params invalid")
	}
	if err != nil {
		return nil, err
	}
	if !f {
		return nil, errors.New("NFT doesn't not exist")
	}
	keyId := converter.StringToAddress(wallet)
	if keyId == 0 {
		return nil, errors.New("request params invalid")
	}

	rets.ID = p.ID
	rets.Hash = hex.EncodeToString(p.TokenHash)
	rets.EnergyPoint = p.EnergyPoint
	rets.DateCreated = p.DateCreated
	rets.Owner = p.Owner

	var stak NftMinerStaking
	err = GetDB(nil).Table(stak.TableName()).Where("token_id = ? and staker = ?", p.ID, wallet).Count(&rets.StakeCount).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Info("get nft info stakeCount err:", err.Error(), " nftId:", p.ID)
		}
		return nil, err
	}

	nowTime := time.Now().Unix()
	f, err = isFound(GetDB(nil).Where("token_id = ? AND staker = ?", p.ID, wallet).Last(&stak))
	if err != nil {
		log.Info("get nft info stakeAmount err:", err.Error(), " nftId:", p.ID)
		return nil, err
	}
	if f {
		rets.StakeStatus = stak.StakingStatus
		if stak.StakingStatus == 1 {
			rets.StakeAmount = stak.StakeAmount
			//rets.Cycle = int64(time.Unix(stak.EndDated, 0).Sub(time.Unix(stak.StartDated, 0)).Hours() / 24)
			if stak.StartDated <= nowTime && stak.EndDated >= nowTime {
				rets.EnergyPower = stak.EnergyPower
			} else {
				rets.StakeStatus = 3
			}
		}

	}

	var reward SumAmount
	var his History
	//kid := converter.StringToAddress(p.Owner)
	q := GetDB(nil).Table(his.TableName()).Where("type = 12 AND comment = ? AND recipient_id = ?", fmt.Sprintf("NFT Miner #%d", p.ID), keyId)
	err = q.Select("sum(amount)").Take(&reward).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Info("get nft miner reward info err:", err.Error(), " nftId:", p.ID)
			return nil, err
		}
	}
	err = q.Count(&rets.RewardCount).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Info("get nft miner info reward Count err:", err.Error(), " nftId:", p.ID)
			return nil, err
		}
	}
	rets.Ins = reward.Sum.String()

	return &rets, nil
}

func (p *NftMinerItems) GetNftMinerTxInfo(search any, page, limit int, order, wallet string) (*GeneralResponse, error) {
	var (
		rets  []NftMinerTxInfoResponse
		total int64
		ret   GeneralResponse
		f     bool
		err   error
	)
	keyId := converter.StringToAddress(wallet)
	if keyId == 0 {
		return nil, errors.New("request params wallet invalid")
	}
	if order == "" {
		order = "id desc"
	}
	switch reflect.TypeOf(search).String() {
	case "string":
		f, err = p.GetByTokenHash(search.(string))
	case "json.Number":
		tokenId, err := search.(json.Number).Int64()
		if err != nil {
			return nil, err
		}
		f, err = p.GetById(tokenId)
	default:
		log.WithFields(log.Fields{"search type": reflect.TypeOf(search).String()}).Warn("Get Nft Miner Tx Info Search Failed")
		return nil, errors.New("request params invalid")
	}

	if err != nil {
		log.Info("get nft miner txInfo err:", err.Error(), " nftId:", p.ID)
		return nil, err
	}
	if !f {
		return nil, errors.New("NFT Miner Doesn't Not Exist")
	}
	var his []History
	err = GetDB(nil).Table("1_history").Where("type = 12 AND comment = ? and recipient_id = ?", fmt.Sprintf("NFT Miner #%d", p.ID), keyId).Count(&total).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Info("get nft tx info total err:", err.Error(), " nftId:", p.ID)
			return nil, err
		}
	}

	err = GetDB(nil).Select("id,created_at,amount").
		Where("type = 12 AND comment = ? AND recipient_id = ?", fmt.Sprintf("NFT Miner #%d", p.ID), keyId).
		Limit(limit).Offset((page - 1) * limit).Order(order).Find(&his).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Info("get nft miner tx info nftIns err:", err.Error(), " nft miner id:", p.ID)
			return nil, err
		}
	}
	rets = make([]NftMinerTxInfoResponse, len(his))
	for i := 0; i < len(his); i++ {
		rets[i].ID = his[i].ID
		rets[i].NftMinerId = p.ID
		rets[i].Time = MsToSeconds(his[i].CreatedAt)
		rets[i].Ins = his[i].Amount.String()
	}
	ret.Page = page
	ret.Limit = limit
	ret.Total = total
	ret.List = rets

	return &ret, nil
}

func (p *NftMinerItems) ParseSvgParams() (string, error) {
	var (
		ret SvgParams
		app AppParam
	)
	err := json.Unmarshal([]byte(p.Attributes), &ret)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Parse Svg Params json unmarshal failed")
		return "", err
	}
	f, err := app.GetById(nil, p.TempId)
	if err != nil {
		log.WithFields(log.Fields{"error": err, "tempId": p.TempId}).Error("Parse Svg Params get app id failed")
		return "", err
	}
	if !f {
		return "", nil
	}
	star := "★"
	if p.EnergyPoint <= 20 {
	} else if p.EnergyPoint <= 40 {
		star = "★★"
	} else if p.EnergyPoint <= 60 {
		star = "★★★"
	} else if p.EnergyPoint <= 80 {
		star = "★★★★"
	} else {
		star = "★★★★★"
	}
	return fmt.Sprintf(app.Value, ret.Point, "#"+strconv.FormatInt(ret.Number, 10), formatLocation(ret.Location), smart.Date(SvgTimeFormat, MsToSeconds(ret.DateCreated)), ret.Owner, star), nil
}

//formatLocation example:NorthAmerica result:North America
func formatLocation(location string) string {
	strs, indexList := StrReplaceAllString(location)
	if strs.CapitalLetter >= 2 {
		for i := 2; i <= strs.CapitalLetter; i++ {
			location = location[:indexList[i-1].CapitalLetter] + " " + location[indexList[i-1].CapitalLetter:]
		}
	}

	return location
}

func StrReplaceAllString(s2 string) (strReplace StrReplaceStruct, indexList []StrReplaceStruct) {
	indexList = make([]StrReplaceStruct, len(s2))
	for i := 0; i < len(s2); i++ {
		switch {
		case 64 < s2[i] && s2[i] < 91:
			strReplace.CapitalLetter += 1
			indexList[strReplace.CapitalLetter-1].CapitalLetter = i
		case 96 < s2[i] && s2[i] < 123:
			strReplace.LowercaseLetters += 1
			indexList[strReplace.LowercaseLetters-1].LowercaseLetters = i
		case 47 < s2[i] && s2[i] < 58:
			strReplace.Number += 1
			indexList[strReplace.Number-1].Number = i
		default:
			strReplace.OtherString += 1
			indexList[strReplace.OtherString-1].OtherString = i
		}
	}
	return strReplace, indexList
}
