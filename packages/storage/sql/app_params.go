package sql

// AppParam is model
type AppParam struct {
	ecosystem  int64
	ID         int64  `gorm:"primary_key;not null"`
	AppID      int64  `gorm:"not null"`
	Name       string `gorm:"not null;size:100"`
	Value      string `gorm:"not null"`
	Conditions string `gorm:"not null"`
}

// TableName returns name of table
func (sp *AppParam) TableName() string {
	if sp.ecosystem == 0 {
		sp.ecosystem = 1
	}
	return `1_app_params`
}

//// GetById is retrieving model from database
func (sp *AppParam) GetById(transaction *DbTransaction, id int64) (bool, error) {
	return isFound(GetDB(transaction).Where("id=?",
		id).First(sp))
}

func getAppValue(appId int64, name string, ecosystem int64) (string, error) {
	var sp AppParam
	_, err := isFound(GetDB(nil).Select("value").Where("app_id = ? AND name = ? AND ecosystem = ?", appId, name, ecosystem).First(&sp))
	if err != nil {
		return "", err
	}
	return sp.Value, nil
}
