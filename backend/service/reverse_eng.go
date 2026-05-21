package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"rosetta/models"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

type ReverseEngService struct {
	db *gorm.DB
}

func NewReverseEngService(db *gorm.DB) *ReverseEngService {
	return &ReverseEngService{db: db}
}

type RevEngTableInfo struct {
	TableName    string `json:"table_name"`
	TableComment string `json:"table_comment"`
	ColumnCount  int    `json:"column_count"`
}

type RevEngColumnInfo struct {
	Ordinal      int    `json:"ordinal"`
	ColumnName   string `json:"column_name"`
	DataType     string `json:"data_type"`
	CharMaxLen   *int   `json:"char_max_len"`
	NumPrecision *int   `json:"num_precision"`
	NumScale     *int   `json:"num_scale"`
	Nullable     string `json:"nullable"`
	ColumnKey    string `json:"column_key"`
	ColumnDef    string `json:"column_default"`
	Comment      string `json:"comment"`
}

type RevEngPreviewResponse struct {
	TableName    string             `json:"table_name"`
	TableComment string             `json:"table_comment"`
	Columns      []RevEngColumnInfo `json:"columns"`
}

func (s *ReverseEngService) ListTables(instanceID uint64, schemaName string) ([]RevEngTableInfo, error) {
	var inst models.DatasourceInstance
	if err := s.db.First(&inst, instanceID).Error; err != nil {
		return nil, fmt.Errorf("实例不存在")
	}

	dsn := buildDSN(&inst, schemaName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("连接失败: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(
		"SELECT TABLE_NAME, TABLE_COMMENT FROM information_schema.TABLES WHERE TABLE_SCHEMA = ? ORDER BY TABLE_NAME",
		schemaName,
	)
	if err != nil {
		return nil, fmt.Errorf("查询表列表失败: %w", err)
	}
	defer rows.Close()

	var tables []RevEngTableInfo
	for rows.Next() {
		var t RevEngTableInfo
		if err := rows.Scan(&t.TableName, &t.TableComment); err != nil {
			continue
		}
		tables = append(tables, t)
	}

	for i := range tables {
		var count int
		_ = db.QueryRow(
			"SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?",
			schemaName, tables[i].TableName,
		).Scan(&count)
		tables[i].ColumnCount = count
	}

	return tables, nil
}

func (s *ReverseEngService) Preview(instanceID uint64, schemaName, tableName string) (*RevEngPreviewResponse, error) {
	var inst models.DatasourceInstance
	if err := s.db.First(&inst, instanceID).Error; err != nil {
		return nil, fmt.Errorf("实例不存在")
	}

	dsn := buildDSN(&inst, schemaName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("连接失败: %w", err)
	}
	defer db.Close()

	var tableComment string
	_ = db.QueryRow(
		"SELECT TABLE_COMMENT FROM information_schema.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?",
		schemaName, tableName,
	).Scan(&tableComment)

	rows, err := db.Query(
		"SELECT ORDINAL_POSITION, COLUMN_NAME, DATA_TYPE, CHARACTER_MAXIMUM_LENGTH, NUMERIC_PRECISION, NUMERIC_SCALE, IS_NULLABLE, COLUMN_KEY, COLUMN_DEFAULT, COLUMN_COMMENT FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? ORDER BY ORDINAL_POSITION",
		schemaName, tableName,
	)
	if err != nil {
		return nil, fmt.Errorf("查询列信息失败: %w", err)
	}
	defer rows.Close()

	var columns []RevEngColumnInfo
	for rows.Next() {
		var c RevEngColumnInfo
		var charMaxLen, numPrec, numScale sql.NullInt64
		var colDef sql.NullString
		if err := rows.Scan(&c.Ordinal, &c.ColumnName, &c.DataType, &charMaxLen, &numPrec, &numScale, &c.Nullable, &c.ColumnKey, &colDef, &c.Comment); err != nil {
			continue
		}
		if charMaxLen.Valid {
			v := int(charMaxLen.Int64)
			c.CharMaxLen = &v
		}
		if numPrec.Valid {
			v := int(numPrec.Int64)
			c.NumPrecision = &v
		}
		if numScale.Valid {
			v := int(numScale.Int64)
			c.NumScale = &v
		}
		if colDef.Valid {
			c.ColumnDef = colDef.String
		}
		c.DataType = strings.ToUpper(c.DataType)
		c.Nullable = strings.ToUpper(c.Nullable)
		columns = append(columns, c)
	}

	return &RevEngPreviewResponse{
		TableName:    tableName,
		TableComment: tableComment,
		Columns:      columns,
	}, nil
}

type ImportRevEngRequest struct {
	InstanceID uint64 `json:"instance_id" binding:"required"`
	SchemaName string `json:"schema_name" binding:"required"`
	TableName  string `json:"table_name" binding:"required"`
	DatabaseID uint64 `json:"database_id" binding:"required"`
	LogicalSchemaID uint64 `json:"logical_schema_id" binding:"required"`
	CreatedBy  uint64 `json:"-"`
}

func (s *ReverseEngService) Import(req ImportRevEngRequest) (*models.LogicalModel, error) {
	preview, err := s.Preview(req.InstanceID, req.SchemaName, req.TableName)
	if err != nil {
		return nil, err
	}

	var exist models.LogicalModel
	if err := s.db.Where("schema_id = ? AND table_name = ?", req.LogicalSchemaID, req.TableName).First(&exist).Error; err == nil {
		return nil, fmt.Errorf("表名 %s 已存在", req.TableName)
	}

	tx := s.db.Begin()

	model := models.LogicalModel{
		DatabaseID:   req.DatabaseID,
		SchemaID:     req.LogicalSchemaID,
		TabName:      req.TableName,
		TableComment: preview.TableComment,
		TableStatus:  "DRAFT",
		Source:       "REVERSE_ENGINEERED",
		CreatedBy:    &req.CreatedBy,
	}
	if err := tx.Create(&model).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	for _, col := range preview.Columns {
		logicalType, typeLen, typeScale := mapMySQLType(col.DataType, col.CharMaxLen, col.NumPrecision, col.NumScale)
		mc := models.ModelColumn{
			ModelID:      model.ID,
			Ordinal:      col.Ordinal,
			ColumnName:   col.ColumnName,
			LogicalType:  logicalType,
			TypeLength:   typeLen,
			TypeScale:    typeScale,
			Nullable:     col.Nullable == "YES",
			DefaultValue: col.ColumnDef,
			Comment:      col.Comment,
			IsPrimaryKey: col.ColumnKey == "PRI",
		}
		if err := tx.Create(&mc).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	tx.Commit()
	return &model, nil
}

func mapMySQLType(dataType string, charMaxLen, numPrec, numScale *int) (string, *int, *int) {
	dt := strings.ToUpper(dataType)
	switch {
	case dt == "VARCHAR" || dt == "CHAR":
		return "VARCHAR", charMaxLen, nil
	case dt == "TEXT" || dt == "MEDIUMTEXT" || dt == "LONGTEXT":
		return "TEXT", nil, nil
	case dt == "BIGINT":
		return "BIGINT", nil, nil
	case dt == "INT" || dt == "INTEGER" || dt == "TINYINT" || dt == "SMALLINT" || dt == "MEDIUMINT":
		return "INT", nil, nil
	case dt == "DECIMAL" || dt == "NUMERIC":
		return "DECIMAL", numPrec, numScale
	case dt == "FLOAT":
		return "FLOAT", nil, nil
	case dt == "DOUBLE":
		return "DOUBLE", nil, nil
	case dt == "DATE":
		return "DATE", nil, nil
	case dt == "DATETIME":
		return "DATETIME", nil, nil
	case dt == "TIMESTAMP":
		return "TIMESTAMP", nil, nil
	case dt == "BOOLEAN" || dt == "TINYINT":
		return "BOOLEAN", nil, nil
	case dt == "JSON":
		return "JSON", nil, nil
	case dt == "CLOB":
		return "TEXT", nil, nil
	default:
		return "VARCHAR", charMaxLen, nil
	}
}

func buildDSN(inst *models.DatasourceInstance, database string) string {
	var user, password, dbName string
	var cred map[string]string
	if err := json.Unmarshal([]byte(inst.Credential), &cred); err == nil {
		user = cred["user"]
		password = cred["password"]
		dbName = cred["database"]
	}
	if database != "" {
		dbName = database
	}
	if user == "" {
		user = "root"
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?timeout=10s&charset=utf8mb4&parseTime=true",
		user, password, inst.Host, inst.Port, dbName)
}
