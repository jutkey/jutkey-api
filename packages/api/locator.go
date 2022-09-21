package api

import (
	"github.com/gin-gonic/gin"
	"jutkey-server/packages/storage/sql"
)

func getLocatorHandler(c *gin.Context) {
	ret := &Response{}
	ipStr := sql.ClientIP(c)
	if ipStr == "" {
		ret.ReturnFailureString("Unknown IP")
		JsonResponse(c, ret)
		return
	}

	rets, err := sql.GetLocator(ipStr)
	if err != nil {
		ret.ReturnFailureString("Failed:" + err.Error())
		JsonResponse(c, ret)
		return
	}
	ret.Return(&rets, CodeSuccess)
	JsonResponse(c, ret)

}
