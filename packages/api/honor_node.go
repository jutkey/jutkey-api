package api

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"jutkey-server/packages/params"
	"jutkey-server/packages/storage/sql"
)

func getNodeStatisticsHandler(c *gin.Context) {
	ret := &Response{}
	if !sql.NodeReady {
		ret.Return(nil, CodeSuccess)
		JsonResponse(c, ret)
		return
	}
	rets, err := sql.GetNodeStatistics()
	if err != nil {
		ret.ReturnFailureString("Get Node Statistics Failed")
		JsonResponse(c, ret)
		return
	}
	ret.Return(rets, CodeSuccess)
	JsonResponse(c, ret)
}

func getHonorNodeListHandler(c *gin.Context) {
	type reqType struct {
		params.GeneralRequest
		params.WalletTp
	}
	var req reqType
	ret := &Response{}
	err := c.ShouldBindWith(&req, binding.JSON)
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}

	if err = req.GeneralRequest.Validate(); err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}
	if err = req.WalletTp.Validate(); err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}

	rets, err := sql.NodeListSearch(req.Page, req.Limit, req.Order, req.Wallet)
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}

	ret.Return(rets, CodeSuccess)
	JsonResponse(c, ret)
}

func nodeDetailHandler(c *gin.Context) {
	ret := &Response{}
	type reqType struct {
		params.WalletTp
		params.SearchTp
	}
	req := &reqType{}
	err := c.ShouldBindWith(&req, binding.JSON)
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}
	if err = req.WalletTp.Validate(); err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}
	if err = req.SearchTp.Validate(); err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}

	rets, err := sql.NodeDetail(req.Search, req.Wallet)
	if err != nil {
		ret.ReturnFailureString("Get Node Detail Failed")
		JsonResponse(c, ret)
		return
	}
	ret.Return(rets, CodeSuccess)
	JsonResponse(c, ret)

}

func getNodeDaoVoteListHandler(c *gin.Context) {
	ret := &Response{}

	req := &params.GeneralRequest{}
	err := params.ParseFrom(c, req)
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}
	if req.Search == nil {
		ret.ReturnFailureString("request params search invalid")
		JsonResponse(c, ret)
		return
	}
	rets, err := sql.GetDaoVoteList(req.Search, req.Page, req.Limit)
	if err != nil {
		ret.ReturnFailureString("Get Node Dao Vote List Failed")
		JsonResponse(c, ret)
		return
	}
	ret.Return(rets, CodeSuccess)
	JsonResponse(c, ret)
}

func getNodeBlockListHandler(c *gin.Context) {
	ret := &Response{}
	req := &params.GeneralRequest{}
	err := params.ParseFrom(c, req)
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}
	if req.Search == nil {
		ret.ReturnFailureString("request params search invalid")
		JsonResponse(c, ret)
		return
	}
	rets, err := sql.GetNodeBlockList(req.Search, req.Page, req.Limit, req.Order)
	if err != nil {
		ret.ReturnFailureString("Get Node Block List Failed")
		JsonResponse(c, ret)
		return
	}
	ret.Return(rets, CodeSuccess)
	JsonResponse(c, ret)
}

func getNodeVoteHistoryHandler(c *gin.Context) {
	ret := &Response{}
	req := &params.HistoryFindForm{}
	err := params.ParseFrom(c, req)
	if err = req.WalletTp.Validate(); err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}
	if req.Search == nil {
		ret.ReturnFailureString("request params node id invalid")
		JsonResponse(c, ret)
		return
	}

	rets, err := sql.GetNodeVoteHistory(req, 1)
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}
	ret.Return(rets, CodeSuccess)
	JsonResponse(c, ret)
}

func getNodeSubstituteHistoryHandler(c *gin.Context) {
	ret := &Response{}
	req := &params.HistoryFindForm{}
	err := params.ParseFrom(c, req)
	if err = req.WalletTp.Validate(); err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}
	if req.Search == nil {
		ret.ReturnFailureString("request params node id invalid")
		JsonResponse(c, ret)
		return
	}

	rets, err := sql.GetNodeVoteHistory(req, 2)
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}
	ret.Return(rets, CodeSuccess)
	JsonResponse(c, ret)
}
