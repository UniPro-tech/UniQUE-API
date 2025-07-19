package scheme

type User struct {
	ID            string `gorm:"primaryKey;column:id;type:varchar(255)"`
	Email         string `gorm:"column:email;type:varchar(255)"`
	CustomID      string `gorm:"column:custom_id;type:varchar(50)"`
	Name          string `gorm:"column:name;type:varchar(50)"`
	ExternalEmail string `gorm:"column:external_email;type:varchar(255)"`
	Period        string `gorm:"column:period;type:varchar(50)"`
	IsEnable      bool   `gorm:"column:is_enable;type:boolean"`
}
