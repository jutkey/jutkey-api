package api

import (
	"fmt"
	"github.com/IBAX-io/go-ibax/packages/consts"
	"github.com/IBAX-io/go-ibax/packages/converter"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	log "github.com/sirupsen/logrus"
	"jutkey-server/packages/params"
	"jutkey-server/packages/storage/sql"
)

func userNftMinerSummaryHandler(c *gin.Context) {
	req := &params.WalletTp{}
	ret := &Response{}
	err := params.ParseFrom(c, req)
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}

	data := sql.NftMinerItems{}
	mines, err := data.GetUserNftMinerSummary(req.Wallet)
	if err != nil {
		log.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("getting Get Nft Miner Summary")
		if err.Error() == "record not found" {
			ret.Return(nil, CodeSuccess)
		} else {
			ret.Return(nil, CodeDBfinderr.Errorf(err))
		}
		JsonResponse(c, ret)
		return
	}
	ret.Return(mines, CodeSuccess)
	JsonResponse(c, ret)
}

func nftMinerDayRewardHandler(c *gin.Context) {
	req := &params.WalletTp{}
	ret := &Response{}
	err := params.ParseFrom(c, req)
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}
	var rets sql.NftMinerItems
	list, err := rets.GetUserNftFifteenDayOverview(15, req.Wallet)
	if err != nil {
		log.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("getting GetNftFifteendayOverView")
		if err.Error() == "record not found" {
			ret.Return(nil, CodeSuccess)
		} else {
			ret.Return(nil, CodeDBfinderr.Errorf(err))
		}
		JsonResponse(c, ret)
		return
	}
	ret.Return(list, CodeSuccess)
	JsonResponse(c, ret)

}

func getNftMinerKeyInfosHandler(c *gin.Context) {
	req := &params.WalletTp{}
	ret := &Response{}
	err := params.ParseFrom(c, req)
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}
	items := &sql.NftMinerItems{}
	res, err := items.GetNftMinerKeyInfo(req.Wallet)
	if err != nil {
		ret.Return(nil, CodeDBfinderr.Errorf(err))
		JsonResponse(c, ret)
		return
	}

	ret.Return(res, CodeSuccess)
	JsonResponse(c, ret)
}

func getNftMinerRewardHistoryHandler(c *gin.Context) {
	req := &params.GeneralRequest{}
	ret := &Response{}
	err := params.ParseFrom(c, req)
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}
	if req.Search == nil {
		ret.ReturnFailureString("request params invalid")
		JsonResponse(c, ret)
		return
	}
	items := &sql.NftMinerItems{}
	if !sql.HasTable(items) {
		ret.Return(nil, CodeSuccess)
		JsonResponse(c, ret)
		return
	}
	res, err := items.GetNftMinerRewardHistory(req.Search, req.Page, req.Limit)
	if err != nil {
		ret.Return(nil, CodeDBfinderr.Errorf(err))
		JsonResponse(c, ret)
		return
	}

	ret.Return(res, CodeSuccess)
	JsonResponse(c, ret)
}

func getNftMinerDetailHandler(c *gin.Context) {
	var req struct {
		params.SearchTp
		params.WalletTp
	}
	ret := &Response{}
	if !sql.NftMinerReady {
		ret.Return(nil, CodeSuccess)
		JsonResponse(c, ret)
		return
	}

	err := c.ShouldBindWith(&req, binding.JSON)
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}
	err = req.SearchTp.Validate()
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

	items := &sql.NftMinerItems{}
	rets, err := items.GetNftMinerDetailBySearch(req.Search, req.Wallet)
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}
	ret.Return(rets, CodeSuccess)
	JsonResponse(c, ret)

}

func getNftMinerStakingHandler(c *gin.Context) {
	var req struct {
		params.GeneralRequest
		params.WalletTp
	}
	ret := &Response{}
	err := c.ShouldBindWith(&req, binding.JSON)
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}
	err = req.GeneralRequest.Validate()
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

	stak := &sql.NftMinerStaking{}
	if !sql.NftMinerReady {
		ret.Return(nil, CodeSuccess)
		JsonResponse(c, ret)
		return
	}
	rets, err := stak.GetNftMinerStakeInfo(req.Search, req.Page, req.Limit, req.Order, req.Wallet)
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}
	ret.Return(rets, CodeSuccess)
	JsonResponse(c, ret)
}

func getNftMinerRewardHandler(c *gin.Context) {
	var req struct {
		params.GeneralRequest
		params.WalletTp
	}
	ret := &Response{}
	err := c.ShouldBindWith(&req, binding.JSON)
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}
	err = req.GeneralRequest.Validate()
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

	if !sql.NftMinerReady {
		ret.Return(nil, CodeSuccess)
		JsonResponse(c, ret)
		return
	}
	items := &sql.NftMinerItems{}
	rets, err := items.GetNftMinerTxInfo(req.Search, req.Page, req.Limit, req.Order, req.Wallet)
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}
	ret.Return(rets, CodeSuccess)
	JsonResponse(c, ret)
}

func getNftMinerFileHandler(c *gin.Context) {
	ret := &Response{}
	idStr := c.Param("id")
	id := converter.StrToInt64(idStr)
	if id <= 0 {
		ret.ReturnFailureString("request params invalid")
		JsonResponse(c, ret)
		return
	}
	var items sql.NftMinerItems
	f, err := items.GetId(id)
	if err != nil {
		ret.ReturnFailureString("request error:" + err.Error())
		JsonResponse(c, ret)
		return
	}
	if !f {
		ret.ReturnFailureString("unknown nft Miner id:" + idStr)
		JsonResponse(c, ret)
		return
	}
	data, err := items.ParseSvgParams()
	if err != nil {
		ret.ReturnFailureString("Get Nft Miner File Failed")
		JsonResponse(c, ret)
		return
	}
	c.Header("Content-Type", "image/svg+xml;utf8")
	c.Header("Access-Control-Allow-Origin", "*")
	_, err = c.Writer.Write([]byte(data))
	if err != nil {
		ret.ReturnFailureString("Get Nft Miner File Handler Write Error:" + err.Error())
		JsonResponse(c, ret)
		return
	}

}

func getNftMinerSynthesizableHandler(c *gin.Context) {
	req := &params.WalletSearchTp{}
	ret := &Response{}
	err := params.ParseFrom(c, req)
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}
	items := &sql.NftMinerItems{}
	res, err := items.NftMinerSynthesizable(req.Wallet, req.Search)
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}

	ret.Return(res, CodeSuccess)
	JsonResponse(c, ret)
}

func getNftMinerSynthesisInfoHandler(c *gin.Context) {
	ret := &Response{}
	txHash := c.Param("txHash")
	if txHash == "" {
		ret.ReturnFailureString("request params invalid")
		JsonResponse(c, ret)
		return
	}
	events := &sql.NftMinerEvents{}
	res, err := events.NftMinerSynthesisInfo(txHash)
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}

	ret.Return(res, CodeSuccess)
	JsonResponse(c, ret)
}

func getNftMinerTransferInfoHandler(c *gin.Context) {
	ret := &Response{}
	idStr := c.Param("id")
	id := converter.StrToInt64(idStr)
	if id <= 0 {
		ret.ReturnFailureString(fmt.Sprintf("request params invalid:%s", idStr))
		JsonResponse(c, ret)
		return
	}
	source := c.Param("source")
	target := c.Param("target")
	if source == "" || target == "" {
		ret.ReturnFailureString("request params invalid")
		JsonResponse(c, ret)
		return
	}

	items := &sql.NftMinerEvents{}
	res, err := items.NftMinerTransferInfo(id, source, target)
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}

	ret.Return(res, CodeSuccess)
	JsonResponse(c, ret)
}
