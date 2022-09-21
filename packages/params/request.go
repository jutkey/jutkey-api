package params

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

const (
	defaultLimit = 10
	maxLimit     = 1000
)

type GeneralRequest struct {
	Search any            `json:"search,omitempty"`
	Page   int            `json:"page,omitempty" example:"1"` //Find the number of pages
	Limit  int            `json:"limit" example:"10"`         //Find the number of limit
	Order  string         `json:"order" example:"id asc"`     //order by
	Where  map[string]any `json:"where"`                      //sql where,example:Find id = 2 "where":{"id =":2}
}

type paramsValidator interface {
	Validate() error
}

type WalletTp struct {
	Wallet string `json:"wallet" example:"xxxx-xxxx-xxxx-xxxx-xxxx"` //wallet address
}

type EcosystemTp struct {
	Ecosystem int64 `json:"ecosystem"`
}

type TimeTp struct {
	Time int64 `json:"time"`
}

type SearchTp struct {
	Search any `json:"search"`
}

//HistoryFindForm example
type HistoryFindForm struct {
	GeneralRequest
	WalletTp
	TimeTp
	EcosystemTp
}

type MineHistoryRequest struct {
	EcosystemTp
	GeneralRequest
	WalletTp
	Opt string `json:"opt"`
}

type HonorNodeStakingInfoRequest struct {
	Ids []int64 `json:"ids"`
	WalletTp
	ReqType int `json:"reqType"`
}

func ParseFrom(c *gin.Context, p paramsValidator) (err error) {
	err = c.ShouldBindWith(&p, binding.JSON)
	if err != nil {
		return
	}
	return p.Validate()
}

func (p *HistoryFindForm) Validate() error {
	err := p.GeneralRequest.Validate()
	if err != nil {
		return err
	}
	err = p.WalletTp.Validate()
	if err != nil {
		return err
	}
	err = p.EcosystemTp.Validate()
	if err != nil {
		return err
	}
	if p.Order == "" {
		p.Order = "id desc"
	}

	return nil
}

func (p *MineHistoryRequest) Validate() error {
	err := p.GeneralRequest.Validate()
	if err != nil {
		return err
	}
	if p.Opt != "send" && p.Opt != "recipient" && p.Opt != "all" {
		return fmt.Errorf("params invalid! opt:%s", p.Opt)
	}
	err = p.WalletTp.Validate()
	if err != nil {
		return err
	}
	err = p.EcosystemTp.Validate()
	if err != nil {
		return err
	}
	if p.Order == "" {
		p.Order = "id DESC"
	} else {
		p.Order += ", id DESC"
	}

	return nil
}

func (p *WalletTp) Validate() error {
	if p.Wallet == "" {
		return errors.New("wallet address Can not be empty")
	}

	return nil
}

func (p *EcosystemTp) Validate() error {
	if p.Ecosystem <= 0 {
		return errors.New("ecosystem id invalid")
	}
	return nil
}

func (p *GeneralRequest) Validate() error {
	if p.Page <= 0 {
		return fmt.Errorf("request params invalid! page:%d", p.Page)
	}
	if p.Limit <= 0 {
		p.Limit = defaultLimit
	}
	if p.Limit > maxLimit {
		p.Limit = maxLimit
	}

	return nil
}

func (p *HonorNodeStakingInfoRequest) Validate() error {
	err := p.WalletTp.Validate()
	if err != nil {
		return err
	}
	maxLen := 100
	reqLen := len(p.Ids)
	if reqLen == 0 {
		return errors.New("request params ids invalid")
	}
	if reqLen > maxLen {
		return errors.New("request params ids length cannot be greater than 100")
	}

	return nil
}

func (p *SearchTp) Validate() error {
	if p.Search == nil {
		return errors.New("request params search invalid")
	}
	return nil
}
