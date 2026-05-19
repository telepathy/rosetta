package ddl

import (
	"encoding/json"

	"rosetta/models"
)

type TableIR struct {
	TableName    string
	TableComment string
	Columns      []ColumnIR
	Indexes      []IndexIR
	ForeignKeys  []ForeignKeyIR
}

type ColumnIR struct {
	Name         string
	LogicalType  string
	TypeLength   *int
	TypeScale    *int
	Nullable     bool
	DefaultValue string
	Comment      string
	IsPrimaryKey bool
}

type IndexIR struct {
	Name    string
	Type    string
	Columns []IndexColumnIR
}

type IndexColumnIR struct {
	Name  string `json:"name"`
	Order string `json:"order"`
}

type ForeignKeyIR struct {
	Name         string
	ColumnName   string
	RefTableName string
	RefColumn    string
}

type Renderer interface {
	Render(ir *TableIR) string
}

func BuildIR(model *models.LogicalModel, columns []models.ModelColumn, indexes []models.ModelIndex, fks []models.ModelForeignKey, refTableNames map[uint64]string) *TableIR {
	ir := &TableIR{
		TableName:    model.TabName,
		TableComment: model.TableComment,
	}

	for _, col := range columns {
		ir.Columns = append(ir.Columns, ColumnIR{
			Name:         col.ColumnName,
			LogicalType:  col.LogicalType,
			TypeLength:   col.TypeLength,
			TypeScale:    col.TypeScale,
			Nullable:     col.Nullable,
			DefaultValue: col.DefaultValue,
			Comment:      col.Comment,
			IsPrimaryKey: col.IsPrimaryKey,
		})
	}

	for _, idx := range indexes {
		var cols []IndexColumnIR
		_ = json.Unmarshal([]byte(idx.Columns), &cols)
		ir.Indexes = append(ir.Indexes, IndexIR{
			Name:    idx.IndexName,
			Type:    idx.IndexType,
			Columns: cols,
		})
	}

	for _, fk := range fks {
		refName := refTableNames[fk.RefModelID]
		if refName == "" {
			refName = "unknown"
		}
		ir.ForeignKeys = append(ir.ForeignKeys, ForeignKeyIR{
			Name:         fk.FkName,
			ColumnName:   fk.ColumnName,
			RefTableName: refName,
			RefColumn:    fk.RefColumnName,
		})
	}

	return ir
}
