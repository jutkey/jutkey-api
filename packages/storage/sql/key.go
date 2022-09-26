package sql

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/IBAX-io/go-ibax/packages/converter"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"reflect"
	"strconv"
	"strings"
)

type Key struct {
	Ecosystem int64
	ID        int64           `gorm:"primary_key;not null"`
	PublicKey []byte          `gorm:"column:pub;not null"`
	Amount    decimal.Decimal `gorm:"not null"`
	Maxpay    decimal.Decimal `gorm:"not null"`
	Deposit   decimal.Decimal `gorm:"not null"`
	Multi     int64           `gorm:"not null"`
	Deleted   int64           `gorm:"not null"`
	Blocked   int64           `gorm:"not null"`
	AccountID string          `gorm:"column:account;not null"`
	Lock      string          `gorm:"column:lock;type:jsonb"`
}

func (p Key) TableName() string {
	return `1_keys`
}

func (p *Key) GetEcosystemsKeysCount(ecoId int64) (int64, error) {
	var cnt int64
	if err := GetDB(nil).Table("1_keys").Where("ecosystem = ?", ecoId).Count(&cnt).Error; err != nil {
		return 0, err
	}
	return cnt, nil
}

func (p *Key) GetEcosystemKeys(ecoId, keyId int64) (bool, error) {
	return isFound(GetDB(nil).Where("id = ? and ecosystem = ?", keyId, ecoId).First(p))
}

func (key *Key) GetEcosystemsKeyAmount(keyId int64, page, limit int, order string, search any) (*EcosystemKeyTotalResult, error) {
	var (
		list  []keyEcosystem
		rss   []EcosystemKeyTotalRet
		where string
	)
	rets := new(EcosystemKeyTotalResult)
	rets.Page = page
	rets.Limit = limit
	if len(order) == 0 {
		order = "ecosystem asc,k1.id asc"
	} else {
		order = "ecosystem asc," + order
	}
	if page == 0 {
		page = 1
	}
	if search != nil {
		switch reflect.TypeOf(search).String() {
		case "string":
			str := search.(string)
			if len(str) == 0 {
				return rets, errors.New("request params invalid")
			}
			ks := strings.Split(str, " ")
			if len(ks) != 3 {
				return rets, fmt.Errorf("Error in query condition: %s. ", str)
			}
			if (strings.Contains(ks[1], ">") || strings.Contains(ks[1], "=") || strings.Contains(ks[1], "<")) && CheckSql(str) {
				where = str
			}
		default:
			log.WithFields(log.Fields{"search type": reflect.TypeOf(search).String()}).Warn("Get Node Detail Failed")
			return rets, errors.New("request params invalid")
		}
	}
	if len(where) > 0 {
		err := GetDB(nil).Raw(fmt.Sprintf(`
	SELECT count(*) FROM "1_keys" as k1 LEFT JOIN "1_ecosystems" AS e1 ON(e1.id = k1.ecosystem AND k1.id = ?) WHERE token_symbol <> '' AND %s
`, where), keyId).Count(&rets.Total).Error
		if err != nil {
			return rets, err
		}
	} else {
		err := GetDB(nil).Raw(`
	SELECT count(*) FROM "1_keys" as k1 LEFT JOIN "1_ecosystems" AS e1 ON(e1.id = k1.ecosystem AND k1.id = ?) WHERE token_symbol <> ''
`, keyId).Count(&rets.Total).Error
		if err != nil {
			return rets, err
		}
	}

	if len(where) > 0 {
		err := GetDB(nil).Raw(fmt.Sprintf(`
	SELECT * FROM "1_keys" as k1 LEFT JOIN "1_ecosystems" AS e1 ON(e1.id = k1.ecosystem AND k1.id = ?) WHERE token_symbol <> '' AND %s
ORDER BY %s OFFSET ? LIMIT ?
`, where, order), keyId, (page-1)*limit, limit).Find(&list).Error
		if err != nil {
			return rets, err
		}
	} else {
		err := GetDB(nil).Raw(fmt.Sprintf(`
	SELECT * FROM "1_keys" as k1 LEFT JOIN "1_ecosystems" AS e1 ON(e1.id = k1.ecosystem AND k1.id = ?) WHERE token_symbol <> ''
ORDER BY %s OFFSET ? LIMIT ?
`, order), keyId, (page-1)*limit, limit).Find(&list).Error
		if err != nil {
			return rets, err
		}
	}
	for _, val := range list {
		rlt, err := val.ChangeResults(keyId)
		if err != nil {
			return rets, err
		}
		rss = append(rss, *rlt)
	}
	rets.Rets = rss

	return rets, nil
}

func (m *keyEcosystem) ChangeResults(keyId int64) (*EcosystemKeyTotalRet, error) {

	escape := func(value any) string {
		return strings.Replace(fmt.Sprint(value), `'`, `''`, -1)
	}

	var spent SpentInfo
	utxoAmount, err := spent.GetAmountByKeyId(keyId, m.Key.Ecosystem)
	if err != nil {
		return nil, err
	}
	s := EcosystemKeyTotalRet{
		ID:            m.Key.Ecosystem,
		AccountAmount: m.Amount.String(), //converter.AddressToString(m.Devid),
		Account:       m.AccountID,
	}
	s.UtxoAmount = utxoAmount.String()
	s.Amount = utxoAmount.Add(m.Amount).String()

	s.FeeModeExpedite.ConversionRate = "100"
	s.FeeModeExpedite.Flag = "1"
	s.FollowFuel = 100
	s.Name = m.Name
	s.Info = m.Info
	s.IsValued = m.IsValued
	s.EmissionAmount = m.EmissionAmount
	s.TokenSymbol = m.TokenSymbol
	s.TokenName = m.TokenName
	s.TypeWithdraw = m.TypeWithdraw
	s.TypeEmission = m.TypeEmission

	if m.Info != "" {
		minfo := make(map[string]any)
		err := json.Unmarshal([]byte(m.Info), &minfo)
		if err != nil {
			return nil, err
		}
		usid, ok := minfo["logo"]
		if ok {
			urid := escape(usid)
			uid, err := strconv.ParseInt(urid, 10, 64)
			if err != nil {
				return nil, err
			}

			hash, err := GetFileHash(uid)
			if err != nil {
				return nil, err
			}
			s.LogoHash = hash

		}
	}
	if m.FeeModeInfo != "" {
		var feeInfo feeModeInfo
		err := json.Unmarshal([]byte(m.FeeModeInfo), &feeInfo)
		if err != nil {
			return nil, err
		}
		s.FollowFuel = feeInfo.FollowFuel * 100
		for key, value := range feeInfo.FeeModeDetail {
			switch key {
			case "expedite_fee":
				s.FeeModeExpedite.ConversionRate = value.ConversionRate
				s.FeeModeExpedite.Flag = value.Flag
			}
		}
	}

	var mb Member
	f1, _ := mb.GetAccount(m.Key.Ecosystem, m.AccountID)
	//if err1 != nil {
	//	return nil, err1
	//}
	if f1 {
		if mb.ImageID != nil {
			s.MemberName = mb.MemberName
			s.MemberImageID = *mb.ImageID
			s.MemberInfo = mb.MemberInfo
			if s.MemberImageID != 0 {
				hash, err := GetFileHash(s.MemberImageID)
				if err != nil {
					return nil, err
				}
				s.MemberImageHash = hash
			}
		}
		if mb.MemberName != "" {
			s.MemberName = mb.MemberName
		}
	}

	return &s, nil
}

func GetKeyAmountByEcosystem(ecosystem int64, wallet string) (*WalletAmount, error) {
	var (
		spent SpentInfo
		rets  WalletAmount
		key   Key
	)
	keyId := converter.StringToAddress(wallet)
	if keyId == 0 {
		return nil, errors.New("request params invalid")
	}

	f, err := key.GetEcosystemKeys(ecosystem, keyId)
	if err != nil {
		return nil, err
	}
	if !f {
		return nil, errors.New("wallet doesn't exist")
	}

	utxoAmount, err := spent.GetAmountByKeyId(keyId, ecosystem)
	if err != nil {
		return nil, err
	}
	rets.UtxoAmount = utxoAmount.String()
	rets.AccountAmount = key.Amount.String()
	rets.Amount = utxoAmount.Add(key.Amount).String()
	rets.TokenSymbol = Tokens.Get(ecosystem)

	return &rets, nil
}
