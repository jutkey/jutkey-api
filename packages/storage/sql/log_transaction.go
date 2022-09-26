package sql

type LogTransaction struct {
	Hash         []byte `gorm:"primary_key;not null"`
	Block        int64  `gorm:"not null"`
	Timestamp    int64  `gorm:"not null"`
	ContractName string `gorm:"not null"`
	Address      int64  `gorm:"not null"`
	EcosystemID  int64  `gorm:"not null"`
	Status       int64  `gorm:"not null"`
}

func (m LogTransaction) TableName() string {
	return `log_transactions`
}

func (lt *LogTransaction) GetByHash(hash []byte) (bool, error) {
	return isFound(GetDB(nil).Where("hash = ?", hash).First(lt))
}
