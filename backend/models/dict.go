package models

import "time"

type DictDefinition struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	DictName  string    `gorm:"size:256;not null" json:"dict_name"`
	DictCode  string    `gorm:"size:128;not null;uniqueIndex" json:"dict_code"`
	DictType  string    `gorm:"size:32;not null;default:STANDARD" json:"dict_type"`
	Remark    string    `gorm:"size:512" json:"remark"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (DictDefinition) TableName() string {
	return "dict_definition"
}

type DictItem struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	DictID    uint64    `gorm:"not null;uniqueIndex:uk_dict_item" json:"dict_id"`
	ItemCode  string    `gorm:"size:128;not null;uniqueIndex:uk_dict_item" json:"item_code"`
	ItemName  string    `gorm:"size:256;not null" json:"item_name"`
	ItemValue string    `gorm:"size:512" json:"item_value"`
	Ordinal   int       `gorm:"not null;default:0" json:"ordinal"`
	CreatedAt time.Time `json:"created_at"`
}

func (DictItem) TableName() string {
	return "dict_item"
}
