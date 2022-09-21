package api

import (
	"github.com/IBAX-io/go-ibax/packages/consts"
	"github.com/IBAX-io/go-ibax/packages/converter"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"jutkey-server/packages/params"
	"jutkey-server/packages/storage/sql"
)

func monthHistoryDetailHandler(c *gin.Context) {
	req := &params.HistoryFindForm{}
	ret := &Response{}
	err := params.ParseFrom(c, req)
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}
	var data sql.History
	mines, err := data.GetMonthFind(req)
	if err != nil {
		if err.Error() == "record not found" {
			ret.Return(nil, CodeSuccess)
		} else {
			ret.ReturnFailureString(err.Error())
		}
		JsonResponse(c, ret)
		return
	}
	ret.Return(mines, CodeSuccess)
	JsonResponse(c, ret)
}

func monthHistoryTotalHandler(c *gin.Context) {
	type reqType struct {
		params.WalletTp
		params.EcosystemTp
	}
	req := &reqType{}
	ret := &Response{}
	err := c.ShouldBindWith(&req, binding.JSON)
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}
	err = req.WalletTp.Validate()
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}
	err = req.EcosystemTp.Validate()
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}
	data := sql.History{}
	var rets sql.WalletMonthHistoryResponse
	list, err := data.GetWalletMonthHistoryTotals(req.Ecosystem, converter.StringToAddress(req.Wallet), 3)
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}
	rets.List = list
	rets.TokenSymbol, _ = sql.GetEcosystemTokenSymbol(req.Ecosystem)

	ret.Return(&rets, CodeSuccess)
	JsonResponse(c, ret)
}

func getHistoryHandler(c *gin.Context) {
	req := &params.MineHistoryRequest{}
	ret := &Response{}
	err := params.ParseFrom(c, req)
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}
	var h sql.History
	rlt, err := h.GetList(req)
	if err != nil {
		if err.Error() == "opt params invalid" {
			ret.Return(nil, CodeParam.Errorf(err))
			JsonResponse(c, ret)
			return
		}
		log.WithFields(log.Fields{"type": consts.DBError, "error": err}).Warn("find history list")
		if err == gorm.ErrRecordNotFound {
			ret.Return(nil, CodeSuccess)
		} else {
			ret.Return(nil, CodeDBfinderr.Errorf(err))
		}
		JsonResponse(c, ret)
		return
	}

	ret.Return(rlt, CodeSuccess)
	JsonResponse(c, ret)
}

func getKeyTotalHandler(c *gin.Context) {
	type reqType struct {
		params.WalletTp
		params.EcosystemTp
	}
	req := &reqType{}
	ret := &Response{}
	err := c.ShouldBindWith(req, binding.JSON)
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}
	if req.Wallet == "" || req.Ecosystem <= 0 {
		ret.ReturnFailureString("request params invalid")
		JsonResponse(c, ret)
		return
	}
	rlt, err := sql.GetAccountHistoryTotal(req.Wallet, req.Ecosystem)
	if err != nil {
		ret.Return(nil, CodeDBfinderr.Errorf(err))
		JsonResponse(c, ret)
		return
	}

	ret.Return(rlt, CodeSuccess)
	JsonResponse(c, ret)
}
