package service

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"rosetta/models"

	_ "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

type InstanceService struct {
	db *gorm.DB
}

func NewInstanceService(db *gorm.DB) *InstanceService {
	return &InstanceService{db: db}
}

type CreateInstanceRequest struct {
	Name     string `json:"name" binding:"required"`
	Type     string `json:"type" binding:"required"`
	Host     string `json:"host" binding:"required"`
	Port     int    `json:"port" binding:"required"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
}

type UpdateInstanceRequest struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
}

type CreateSchemaRequest struct {
	SchemaName string `json:"schema_name" binding:"required"`
	Layer      string `json:"layer" binding:"required"`
}

func (s *InstanceService) List(page Pagination) ([]models.DatasourceInstance, int64, error) {
	page.Normalize()
	var instances []models.DatasourceInstance
	var total int64
	if err := s.db.Model(&models.DatasourceInstance{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := s.db.Offset(page.Offset()).Limit(page.PageSize).Find(&instances).Error; err != nil {
		return nil, 0, err
	}
	return instances, total, nil
}

func (s *InstanceService) GetByID(id uint64) (*models.DatasourceInstance, error) {
	var inst models.DatasourceInstance
	if err := s.db.First(&inst, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &inst, nil
}

func (s *InstanceService) Create(req CreateInstanceRequest) (*models.DatasourceInstance, error) {
	inst := models.DatasourceInstance{
		Name:       req.Name,
		Type:       req.Type,
		Host:       req.Host,
		Port:       req.Port,
		Credential: fmt.Sprintf(`{"user":"%s","password":"%s","database":"%s"}`, req.User, req.Password, req.Database),
		Status:     "ACTIVE",
	}
	if err := s.db.Create(&inst).Error; err != nil {
		return nil, err
	}
	return &inst, nil
}

func (s *InstanceService) Update(id uint64, req UpdateInstanceRequest) (*models.DatasourceInstance, error) {
	inst, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	if inst == nil {
		return nil, nil
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Type != "" {
		updates["type"] = req.Type
	}
	if req.Host != "" {
		updates["host"] = req.Host
	}
	if req.Port > 0 {
		updates["port"] = req.Port
	}
	if req.User != "" || req.Password != "" {
		updates["credential"] = fmt.Sprintf(`{"user":"%s","password":"%s","database":"%s"}`, req.User, req.Password, req.Database)
	}
	if len(updates) > 0 {
		if err := s.db.Model(&inst).Updates(updates).Error; err != nil {
			return nil, err
		}
	}
	return s.GetByID(id)
}

func (s *InstanceService) Delete(id uint64) error {
	tx := s.db.Begin()
	if err := tx.Where("instance_id = ?", id).Delete(&models.DatasourceSchema{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Delete(&models.DatasourceInstance{}, id).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (s *InstanceService) TestConnection(id uint64) error {
	inst, err := s.GetByID(id)
	if err != nil {
		return err
	}
	if inst == nil {
		return gorm.ErrRecordNotFound
	}

	dsn := s.buildDSN(inst)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("连接失败: %w", err)
	}
	defer db.Close()

	db.SetConnMaxLifetime(5 * time.Second)
	if err := db.Ping(); err != nil {
		return fmt.Errorf("连接测试失败: %w", err)
	}
	return nil
}

func (s *InstanceService) buildDSN(inst *models.DatasourceInstance) string {
	return fmt.Sprintf("root:root@tcp(%s:%d)/?timeout=5s", inst.Host, inst.Port)
}

func (s *InstanceService) ListSchemas(instanceID uint64) ([]models.DatasourceSchema, error) {
	var schemas []models.DatasourceSchema
	if err := s.db.Where("instance_id = ?", instanceID).Find(&schemas).Error; err != nil {
		return nil, err
	}
	return schemas, nil
}

func (s *InstanceService) CreateSchema(instanceID uint64, req CreateSchemaRequest) (*models.DatasourceSchema, error) {
	schema := models.DatasourceSchema{
		InstanceID: instanceID,
		SchemaName: req.SchemaName,
		Layer:      req.Layer,
	}
	if err := s.db.Create(&schema).Error; err != nil {
		return nil, err
	}
	return &schema, nil
}
