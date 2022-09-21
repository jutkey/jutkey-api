package sql

import (
	"github.com/IBAX-io/go-ibax/packages/converter"
)

// Member represents a ecosystem member
type Member struct {
	Ecosystem  int64
	ID         int64  `gorm:"primary_key;not null"`
	MemberName string `gorm:"not null"`
	ImageID    *int64
	MemberInfo string `gorm:"type:jsonb(PostgreSQL)"`
}

// SetTablePrefix is setting table prefix
func (m *Member) SetTablePrefix(prefix string) {
	m.Ecosystem = converter.StrToInt64(prefix)
}

// TableName returns name of table
func (m *Member) TableName() string {
	if m.Ecosystem == 0 {
		m.Ecosystem = 1
	}
	return `1_members`
}

// Count returns count of records in table
func (m *Member) Count() (count int64, err error) {
	err = GetDB(nil).Table(m.TableName()).Where(`ecosystem=?`, m.Ecosystem).Count(&count).Error
	return
}

// Get init m as member with ID
func (m *Member) Get(account string) (bool, error) {
	return isFound(GetDB(nil).Where("ecosystem=? and account = ?", m.Ecosystem, account).First(m))
}

func (m *Member) GetAccount(eid int64, account string) (bool, error) {
	return isFound(GetDB(nil).Where("ecosystem=? and account = ?", eid, account).First(m))
}
