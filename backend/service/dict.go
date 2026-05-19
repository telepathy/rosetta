package service

import (
	"errors"

	"rosetta/models"

	"gorm.io/gorm"
)

type DictService struct {
	db *gorm.DB
}

func NewDictService(db *gorm.DB) *DictService {
	return &DictService{db: db}
}

type CreateDictRequest struct {
	DictName string `json:"dict_name" binding:"required"`
	DictCode string `json:"dict_code" binding:"required"`
	DictType string `json:"dict_type"`
	Remark   string `json:"remark"`
}

type UpdateDictRequest struct {
	DictName string `json:"dict_name"`
	DictType string `json:"dict_type"`
	Remark   string `json:"remark"`
}

type CreateDictItemRequest struct {
	ItemCode  string `json:"item_code" binding:"required"`
	ItemName  string `json:"item_name" binding:"required"`
	ItemValue string `json:"item_value"`
	Ordinal   int    `json:"ordinal"`
}

type UpdateDictItemRequest struct {
	ItemCode  string `json:"item_code"`
	ItemName  string `json:"item_name"`
	ItemValue string `json:"item_value"`
	Ordinal   int    `json:"ordinal"`
}

type UpdateDictItemOrderRequest struct {
	Items []DictItemOrderEntry `json:"items" binding:"required"`
}

type DictItemOrderEntry struct {
	ID      uint64 `json:"id"`
	Ordinal int    `json:"ordinal"`
}

var (
	ErrDictCodeExists = errors.New("字典编码已存在")
	ErrItemCodeExists = errors.New("字典条目编码已存在")
)

func (s *DictService) ListDefinitions(page Pagination) ([]models.DictDefinition, int64, error) {
	page.Normalize()
	var dicts []models.DictDefinition
	var total int64
	if err := s.db.Model(&models.DictDefinition{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := s.db.Offset(page.Offset()).Limit(page.PageSize).Order("created_at DESC").Find(&dicts).Error; err != nil {
		return nil, 0, err
	}
	return dicts, total, nil
}

func (s *DictService) GetDefinition(id uint64) (*models.DictDefinition, error) {
	var dict models.DictDefinition
	if err := s.db.First(&dict, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &dict, nil
}

func (s *DictService) CreateDefinition(req CreateDictRequest) (*models.DictDefinition, error) {
	var exist models.DictDefinition
	if err := s.db.Where("dict_code = ?", req.DictCode).First(&exist).Error; err == nil {
		return nil, ErrDictCodeExists
	}

	dictType := req.DictType
	if dictType == "" {
		dictType = "STANDARD"
	}

	dict := models.DictDefinition{
		DictName: req.DictName,
		DictCode: req.DictCode,
		DictType: dictType,
		Remark:   req.Remark,
	}
	if err := s.db.Create(&dict).Error; err != nil {
		return nil, err
	}
	return &dict, nil
}

func (s *DictService) UpdateDefinition(id uint64, req UpdateDictRequest) (*models.DictDefinition, error) {
	updates := map[string]interface{}{}
	if req.DictName != "" {
		updates["dict_name"] = req.DictName
	}
	if req.DictType != "" {
		updates["dict_type"] = req.DictType
	}
	if req.Remark != "" {
		updates["remark"] = req.Remark
	}
	if len(updates) > 0 {
		if err := s.db.Model(&models.DictDefinition{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return nil, err
		}
	}
	return s.GetDefinition(id)
}

func (s *DictService) DeleteDefinition(id uint64) error {
	tx := s.db.Begin()
	if err := tx.Where("dict_id = ?", id).Delete(&models.DictItem{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Delete(&models.DictDefinition{}, id).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (s *DictService) ListItems(dictID uint64) ([]models.DictItem, error) {
	var items []models.DictItem
	if err := s.db.Where("dict_id = ?", dictID).Order("ordinal ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *DictService) CreateItem(dictID uint64, req CreateDictItemRequest) (*models.DictItem, error) {
	var exist models.DictItem
	if err := s.db.Where("dict_id = ? AND item_code = ?", dictID, req.ItemCode).First(&exist).Error; err == nil {
		return nil, ErrItemCodeExists
	}

	item := models.DictItem{
		DictID:    dictID,
		ItemCode:  req.ItemCode,
		ItemName:  req.ItemName,
		ItemValue: req.ItemValue,
		Ordinal:   req.Ordinal,
	}
	if err := s.db.Create(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *DictService) UpdateItem(dictID, itemID uint64, req UpdateDictItemRequest) (*models.DictItem, error) {
	updates := map[string]interface{}{}
	if req.ItemCode != "" {
		updates["item_code"] = req.ItemCode
	}
	if req.ItemName != "" {
		updates["item_name"] = req.ItemName
	}
	updates["item_value"] = req.ItemValue
	updates["ordinal"] = req.Ordinal

	if err := s.db.Model(&models.DictItem{}).Where("id = ? AND dict_id = ?", itemID, dictID).Updates(updates).Error; err != nil {
		return nil, err
	}

	var item models.DictItem
	if err := s.db.First(&item, itemID).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *DictService) DeleteItem(dictID, itemID uint64) error {
	result := s.db.Where("id = ? AND dict_id = ?", itemID, dictID).Delete(&models.DictItem{})
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

func (s *DictService) UpdateItemsOrder(dictID uint64, req UpdateDictItemOrderRequest) error {
	tx := s.db.Begin()
	for _, entry := range req.Items {
		if err := tx.Model(&models.DictItem{}).Where("id = ? AND dict_id = ?", entry.ID, dictID).Update("ordinal", entry.Ordinal).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}
