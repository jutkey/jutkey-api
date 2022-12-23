package sql

import (
	"github.com/shopspring/decimal"
)

type NftMinerSummaryResponse struct {
	NftMinerCount int64  `json:"nftMinerCount"`
	EnergyPower   int64  `json:"energyPower"`
	NftMinerIns   string `json:"nftMinerIns"`
	StakeAmount   string `json:"stakeAmount"`
}

type WalletMonthHistory struct {
	Month     string          `json:"month"`
	Time      int64           `json:"time"`
	InCount   int64           `json:"inCount"`
	OutCount  int64           `json:"outCount"`
	InAmount  decimal.Decimal `json:"inAmount"`
	OutAmount decimal.Decimal `json:"outAmount"`
}

type WalletMonthHistoryResponse struct {
	List        []WalletMonthHistory `json:"list"`
	TokenSymbol string               `json:"tokenSymbol"`
}

type WalletMonthDetailResponse struct {
	GeneralResponse
	TokenSymbol string `json:"tokenSymbol"`
}

type NftMinerOverviewResponse struct {
	Amount string `json:"amount"`
	Time   int64  `json:"time"`
}

type TotalResult struct {
	Total int64 `json:"total"`
}

type nftMinerInfo struct {
	Id          int64  `json:"id"`
	TokenHash   string `json:"token_hash"`
	EnergyPoint int    `json:"energy_point"`
	StakeAmount string `json:"stake_amount"`
	EnergyPower int64  `json:"energy_power"`
	Burst       int64  `json:"burst"`
	StartDated  int64  `json:"start_dated"`
	EndDated    int64  `json:"end_dated"`
}

type CommonResult struct {
	TotalResult
	IsCreate bool           `json:"isCreate"`
	Rets     []nftMinerInfo `json:"rets"`
}

type EcosystemSearchResponse struct {
	Name        string `json:"name"`
	Id          int64  `json:"id"`
	TokenSymbol string `json:"tokenSymbol"`
	IsJoin      bool   `json:"isJoin"`
	LogoHash    string `json:"logoHash"`
	Amount      string `json:"amount"`
}

type AccountHistoryTotal struct {
	InAmount    string `json:"inAmount"`
	OutAmount   string `json:"outAmount"`
	AllAmount   string `json:"allAmount"`
	InTx        int64  `json:"inTx"`
	OutTx       int64  `json:"outTx"`
	AllTx       int64  `json:"allTx"`
	TokenSymbol string `json:"tokenSymbol"`
	Ecosystem   int64  `json:"ecosystem"`
}

type GeneralResponse struct {
	Total int64 `json:"total"`
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	List  any   `json:"list"`
}

type NodeListResponse struct {
	Ranking        int64   `json:"ranking"`
	Id             int64   `json:"id"`
	IconUrl        string  `json:"iconUrl"`
	Name           string  `json:"name"`
	Website        string  `json:"website"`
	ApiAddress     string  `json:"apiAddress"`
	Address        string  `json:"address"`
	Packed         int64   `json:"packed"`
	PackedRate     string  `json:"packedRate"`
	Vote           string  `json:"vote"`
	VoteRate       string  `json:"voteRate"`
	VoteTrend      int     `json:"voteTrend"` //0:unknown 1:Up 2:Down 3:Equal
	Staking        string  `json:"staking"`
	FrontCommittee bool    `json:"frontCommittee"`
	Committee      bool    `json:"committee"`
	MyVote         float64 `json:"myVote"`
	MyStaking      string  `json:"myStaking"`
	Pending        bool    `json:"pending"`
}

type NodeDetailResponse struct {
	NodeListResponse
	StakeRate string `json:"stakeRate"`
	Account   string `json:"account"`
}

type VoteInfo struct {
	Id           int64   `json:"id"`
	Title        string  `json:"title"`
	Created      int64   `json:"created"`
	VotedRate    int     `json:"voted_rate"`
	ResultRate   float64 `json:"result_rate"`
	RejectedRate float64 `json:"rejected_rate"`
}

type NodeVoteResponse struct {
	VoteInfo
	Result int `json:"result"` //vote result
	Status int `json:"status"` //node vote status
}

type GasFee struct {
	Amount      string `json:"amount"`
	TokenSymbol string `json:"token_symbol"`
}

type NodeBlockListResponse struct {
	BlockId   int64  `json:"block_id"`
	Time      int64  `json:"time"`
	Tx        int32  `json:"tx"`
	EcoNumber int    `json:"eco_number"`
	GasFee1   GasFee `json:"gas_fee_1"` //IBXC
	GasFee2   GasFee `json:"gas_fee_2"`
	GasFee3   GasFee `json:"gas_fee_3"`
	GasFee4   GasFee `json:"gas_fee_4"`
	GasFee5   GasFee `json:"gas_fee_5"`
}

type NodeStatisticsResponse struct {
	CandidateTotal int64 `json:"candidate_total"`
	HonorTotal     int64 `json:"honor_total"`
}

type NftMinerInfoResponse struct {
	ID          int64  `json:"id"`   //NFT Miner ID
	Hash        string `json:"hash"` //NFT Miner hash
	EnergyPoint int    `json:"energyPoint"`
	StakeAmount string `json:"stakeAmount"` //starking
	StakeCount  int64  `json:"stakeCount"`
	Creator     string `json:"creator"`
	Owner       string `json:"owner"` //owner account
	RewardCount int64  `json:"rewardCount"`
	DateCreated int64  `json:"dateCreated"` //create time
	Ins         string `json:"ins"`
	EnergyPower int64  `json:"energyPower"`
	StakeStatus int64  `json:"stakeStatus"`
}

type WalletAmount struct {
	TokenSymbol   string `json:"tokenSymbol" example:""`   //
	AccountAmount string `json:"accountAmount" example:""` //
	UtxoAmount    string `json:"utxoAmount" example:""`    //
	Amount        string `json:"amount" example:""`
}

type NftMinerStakeInfoResponse struct {
	ID          int64 `json:"id"`
	NftMinerId  int64 `json:"nftMinerId"`
	StakeAmount int64 `json:"stakeAmount"` //starking
	Cycle       int64 `json:"cycle"`
	StartDated  int64 `json:"startDated"`
	EndDated    int64 `json:"endDated"`
	StakeStatus bool  `json:"stakeStatus"`
}

type NftMinerTxInfoResponse struct {
	ID         int64  `json:"id"`
	NftMinerId int64  `json:"nftMinerId"`
	Time       int64  `json:"time"`
	Ins        string `json:"ins"`
	Txhash     string `json:"txhash"`
}

type NodeStakingHistory struct {
	Hash   string `json:"hash"`
	Time   int64  `json:"time"`
	Vote   string `json:"vote,omitempty"`
	Amount string `json:"amount"`
}

type NodeStakingHistoryResponse struct {
	GeneralResponse
	TotalVote    string `json:"totalVote,omitempty"`
	Staking      string `json:"staking"`
	DateWithdraw int64  `json:"dateWithdraw"`
	GetStatus    int    `json:"getStatus"`
}

type UtxoInputResponse struct {
	Ecosystem int64  `json:"ecosystem"`
	Input     int64  `json:"input"`
	FuelRate  string `json:"fuelRate"`
}

type JoinEcosystemResponse struct {
	Id          int64  `json:"id"`
	Name        string `json:"name"`
	TokenSymbol string `json:"tokenSymbol"`
}

type keyEcosystem struct {
	Key
	Ecosystem
}

type AccountTxHistory struct {
	Id          int    `json:"id"`
	Address     string `json:"address"`
	BlockId     int64  `json:"block_id"`
	Hash        string `json:"hash"`
	Contract    string `json:"contract"`
	CreatedAt   int64  `json:"created_at"`
	TokenSymbol string `json:"token_symbol"`
	Type        int    `json:"type"`
	Recipient   string `json:"recipient"`
	Sender      string `json:"sender"`
	Amount      string `json:"amount"`
}

type MonthHistoryResponse struct {
	ID      int64  `json:"id"`
	Balance string `json:"balance"`
	Amount  string `json:"amount"`
	Time    int64  `json:"time"`
}

type historyMonthRet struct {
	Block            int64
	Hash             []byte
	SenderId         int64
	RecipientId      int64
	Type             int64
	CreatedAt        int64
	SenderBalance    string
	RecipientBalance string
	Amount           string
}

type SynthesizableResponse struct {
	Id          int64  `json:"id"`
	TokenHash   string `json:"token_hash"`
	EnergyPoint int    `json:"energy_point"`
}

type NFtMinerTransferInfoResponse struct {
	Id          int64  `json:"id"`
	TokenHash   string `json:"tokenHash"`
	EnergyPoint int    `json:"energyPoint"`
	Creator     string `json:"creator"`
	Owner       string `json:"owner"`
	MemberName  string `json:"memberName"`
	DateCreated int64  `json:"dateCreated"`
	TxHash      string `json:"txHash"`
}

type NftMinerSynthesisResponse struct {
	Id          int64  `json:"id"`
	TokenHash   string `json:"tokenHash"`
	EnergyPoint int    `json:"energyPoint"`
	Creator     string `json:"creator"`
	Owner       string `json:"owner"`
	DateCreated int64  `json:"dateCreated"`
	TxHash      string `json:"txHash"`
}

type AirdropInfoResponse struct {
	Total      string `json:"total"`
	IsGet      string `json:"is_get"`
	PerGet     string `json:"per_get"`
	Lock       string `json:"lock"`
	X5Lock     string `json:"x5_lock"`
	X5Get      string `json:"x5_get"`
	X5Period   int64  `json:"x5_period"`
	X10Lock    string `json:"x10_lock"`
	X10Get     string `json:"x10_get"`
	X10Period  int64  `json:"x10_period"`
	X20Lock    string `json:"x20_lock"`
	X20Get     string `json:"x20_get"`
	X20Period  int64  `json:"x20_period"`
	CanSpeedUp bool   `json:"can_speed_up"`
	UnLockAll  bool   `json:"un_lock_all"`
	NowSpeedUp int64  `json:"now_speed_up"`

	NextPeriod int64 `json:"next_period"`
	Surplus    int64 `json:"surplus"`
}

type AirdropBalanceResponse struct {
	Lock   decimal.Decimal `json:"lock"`
	Amount decimal.Decimal `json:"amount"`
	Show   bool            `json:"show"`
}
