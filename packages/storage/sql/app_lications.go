package sql

type Applications struct {
	ID         int64  `gorm:"primary_key;not null"`
	Name       string `gorm:"column:name;type:varchar(255);not null"`
	Conditions string `gorm:"column:conditions;not null"`
	Deleted    int64  `gorm:"column:deleted;not null"` //1:deleted
	Ecosystem  int64  `gorm:"column:ecosystem;not null"`
}

func (p Applications) TableName() string {
	return "1_applications"
}

//func (p *Applications) FindApp(page, limit int, order, where string) (*[]Applications, int64, error) {
//	var rets []Applications
//	var total int64
//	if err := GetDB(nil).Table(p.TableName()).Where(where).Count(&total).Error; err != nil {
//		return nil, total, err
//	}
//	if err := GetDB(nil).Offset((page - 1) * limit).Limit(limit).Where(where).Order(order).Find(&rets).Error; err != nil {
//		return nil, total, err
//	}
//	return &rets, total, nil
//}
//
//func (p *Applications) GetById(appid int64) (bool, error) {
//	return isFound(GetDB(nil).Where("id = ?", appid).Last(p))
//}

func (p *Applications) GetByName(name string, ecosystem int64) (bool, error) {
	return isFound(GetDB(nil).Where("name = ? AND ecosystem = ?", name, ecosystem).First(p))
}
