package service

import (
	"errors"

	"rosetta/models"

	"gorm.io/gorm"
)

type DatabaseService struct {
	db *gorm.DB
}

func NewDatabaseService(db *gorm.DB) *DatabaseService {
	return &DatabaseService{db: db}
}

type CreateDatabaseRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type CreateLogicalSchemaRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type DatabaseDetailResponse struct {
	*models.LogicalDatabase
	Schemas []models.LogicalSchema `json:"schemas"`
}

func (s *DatabaseService) ListDatabases() ([]models.LogicalDatabase, error) {
	var dbs []models.LogicalDatabase
	if err := s.db.Order("name ASC").Find(&dbs).Error; err != nil {
		return nil, err
	}
	return dbs, nil
}

func (s *DatabaseService) GetDatabase(id uint64) (*models.LogicalDatabase, error) {
	var db models.LogicalDatabase
	if err := s.db.First(&db, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &db, nil
}

func (s *DatabaseService) CreateDatabase(req CreateDatabaseRequest) (*models.LogicalDatabase, error) {
	db := models.LogicalDatabase{
		Name:        req.Name,
		Description: req.Description,
	}
	if err := s.db.Create(&db).Error; err != nil {
		return nil, err
	}
	return &db, nil
}

func (s *DatabaseService) UpdateDatabase(id uint64, req CreateDatabaseRequest) (*models.LogicalDatabase, error) {
	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	updates["description"] = req.Description
	if len(updates) > 0 {
		if err := s.db.Model(&models.LogicalDatabase{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return nil, err
		}
	}
	return s.GetDatabase(id)
}

func (s *DatabaseService) DeleteDatabase(id uint64) error {
	tx := s.db.Begin()
	tx.Where("database_id = ?", id).Delete(&models.LogicalSchema{})
	tx.Where("database_id = ?", id).Delete(&models.DatabaseInstanceMapping{})
	tx.Where("database_id = ?", id).Delete(&models.LogicalModel{})
	if err := tx.Delete(&models.LogicalDatabase{}, id).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (s *DatabaseService) ListSchemas(databaseID uint64) ([]models.LogicalSchema, error) {
	var schemas []models.LogicalSchema
	if err := s.db.Where("database_id = ?", databaseID).Order("name ASC").Find(&schemas).Error; err != nil {
		return nil, err
	}
	return schemas, nil
}

func (s *DatabaseService) GetSchema(id uint64) (*models.LogicalSchema, error) {
	var schema models.LogicalSchema
	if err := s.db.First(&schema, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &schema, nil
}

func (s *DatabaseService) CreateSchema(databaseID uint64, req CreateLogicalSchemaRequest) (*models.LogicalSchema, error) {
	schema := models.LogicalSchema{
		DatabaseID:  databaseID,
		Name:        req.Name,
		Description: req.Description,
	}
	if err := s.db.Create(&schema).Error; err != nil {
		return nil, err
	}
	return &schema, nil
}

func (s *DatabaseService) DeleteSchema(id uint64) error {
	result := s.db.Delete(&models.LogicalSchema{}, id)
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

func (s *DatabaseService) ListMappedInstances(databaseID uint64) ([]models.DatasourceInstance, error) {
	var instances []models.DatasourceInstance
	if err := s.db.Joins("JOIN database_instance_mapping ON database_instance_mapping.instance_id = datasource_instance.id").
		Where("database_instance_mapping.database_id = ?", databaseID).
		Find(&instances).Error; err != nil {
		return nil, err
	}
	return instances, nil
}

func (s *DatabaseService) MapInstance(databaseID, instanceID uint64) error {
	mapping := models.DatabaseInstanceMapping{
		DatabaseID: databaseID,
		InstanceID: instanceID,
	}
	return s.db.Create(&mapping).Error
}

func (s *DatabaseService) UnmapInstance(databaseID, instanceID uint64) error {
	result := s.db.Where("database_id = ? AND instance_id = ?", databaseID, instanceID).
		Delete(&models.DatabaseInstanceMapping{})
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

func (s *DatabaseService) GetAllDatabasesWithSchemas() ([]DatabaseDetailResponse, error) {
	dbs, err := s.ListDatabases()
	if err != nil {
		return nil, err
	}
	result := make([]DatabaseDetailResponse, len(dbs))
	for i, db := range dbs {
		schemas, _ := s.ListSchemas(db.ID)
		result[i] = DatabaseDetailResponse{
			LogicalDatabase: &dbs[i],
			Schemas:         schemas,
		}
	}
	return result, nil
}
