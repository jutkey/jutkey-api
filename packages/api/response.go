package api

type Response struct {
	Code    int    `json:"code" `
	Data    any    `json:"data" `
	Message string `json:"message" `
}

func (r *Response) ReturnFailureString(str string) {
	r.Code = -1
	r.Message = str
}
func (r *Response) Return(dat any, ct CodeType) {
	r.Code = ct.Code
	r.Message = ct.Message
	r.Data = dat
}

type EcosystemList struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	Info           string `json:"info"`
	IsValued       int64  `json:"isValued"`
	EmissionAmount string `json:"emissionAmount"`
	TokenSymbol    string `json:"tokenSymbol"`
	TypeEmission   int64  `json:"typeEmission"`
	TypeWithdraw   int64  `json:"typeWithdraw"`
	Member         int64  `json:"member"`
	Status         int    `json:"status"` //0:Not joined 1:join
}

//EcosystemListResult example
type EcosystemListResult struct {
	Total int64           `json:"total"`
	Page  int             `json:"page"`
	Limit int             `json:"limit"`
	Rets  []EcosystemList `json:"rets"`
}
