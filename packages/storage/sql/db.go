package sql

import (
	"errors"
	"gorm.io/gorm"
	"jutkey-server/conf"
)

type DbTransaction struct {
	conn *gorm.DB
}

// GormClose is closing Gorm connection
func GormClose() error {
	if err := conf.GetEnvConf().DatabaseInfo.Close(); err != nil {
		return err
	}
	return nil
}

// GetDB is returning gorm.DB
func GetDB(db *DbTransaction) *gorm.DB {
	if db != nil && db.conn != nil {
		return db.conn
	}
	return conf.GetDbConn().Conn()
}

func isFound(db *gorm.DB) (bool, error) {
	if errors.Is(db.Error, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return true, db.Error
}

func HasTableOrView(tr *DbTransaction, names string) bool {
	var name string
	conf.GetDbConn().Conn().Table("information_schema.tables").
		Where("table_type IN ('BASE TABLE', 'VIEW') AND table_schema NOT IN ('pg_catalog', 'information_schema') AND table_name=?", names).
		Select("table_name").Row().Scan(&name)

	return name == names
}

//HasTable p is struct Pointer
func HasTable(p any) bool {
	if !GetDB(nil).Migrator().HasTable(p) {
		return false
	}
	return true
}
