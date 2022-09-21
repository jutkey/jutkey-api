package api

import (
	"github.com/gin-gonic/gin"
	"jutkey-server/packages/services"
)

func getWebsocketToken(c *gin.Context) {
	ret := &Response{}
	rets, err := services.GetJWTCentToken(1, 60*60)
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
	} else {
		ret.Return(rets, CodeSuccess)
		JsonResponse(c, ret)
	}
}
