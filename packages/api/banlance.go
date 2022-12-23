package api

import (
	"encoding/json"
	"errors"
	"github.com/IBAX-io/go-ibax/packages/conf/syspar"
	"github.com/IBAX-io/go-ibax/packages/consts"
	"github.com/IBAX-io/go-ibax/packages/converter"
	"github.com/IBAX-io/go-ibax/packages/storage/sqldb"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"jutkey-server/packages/params"
	"jutkey-server/packages/storage/sql"
)

func getMyAssignBalanceHandler(c *gin.Context) {
	wallet := c.Param("wallet")

	ret := &Response{}
	keyId := converter.StringToAddress(wallet)
	if keyId == 0 {
		log.WithFields(log.Fields{"type": consts.ConversionError, "value": wallet}).Error("assign balance converting wallet to address")
		ret.Return(nil, CodeRequestformat.Errorf(errors.New("request params wallet invalid:"+wallet)))
		JsonResponse(c, ret)
		return
	}

	assign := &sql.AssignInfo{}
	if !sql.HasTable(assign) {
		ret.Return(nil, CodeSuccess)
		JsonResponse(c, ret)
		return
	}
	show, balance, totalBalance, err := assign.GetBalance(nil, wallet)
	if err != nil {
		log.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("assign get balance")
		if err == gorm.ErrRecordNotFound {
			ret.Return(nil, CodeSuccess)
		} else {
			ret.Return(nil, CodeDBfinderr.Errorf(err))
		}
		JsonResponse(c, ret)
		return
	}
	if balance.Equal(decimal.Zero) && totalBalance.Equal(decimal.Zero) {
		show = false
	}

	ret.Return(sql.MyAssignBalanceResult{
		Amount:  balance.String(),
		Balance: totalBalance.String(),
		Show:    show,
	}, CodeSuccess)
	JsonResponse(c, ret)
}

func getKeyInfoHandler(c *gin.Context) {
	var found bool
	keysList := make([]*sql.KeyEcosystemInfo, 0)
	var account string
	ret := &Response{}

	keyid := converter.StringToAddress(c.Param("account"))
	if keyid == 0 {
		log.WithFields(log.Fields{"type": consts.ConversionError, "value": c.Param("account")}).Error("converting wallet to address")
		ret.Return(nil, CodeRequestformat.Errorf(errors.New("account params invalid:"+c.Param("account"))))
		JsonResponse(c, ret)
		return
	}

	ids, names, err := sql.GetAllSystemStatesIDs()
	if err != nil {
		ret.Return(nil, CodeDBfinderr.Errorf(err))
		JsonResponse(c, ret)
		return
	}

	for i, ecosystemID := range ids {
		key := &sqldb.Key{}
		key.SetTablePrefix(ecosystemID)
		found, err = key.Get(nil, keyid)
		if err != nil {
			ret.Return(nil, CodeDBfinderr.Errorf(err))
			JsonResponse(c, ret)
			return
		}
		if !found {
			continue
		}

		account = key.AccountID

		keyRes := &sql.KeyEcosystemInfo{
			Ecosystem: converter.Int64ToStr(ecosystemID),
			Name:      names[i],
		}
		ra := &sqldb.RolesParticipants{}
		roles, err := ra.SetTablePrefix(ecosystemID).GetActiveMemberRoles(key.AccountID)
		if err != nil {
			ret.Return(nil, CodeDBfinderr.Errorf(err))
			JsonResponse(c, ret)
			return
		}
		for _, r := range roles {
			var role sql.RoleInfo
			if err := json.Unmarshal([]byte(r.Role), &role); err != nil {
				log.WithFields(log.Fields{"type": consts.JSONUnmarshallError, "error": err}).Error("unmarshalling role")
				ret.Return(nil, CodeJsonformaterr.Errorf(err))
				JsonResponse(c, ret)
				return
			}
			keyRes.Roles = append(keyRes.Roles, role)
		}
		keyRes.Notifications, err = sql.GetNotifications(ecosystemID, key)
		if err != nil {
			log.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("getting notifications")
			ret.Return(nil, CodeJsonformaterr.Errorf(err))
			JsonResponse(c, ret)
			return
		}

		keysList = append(keysList, keyRes)
	}

	// in test mode, registration is open in the first ecosystem
	if len(keysList) == 0 && syspar.IsTestMode() {
		account = converter.AddressToString(keyid)
		notify := make([]sql.NotifyInfo, 0)
		notify = append(notify, sql.NotifyInfo{})
		keysList = append(keysList, &sql.KeyEcosystemInfo{
			Ecosystem:     converter.Int64ToStr(ids[0]),
			Name:          names[0],
			Notifications: notify,
		})
	}
	info := &sql.KeyInfoResult{
		Account:    account,
		Ecosystems: keysList,
	}
	ret.Return(info, CodeSuccess)
	JsonResponse(c, ret)

}

func getUtxoInputHandler(c *gin.Context) {
	ret := &Response{}
	type reqType struct {
		Search []int64
		params.WalletTp
	}
	var req reqType
	err := c.ShouldBindWith(&req, binding.JSON)
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}
	if req.Wallet == "" || req.Search == nil {
		ret.ReturnFailureString("request params invalid")
		JsonResponse(c, ret)
		return
	}
	keyId := converter.StringToAddress(req.Wallet)
	if keyId == 0 {
		log.WithFields(log.Fields{"type": consts.ConversionError, "wallet": req.Wallet}).Error("converting wallet to address")
		ret.Return(nil, CodeRequestformat.Errorf(errors.New("wallet params invalid:"+req.Wallet)))
		JsonResponse(c, ret)
		return
	}
	rets, err := sql.GetUtxoInput(keyId, req.Search)
	if err != nil {
		ret.Return(nil, CodeDBfinderr.Errorf(err))
		JsonResponse(c, ret)
		return
	}

	ret.Return(rets, CodeSuccess)
	JsonResponse(c, ret)
}
