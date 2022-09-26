package sql

import (
	"github.com/IBAX-io/go-ibax/packages/converter"
	"github.com/shopspring/decimal"
)

// Block is model
type Block struct {
	ID             int64  `gorm:"primary_key;not_null"`
	Hash           []byte `gorm:"not null"`
	RollbacksHash  []byte `gorm:"not null"`
	Data           []byte `gorm:"not null"`
	EcosystemID    int64  `gorm:"not null"`
	KeyID          int64  `gorm:"not null"`
	NodePosition   int64  `gorm:"not null"`
	Time           int64  `gorm:"not null"`
	Tx             int32  `gorm:"not null"`
	ConsensusMode  int32  `gorm:"not null"`
	CandidateNodes []byte `gorm:"not null"`
}

// TableName returns name of table
func (Block) TableName() string {
	return "block_chain"
}

func getNodePkgInfo(nodePosition int64, consensusMode int32, ret []nodePkg) (decimal.Decimal, int64, string) {
	zero := decimal.New(0, 0)
	for i := 0; i < len(ret); i++ {
		if ret[i].NodePosition == nodePosition && ret[i].ConsensusMode == consensusMode {
			return ret[i].PkgFor, ret[i].Count, converter.AddressToString(ret[i].KeyId)
		}
	}
	return zero, 0, ""
}

func (b *Block) GetMaxBlock() (bool, error) {
	return isFound(GetDB(nil).Last(b))
}

func (b *Block) GetSystemTime() (int64, error) {
	f, err := isFound(GetDB(nil).Select("time").Where("id = 1").First(b))
	if err == nil && f {
		return b.Time, nil
	}
	return 0, err
}

func (b *Block) GetId(blockId int64) (bool, error) {
	return isFound(GetDB(nil).Where("id = ?", blockId).First(b))
}
