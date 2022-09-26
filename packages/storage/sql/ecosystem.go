package sql

import (
	"encoding/json"
	"errors"
	"github.com/IBAX-io/go-ibax/packages/converter"
	"github.com/IBAX-io/go-ibax/packages/storage/sqldb"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"reflect"
	"strconv"
	"sync"
)

var (
	Tokens   *EcosystemInfoMap
	EcoNames *EcosystemInfoMap
)

type Ecosystem struct {
	ID             int64 `gorm:"primary_key;not null"`
	Name           string
	Info           string `gorm:"type:jsonb(PostgreSQL)"`
	FeeModeInfo    string
	IsValued       int64
	EmissionAmount string `gorm:"type:jsonb(PostgreSQL)"`
	TokenSymbol    string
	TokenName      string
	TypeEmission   int64
	TypeWithdraw   int64
	ControlMode    int64
}

//EcosystemKeyTotalResult
type EcosystemKeyTotalResult struct {
	Total int64                  `json:"total"`
	Page  int                    `json:"page"`
	Limit int                    `json:"limit"`
	Rets  []EcosystemKeyTotalRet `json:"rets"`
}

// EcosystemKeyTotalRet
type EcosystemKeyTotalRet struct {
	ID              int64             `json:"id" example:"1"`            //ID
	Name            string            `json:"name" example:""`           //
	Info            string            `json:"info" example:""`           //
	IsValued        int64             `json:"isValued" example:""`       //
	EmissionAmount  string            `json:"emissionAmount" example:""` //
	TokenSymbol     string            `json:"tokenSymbol" example:""`    //
	TokenName       string            `json:"tokenName" example:""`      //
	TypeEmission    int64             `json:"typeEmission" example:""`   //
	TypeWithdraw    int64             `json:"typeWithdraw" example:""`   //
	AccountAmount   string            `json:"accountAmount" example:""`  //
	UtxoAmount      string            `json:"utxoAmount" example:""`     //
	Amount          string            `json:"amount" example:""`
	MemberName      string            `json:"memberName" example:""`      //
	MemberImageID   int64             `json:"memberImageID" example:""`   //ID
	MemberImageHash string            `json:"memberImageHash" example:""` //url
	LogoHash        string            `json:"logoHash" example:""`
	MemberInfo      string            `json:"memberInfo" example:""` //
	Account         string            `json:"account" example:""`    //
	FollowFuel      float64           `json:"followFuel" example:"100"`
	FeeModeExpedite sqldb.FeeModeFlag `json:"expedite"`
}

type combustion struct {
	Flag    int64 `json:"flag"`
	Percent int64 `json:"percent"`
}

type feeModeInfo struct {
	FeeModeDetail map[string]sqldb.FeeModeFlag `json:"fee_mode_detail"`
	Combustion    combustion                   `json:"combustion"`
	FollowFuel    float64                      `json:"follow_fuel"`
}

type EcosystemInfoMap struct {
	sync.RWMutex
	Map map[int64]string
}

func (e *Ecosystem) TableName() string {
	return "1_ecosystems"
}

func (e *Ecosystem) Get(id int64) (bool, error) {
	return isFound(GetDB(nil).First(e, "id = ?", id))
}

func (e *Ecosystem) GetTokenExist(id int64) (bool, error) {
	return isFound(GetDB(nil).Where("token_symbol <> ''").First(e, "id = ?", id))
}

func (e *Ecosystem) GetFind(limit, page int, order string, where map[string]any) ([]Ecosystem, int64, error) {
	var rs []Ecosystem
	var total int64
	if len(where) == 0 {
		if err := GetDB(nil).Table(e.TableName()).Count(&total).Error; err != nil {
			return nil, 0, err
		}
		if err := GetDB(nil).Order(order).Offset((page - 1) * limit).Limit(limit).Find(&rs).Error; err != nil {
			return nil, 0, err
		}

	} else {
		cond, vals, err := WhereBuild(where)
		if err != nil {
			return nil, 0, err
		}
		if err := GetDB(nil).Table(e.TableName()).Where(cond, vals...).Count(&total).Error; err != nil {
			return nil, 0, err
		}
		if err := GetDB(nil).Where(cond, vals...).Order(order).Offset((page - 1) * limit).Limit(limit).Find(&rs).Error; err != nil {
			return nil, 0, err
		}

	}

	return rs, total, nil
}

func (e *Ecosystem) GetTokenSymbol(id int64) (bool, error) {
	return isFound(GetDB(nil).Select("token_symbol,name").First(e, "id = ?", id))
}

// GetAllSystemStatesIDs is retrieving all ecosystems ids
func GetAllSystemStatesIDs() ([]int64, []string, error) {
	var ecosystems []Ecosystem
	if err := GetDB(nil).Select("id,name").Order("id asc").Find(&ecosystems).Error; err != nil {
		return nil, nil, err
	}

	ids := make([]int64, len(ecosystems))
	names := make([]string, len(ecosystems))
	for i, s := range ecosystems {
		ids[i] = s.ID
		names[i] = s.Name
	}

	return ids, names, nil
}

func EcosystemSearch(search any, order, account string) (*[]EcosystemSearchResponse, error) {
	var keyword string
	if search != nil {
		switch reflect.TypeOf(search).String() {
		case "string":
		default:
			log.WithFields(log.Fields{"search type": reflect.TypeOf(search).String()}).Info("ecosystem search params invalid")
			return nil, errors.New("request params invalid")
		}
		keyword = "%" + search.(string) + "%"
	} else {
		keyword = "%%"
	}
	var rets []EcosystemSearchResponse

	kid := converter.StringToAddress(account)
	if kid == 0 {
		return nil, errors.New("request params account invalid")
	}
	if order == "all" {
		query := GetDB(nil).Raw(`
SELECT name,id,COALESCE(token_symbol,'')token_symbol,true AS is_join FROM "1_ecosystems" WHERE id in(SELECT ecosystem FROM "1_keys" WHERE 
id = ?) and (name like ? or token_symbol like ?) AND token_symbol <> '' AND id <> 1

UNION

SELECT v2.name,v2.id,v2.token_symbol,
	CASE WHEN COALESCE((
			SELECT ecosystem FROM "1_keys" WHERE id = ? AND v1.ecosystem = ecosystem),0) > 0 THEN
		TRUE
	ELSE
		FALSE
	END AS is_join FROM(
	SELECT ecosystem FROM "1_parameters" WHERE name = 'free_membership' AND value = '1'
)AS v1
LEFT JOIN
(
	SELECT id,name,COALESCE(token_symbol,'')AS token_symbol FROM "1_ecosystems" 
)AS v2 ON(v1.ecosystem = v2.id)
WHERE (v2.name like ? or v2.token_symbol like ?) AND token_symbol <> '' AND id <> 1
`, kid, keyword, keyword, kid, keyword, keyword)
		if err := query.Where("token_symbol <> ''").Find(&rets).Error; err != nil {
			log.Info("Ecosystem Search failed:", err, " keyword:", keyword, " account:", account)
			return nil, errors.New("search account ecosystem failed")
		}
	} else if order == "token_join" {
		query := GetDB(nil).Table("1_ecosystems").Select("name,id,token_symbol,true as is_join").Where(`id in(SELECT ecosystem FROM "1_keys" WHERE 
id = ?) and (name like ? or token_symbol like ?)`, kid, keyword, keyword)
		if err := query.Where("token_symbol <> ''").Find(&rets).Error; err != nil {
			log.Info("Ecosystem Search failed:", err, " account:", account)
			return nil, errors.New("search account ecosystem failed")
		}
	} else {
		return nil, errors.New("request params invalid")
	}

	return &rets, nil
}

func GetFuelRate() (rlt map[int64]decimal.Decimal) {
	var pla sqldb.PlatformParameter
	f, err := pla.Get(nil, "fuel_rate")
	if err == nil && f {
		rlt = make(map[int64]decimal.Decimal)
		var values [][]string
		err = json.Unmarshal([]byte(pla.Value), &values)
		if err == nil {
			for _, v1 := range values {
				if len(v1) == 2 {
					ecoId, _ := strconv.ParseInt(v1[0], 10, 64)
					if ecoId > 0 {
						fuelRate := v1[1]
						rlt[ecoId], _ = decimal.NewFromString(fuelRate)
					}
				}
			}
		}
	}
	return
}

func GetAllTokenSymbol() ([]Ecosystem, error) {
	var (
		list []Ecosystem
	)
	err := GetDB(nil).Select("token_symbol,id").Find(&list).Error
	if err != nil {
		log.WithFields(log.Fields{"INFO": err}).Info("get all token symbol failed")
		return nil, err
	}
	return list, nil
}

func GetAllEcosystemName() ([]Ecosystem, error) {
	var (
		list []Ecosystem
	)
	err := GetDB(nil).Select("name,id").Find(&list).Error
	if err != nil {
		log.WithFields(log.Fields{"INFO": err}).Info("get all ecosystem name failed")
		return nil, err
	}
	return list, nil
}

func (p *EcosystemInfoMap) Get(ecosystem int64) string {
	p.RLock()
	defer p.RUnlock()
	value, ok := p.Map[ecosystem]
	if ok {
		return value
	}
	return ""
}

func (p *EcosystemInfoMap) Set(ecosystem int64, value string) {
	p.Lock()
	defer p.Unlock()
	p.Map[ecosystem] = value
}

func InitEcosystemInfo() {
	Tokens = &EcosystemInfoMap{
		Map: make(map[int64]string),
	}
	EcoNames = &EcosystemInfoMap{
		Map: make(map[int64]string),
	}
}

func SyncEcosystemInfo() {
	list, err := GetAllTokenSymbol()
	if err == nil {
		for _, val := range list {
			Tokens.Set(val.ID, val.TokenSymbol)
		}
	}
	list, err = GetAllEcosystemName()
	if err == nil {
		for _, val := range list {
			EcoNames.Set(val.ID, val.Name)
		}
	}
}
