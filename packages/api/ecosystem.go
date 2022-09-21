package api

import (
	"errors"
	"fmt"
	"github.com/IBAX-io/go-ibax/packages/converter"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"jutkey-server/packages/params"
	"jutkey-server/packages/storage/sql"
	"net/http"
	"os"
	"unicode/utf8"
)

type findEcosystem struct {
	params.WalletTp
	params.GeneralRequest
}

func (p *findEcosystem) Validate() (err error) {
	err = p.GeneralRequest.Validate()
	if err != nil {
		return
	}
	if p.Wallet == "" {
		return errors.New("wallet invalid")
	}
	return nil
}

// getAllEcosystemList godoc
// @Summary      get all ecosystem list
// @Description  get dashboard user ecosystem list
// @Tags         accounts
// @Accept       json
// @Produce      json
// @Param params body findEcosystem{search=string} true "params"
// @Success      200 {object} Response{data=EcosystemListResult} code:0
// @Failure      200 {object} Response{data=EcosystemListResult} code:1
// @Router       /api/v1/eco_libs [post]
func getAllEcosystemList(c *gin.Context) {
	var (
		i int
	)
	req := &findEcosystem{}
	ret := &Response{}
	err := params.ParseFrom(c, req)
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}

	var list EcosystemListResult
	list.Limit = req.Limit
	list.Page = req.Page
	var eco sql.Ecosystem
	ecosystems, total, err := eco.GetFind(req.Limit, req.Page, req.Order, req.Where)
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	} else {
		list.Total = total
		for i = 0; i < len(ecosystems); i++ {
			var ec = EcosystemList{
				ID:             ecosystems[i].ID,
				Name:           ecosystems[i].Name,
				IsValued:       ecosystems[i].IsValued,
				Info:           ecosystems[i].Info,
				EmissionAmount: ecosystems[i].EmissionAmount,
				TokenSymbol:    ecosystems[i].TokenSymbol,
				TypeEmission:   ecosystems[i].TypeEmission,
				TypeWithdraw:   ecosystems[i].TypeWithdraw,
			}

			var key sql.Key
			count, err := key.GetEcosystemsKeysCount(ecosystems[i].ID)
			if err != nil {
				ret.ReturnFailureString(err.Error())
				JsonResponse(c, ret)
				return
			}
			ec.Member = count
			f, err := key.GetEcosystemKeys(ecosystems[i].ID, converter.StringToAddress(req.Wallet))
			if err != nil {
				ret.ReturnFailureString(err.Error())
				JsonResponse(c, ret)
				return
			}
			if f {
				ec.Status = 1
			} else {
				ec.Status = 0
			}

			list.Rets = append(list.Rets, ec)
		}
		ret.Return(&list, CodeSuccess)
		JsonResponse(c, ret)
	}
}

func getEcosystemThroughKey(c *gin.Context) {
	req := &findEcosystem{}
	ret := &Response{}
	err := params.ParseFrom(c, req)
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}
	data := &sql.Key{}
	if req.Wallet == "" {
		ret.ReturnFailureString("wallet is invalid")
		JsonResponse(c, ret)
		return
	}
	keyid := converter.StringToAddress(req.Wallet)
	mines, err := data.GetEcosystemsKeyAmount(keyid, req.Page, req.Limit, req.Order, req.Search)
	if err != nil {
		if err.Error() == "record not found" {
			ret.Return(nil, CodeSuccess)
		} else {
			ret.ReturnFailureString("get ecosystem key amount err:" + err.Error())
		}
		JsonResponse(c, ret)
		return
	}
	ret.Return(mines, CodeSuccess)
	JsonResponse(c, ret)

}

func ecosystemSearchHandler(c *gin.Context) {
	type reqType struct {
		params.GeneralRequest
		params.WalletTp
	}
	req := &reqType{}

	ret := &Response{}
	if err := c.ShouldBindWith(req, binding.JSON); err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}
	if req.Wallet == "" || req.Order == "" {
		ret.ReturnFailureString("request params invalid")
		JsonResponse(c, ret)
		return
	}
	rets, err := sql.EcosystemSearch(req.Search, req.Order, req.Wallet)
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}
	ret.Return(&rets, CodeSuccess)
	JsonResponse(c, ret)
}

func getStatisticsHandler(c *gin.Context) {
	var rets sql.Statistics
	ret := &Response{}
	f, err := rets.GetRedis()
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}
	if !f {
		ret.Return(&rets, CodeSuccess)
		JsonResponse(c, ret)
		return
	}
	ret.Return(&rets, CodeSuccess)
	JsonResponse(c, ret)
}

func getAttachmentHandler(c *gin.Context) {
	ret := &Response{}
	hash := c.Param("hash")
	if hash == "" || utf8.RuneCountInString(hash) > 100 {
		ret.ReturnFailureString("Request params invalid")
		JsonResponse(c, ret)
		return
	}

	fileName, id, err := sql.GetFileNameByHash(hash)
	if err != nil {
		ret.ReturnFailureString("Get attachment failed:" + err.Error())
		JsonResponse(c, ret)
		return
	}
	if fileName == "" {
		ret.ReturnFailureString("Get attachment failed:File doesn't not exist")
		JsonResponse(c, ret)
		return
	}
	//Save the file to the local. If the file does not exist, search for the file from the database
	if !sql.IsExist(sql.UploadDir + fileName) {
		fileName, err = sql.LoadFile(id)
		if err != nil {
			ret.ReturnFailureString("loadFile failed:" + err.Error())
			JsonResponse(c, ret)
			return
		}
		if fileName == "" {
			ret.ReturnFailureString("hash doesn't not exist")
			JsonResponse(c, ret)
			return
		}

	}

	data, err := os.ReadFile(sql.UploadDir + fileName)
	if err != nil {
		ret.ReturnFailureString("Get attachment read file failed:" + err.Error())
		JsonResponse(c, ret)
		return
	}

	if !sql.CompareHash(data, hash) {
		ret.ReturnFailureString("Hash is incorrect")
		JsonResponse(c, ret)
		return
	}

	//c.Header("Content-Type", bin.MimeType)
	c.Header("Content-Type", http.DetectContentType(data))
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	c.Header("Access-Control-Allow-Origin", "*")
	_, err = c.Writer.Write(data)
	if err != nil {
		ret.ReturnFailureString("Attachment File Write Error:" + err.Error())
		JsonResponse(c, ret)
		return
	}
}

func getKeyAmountHandler(c *gin.Context) {
	type reqType struct {
		params.EcosystemTp
		params.WalletTp
	}
	req := &reqType{}

	ret := &Response{}
	if err := c.ShouldBindWith(req, binding.JSON); err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}
	if req.Ecosystem <= 0 || req.Wallet == "" {
		ret.ReturnFailureString("request params invalid")
		JsonResponse(c, ret)
		return
	}

	rets, err := sql.GetKeyAmountByEcosystem(req.Ecosystem, req.Wallet)
	if err != nil {
		ret.ReturnFailureString(err.Error())
		JsonResponse(c, ret)
		return
	}
	ret.Return(&rets, CodeSuccess)
	JsonResponse(c, ret)
}
