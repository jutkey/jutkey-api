package sql

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/IBAX-io/go-ibax/packages/converter"
)

type NftMinerEvents struct {
	ID           int64  `gorm:"primary_key;not null"`
	TokenId      int64  `gorm:"column:token_id;not null"`
	TokenHash    []byte `gorm:"column:token_hash;not null"`
	Event        string `gorm:"column:event;not null"`
	ContractName string `gorm:"column:contract_name;not null"`
	DateCreated  int64  `gorm:"column:date_created;not null"`
	TxHash       []byte `gorm:"column:tx_hash;not null"`
	Source       string `gorm:"column:source"`
}

func (p *NftMinerEvents) TableName() string {
	return "1_nft_miner_events"
}

func (p *NftMinerEvents) GetByTokenId(tokenId int64) (bool, error) {
	return isFound(GetDB(nil).Where("token_id = ?", tokenId).First(&p))
}

func (p *NftMinerEvents) GetSource(tokenId int64, source, event string) (bool, error) {
	return isFound(GetDB(nil).Where("token_id = ? AND source = ? AND event = ?", tokenId, source, event).First(&p))
}

func (p *NftMinerEvents) GetSynthesis(source string) (bool, error) {
	return isFound(GetDB(nil).Where("source = ? AND event = 'Synthesis'", source).First(&p))
}

func (p *NftMinerEvents) NftMinerTransferInfo(tokenId int64, source, target string) (*NFtMinerTransferInfoResponse, error) {
	kid := converter.StringToAddress(source)
	if kid == 0 {
		return nil, fmt.Errorf("source account invalid:%s", source)
	}
	kid = converter.StringToAddress(target)
	if kid == 0 {
		return nil, fmt.Errorf("target account invalid:%s", target)
	}
	it := &NftMinerItems{}
	f, err := it.GetId(tokenId)
	if err != nil {
		return nil, err
	}
	if !f {
		return nil, errors.New("NFT doesn't not exist")
	}
	if it.Owner != target {
		return nil, errors.New("record doesn't not exist")
	}

	f, err = p.GetSource(tokenId, source, "Transfer")
	if err != nil {
		return nil, err
	}
	if !f {
		return nil, errors.New("record doesn't not exist")
	}
	rets := &NFtMinerTransferInfoResponse{}
	rets.TxHash = hex.EncodeToString(p.TxHash)
	rets.TokenHash = hex.EncodeToString(p.TokenHash)
	rets.Id = p.TokenId

	rets.Creator = it.Creator
	rets.DateCreated = it.DateCreated
	rets.Owner = it.Owner
	rets.EnergyPoint = it.EnergyPoint
	var mb Member
	f, _ = mb.GetAccount(1, it.Owner)
	if f {
		if mb.MemberName != "" {
			rets.MemberName = mb.MemberName
		}
	}

	return rets, nil
}

func (p *NftMinerEvents) NftMinerSynthesisInfo(txHashStr string) (*NftMinerSynthesisResponse, error) {
	_, err := hex.DecodeString(txHashStr)
	if err != nil {
		return nil, errors.New("request params invalid:" + err.Error())
	}

	f, err := p.GetSynthesis(txHashStr)
	if err != nil {
		return nil, err
	}
	if !f {
		return nil, errors.New("record doesn't not exist")
	}

	items := &NftMinerItems{}
	f, err = items.GetId(p.TokenId)
	if err != nil {
		return nil, err
	}
	if !f {
		return nil, errors.New("record doesn't not exist")
	}

	rets := &NftMinerSynthesisResponse{}
	rets.TokenHash = hex.EncodeToString(p.TokenHash)
	rets.Id = p.TokenId
	rets.EnergyPoint = items.EnergyPoint
	rets.DateCreated = items.DateCreated
	rets.Owner = items.Owner
	rets.Creator = items.Creator
	rets.TxHash = txHashStr

	return rets, nil
}
