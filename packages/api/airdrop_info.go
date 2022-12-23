package api

import (
	"errors"
	"fmt"
	"github.com/IBAX-io/go-ibax/packages/converter"
	"github.com/gin-gonic/gin"
	"jutkey-server/packages/storage/sql"
)

func GetAirdropInfoHandler(c *gin.Context) {
	wallet := c.Param("wallet")

	ret := &Response{}
	keyId := converter.StringToAddress(wallet)
	if keyId == 0 {
		ret.Return(nil, CodeRequestformat.Errorf(errors.New("request params wallet invalid:"+wallet)))
		JsonResponse(c, ret)
		return
	}
	rets := &sql.AirdropInfoResponse{}
	var (
		info = &sql.AirdropInfo{}
		err  error
	)
	if !sql.AirdropReady {
		ret.Return(rets, CodeSuccess)
		JsonResponse(c, ret)
		return
	}
	info.Account = wallet
	rets, err = info.GetAirdropInfo()
	if err != nil {
		ret.ReturnFailureString(fmt.Sprintf("get airdrop info failed:%s", err.Error()))
		JsonResponse(c, ret)
		return
	}
	ret.Return(rets, CodeSuccess)
	JsonResponse(c, ret)
}

func GetAirdropBalanceHandler(c *gin.Context) {
	wallet := c.Param("wallet")

	ret := &Response{}
	keyId := converter.StringToAddress(wallet)
	if keyId == 0 {
		ret.Return(nil, CodeRequestformat.Errorf(errors.New("request params wallet invalid:"+wallet)))
		JsonResponse(c, ret)
		return
	}
	var (
		rets = &sql.AirdropBalanceResponse{}
		err  error
	)
	if !sql.AirdropReady {
		ret.Return(rets, CodeSuccess)
		JsonResponse(c, ret)
		return
	}
	var info = &sql.AirdropInfo{}
	info.Account = wallet
	rets, err = info.GetAirdropBalance()
	if err != nil {
		ret.ReturnFailureString(fmt.Sprintf("get airdrop balance failed:%s", err.Error()))
		JsonResponse(c, ret)
		return
	}
	ret.Return(rets, CodeSuccess)
	JsonResponse(c, ret)

}
