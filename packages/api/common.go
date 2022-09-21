package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func JsonResponse(c *gin.Context, body *Response) {
	c.JSON(http.StatusOK, body)
}

func PureJsonResponse(c *gin.Context, body *Response) {
	c.PureJSON(http.StatusOK, body)
}

//IndentedJsonResponse Json Format
func IndentedJsonResponse(c *gin.Context, body any) {
	c.IndentedJSON(http.StatusOK, body)
}
