package service

import (
	"encoding/json"
	"errors"
	"rosetta/ddl"
	"rosetta/models"
	"time"

	"gorm.io/gorm"
)

type ModelService struct {
	db *gorm.DB
}

func NewModelService(db *gorm.DB) *ModelService {
	return &ModelService{db: db}
}

type CreateModelRequest struct {
	TableName    string `json:"table_name" binding:"required"`
	TableComment string `json:"table_comment"`
}

type UpdateModelRequest struct {
	TableComment string `json:"table_comment"`
	TableStatus  string `json:"table_status"`
}

type UpsertColumnRequest struct {
	Ordinal      int    `json:"ordinal" binding:"required"`
	ColumnName   string `json:"column_name" binding:"required"`
	LogicalType  string `json:"logical_type" binding:"required"`
	TypeLength   *int   `json:"type_length"`
	TypeScale    *int   `json:"type_scale"`
	Nullable     bool   `json:"nullable"`
	DefaultValue string `json:"default_value"`
	Comment      string `json:"comment"`
	IsPrimaryKey bool   `json:"is_primary_key"`
}

type BatchUpdateColumnsRequest struct {
	Columns []UpsertColumnRequest `json:"columns" binding:"required"`
}

type CreateIndexRequest struct {
	IndexName string                `json:"index_name" binding:"required"`
	IndexType string                `json:"index_type"`
	Columns   []ddl.IndexColumnIR   `json:"columns" binding:"required"`
}

type CreateForeignKeyRequest struct {
	FkName        string `json:"fk_name" binding:"required"`
	ColumnName    string `json:"column_name" binding:"required"`
	RefModelID    uint64 `json:"ref_model_id" binding:"required"`
	RefColumnName string `json:"ref_column_name" binding:"required"`
}

type DeployRequest struct {
	SchemaID uint64 `json:"schema_id" binding:"required"`
	Dialect  string `json:"dialect" binding:"required"`
}

type ModelDetailResponse struct {
	*models.LogicalModel
	Columns      []models.ModelColumn     `json:"columns"`
	Indexes      []models.ModelIndex      `json:"indexes"`
	ForeignKeys  []ForeignKeyWithRefName  `json:"foreign_keys"`
}

type ForeignKeyWithRefName struct {
	models.ModelForeignKey
	RefTableName string `json:"ref_table_name"`
}

type ModelListItem struct {
	models.LogicalModel
	ColumnCount int `json:"column_count"`
}

var (
	ErrTableNameExists = errors.New("表名已存在")
)

func (s *ModelService) List(page Pagination, keyword string) ([]ModelListItem, int64, error) {
	page.Normalize()
	var total int64
	query := s.db.Model(&models.LogicalModel{})
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("table_name LIKE ? OR table_comment LIKE ?", like, like)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rawModels []models.LogicalModel
	if err := query.Offset(page.Offset()).Limit(page.PageSize).Order("updated_at DESC").Find(&rawModels).Error; err != nil {
		return nil, 0, err
	}

	items := make([]ModelListItem, len(rawModels))
	for i, m := range rawModels {
		var count int64
		s.db.Model(&models.ModelColumn{}).Where("model_id = ?", m.ID).Count(&count)
		items[i] = ModelListItem{LogicalModel: m, ColumnCount: int(count)}
	}
	return items, total, nil
}

func (s *ModelService) GetDetail(id uint64) (*ModelDetailResponse, error) {
	var model models.LogicalModel
	if err := s.db.First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	var columns []models.ModelColumn
	s.db.Where("model_id = ?", id).Order("ordinal ASC").Find(&columns)

	var indexes []models.ModelIndex
	s.db.Where("model_id = ?", id).Find(&indexes)

	var fks []models.ModelForeignKey
	s.db.Where("model_id = ?", id).Find(&fks)

	fkWithNames := make([]ForeignKeyWithRefName, len(fks))
	refIDs := make([]uint64, 0, len(fks))
	for _, fk := range fks {
		refIDs = append(refIDs, fk.RefModelID)
	}
	refNames := make(map[uint64]string)
	if len(refIDs) > 0 {
		var refModels []models.LogicalModel
		s.db.Where("id IN ?", refIDs).Find(&refModels)
		for _, rm := range refModels {
			refNames[rm.ID] = rm.TabName
		}
	}
	for i, fk := range fks {
		fkWithNames[i] = ForeignKeyWithRefName{
			ModelForeignKey: fk,
			RefTableName:    refNames[fk.RefModelID],
		}
	}

	return &ModelDetailResponse{
		LogicalModel: &model,
		Columns:      columns,
		Indexes:      indexes,
		ForeignKeys:  fkWithNames,
	}, nil
}

func (s *ModelService) Create(req CreateModelRequest, userID uint64) (*models.LogicalModel, error) {
	var exist models.LogicalModel
	if err := s.db.Where("table_name = ?", req.TableName).First(&exist).Error; err == nil {
		return nil, ErrTableNameExists
	}

	model := models.LogicalModel{
		TabName:      req.TableName,
		TableComment: req.TableComment,
		TableStatus:  "DRAFT",
		Source:       "MANUAL",
		CreatedBy:    &userID,
	}
	if err := s.db.Create(&model).Error; err != nil {
		return nil, err
	}
	return &model, nil
}

func (s *ModelService) Update(id uint64, req UpdateModelRequest, userID uint64) (*models.LogicalModel, error) {
	updates := map[string]interface{}{"updated_by": userID}
	if req.TableComment != "" {
		updates["table_comment"] = req.TableComment
	}
	if req.TableStatus != "" {
		updates["table_status"] = req.TableStatus
	}
	if err := s.db.Model(&models.LogicalModel{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}
	var model models.LogicalModel
	s.db.First(&model, id)
	return &model, nil
}

func (s *ModelService) Delete(id uint64) error {
	tx := s.db.Begin()
	tx.Where("model_id = ?", id).Delete(&models.ModelForeignKey{})
	tx.Where("model_id = ?", id).Delete(&models.ModelIndex{})
	tx.Where("model_id = ?", id).Delete(&models.ModelColumn{})
	tx.Where("model_id = ?", id).Delete(&models.ModelDeployment{})
	if err := tx.Delete(&models.LogicalModel{}, id).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (s *ModelService) BatchUpdateColumns(modelID uint64, req BatchUpdateColumnsRequest) ([]models.ModelColumn, error) {
	tx := s.db.Begin()
	if err := tx.Where("model_id = ?", modelID).Delete(&models.ModelColumn{}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	for _, col := range req.Columns {
		mc := models.ModelColumn{
			ModelID:      modelID,
			Ordinal:      col.Ordinal,
			ColumnName:   col.ColumnName,
			LogicalType:  col.LogicalType,
			TypeLength:   col.TypeLength,
			TypeScale:    col.TypeScale,
			Nullable:     col.Nullable,
			DefaultValue: col.DefaultValue,
			Comment:      col.Comment,
			IsPrimaryKey: col.IsPrimaryKey,
		}
		if err := tx.Create(&mc).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}
	tx.Commit()

	var columns []models.ModelColumn
	s.db.Where("model_id = ?", modelID).Order("ordinal ASC").Find(&columns)
	return columns, nil
}

func (s *ModelService) CreateIndex(modelID uint64, req CreateIndexRequest) (*models.ModelIndex, error) {
	indexType := req.IndexType
	if indexType == "" {
		indexType = "NORMAL"
	}
	colsJSON, _ := json.Marshal(req.Columns)
	idx := models.ModelIndex{
		ModelID:   modelID,
		IndexName: req.IndexName,
		IndexType: indexType,
		Columns:   string(colsJSON),
	}
	if err := s.db.Create(&idx).Error; err != nil {
		return nil, err
	}
	return &idx, nil
}

func (s *ModelService) UpdateIndex(modelID, indexID uint64, req CreateIndexRequest) (*models.ModelIndex, error) {
	colsJSON, _ := json.Marshal(req.Columns)
	updates := map[string]interface{}{
		"index_name": req.IndexName,
		"index_type": req.IndexType,
		"columns":    string(colsJSON),
	}
	if err := s.db.Model(&models.ModelIndex{}).Where("id = ? AND model_id = ?", indexID, modelID).Updates(updates).Error; err != nil {
		return nil, err
	}
	var idx models.ModelIndex
	s.db.First(&idx, indexID)
	return &idx, nil
}

func (s *ModelService) DeleteIndex(modelID, indexID uint64) error {
	result := s.db.Where("id = ? AND model_id = ?", indexID, modelID).Delete(&models.ModelIndex{})
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

func (s *ModelService) CreateForeignKey(modelID uint64, req CreateForeignKeyRequest) (*models.ModelForeignKey, error) {
	fk := models.ModelForeignKey{
		ModelID:       modelID,
		FkName:        req.FkName,
		ColumnName:    req.ColumnName,
		RefModelID:    req.RefModelID,
		RefColumnName: req.RefColumnName,
	}
	if err := s.db.Create(&fk).Error; err != nil {
		return nil, err
	}
	return &fk, nil
}

func (s *ModelService) DeleteForeignKey(modelID, fkID uint64) error {
	result := s.db.Where("id = ? AND model_id = ?", fkID, modelID).Delete(&models.ModelForeignKey{})
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

func (s *ModelService) RenderDDL(modelID uint64, dialect string) (string, error) {
	detail, err := s.GetDetail(modelID)
	if err != nil {
		return "", err
	}
	if detail == nil {
		return "", gorm.ErrRecordNotFound
	}

	refNames := make(map[uint64]string)
	for _, fk := range detail.ForeignKeys {
		refNames[fk.RefModelID] = fk.RefTableName
	}

	var indexes []models.ModelIndex
	s.db.Where("model_id = ?", modelID).Find(&indexes)

	var fks []models.ModelForeignKey
	s.db.Where("model_id = ?", modelID).Find(&fks)

	ir := ddl.BuildIR(detail.LogicalModel, detail.Columns, indexes, fks, refNames)

	var renderer ddl.Renderer
	switch dialect {
	case "GAUSSDB", "GAUSSDB_M":
		renderer = &ddl.GaussDBRenderer{}
	default:
		renderer = &ddl.MySQLRenderer{}
	}
	return renderer.Render(ir), nil
}

func (s *ModelService) Deploy(modelID uint64, req DeployRequest, userID uint64) (*models.ModelDeployment, error) {
	ddlText, err := s.RenderDDL(modelID, req.Dialect)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	deploy := models.ModelDeployment{
		ModelID:     modelID,
		SchemaID:    req.SchemaID,
		Dialect:     req.Dialect,
		DeployedDDL: ddlText,
		DeployedAt:  &now,
		DeployedBy:  &userID,
	}
	if err := s.db.Where("model_id = ? AND schema_id = ?", modelID, req.SchemaID).Delete(&models.ModelDeployment{}).Error; err != nil {
		return nil, err
	}
	if err := s.db.Create(&deploy).Error; err != nil {
		return nil, err
	}
	return &deploy, nil
}

func (s *ModelService) ListDeployments(modelID uint64) ([]models.ModelDeployment, error) {
	var deps []models.ModelDeployment
	if err := s.db.Where("model_id = ?", modelID).Order("deployed_at DESC").Find(&deps).Error; err != nil {
		return nil, err
	}
	return deps, nil
}
