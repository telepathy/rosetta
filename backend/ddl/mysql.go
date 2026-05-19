package ddl

import (
	"fmt"
	"strings"
)

type MySQLRenderer struct{}

func (r *MySQLRenderer) Render(ir *TableIR) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("CREATE TABLE `%s` (\n", ir.TableName))

	for i, col := range ir.Columns {
		sb.WriteString("  ")
		sb.WriteString(fmt.Sprintf("`%s`", col.Name))
		sb.WriteString(" ")
		sb.WriteString(r.renderType(col))
		if !col.Nullable {
			sb.WriteString(" NOT NULL")
		}
		if col.DefaultValue != "" {
			sb.WriteString(fmt.Sprintf(" DEFAULT '%s'", col.DefaultValue))
		}
		if col.Comment != "" {
			sb.WriteString(fmt.Sprintf(" COMMENT '%s'", col.Comment))
		}
		if i < len(ir.Columns)-1 || len(ir.primaryKeys()) > 0 || len(ir.Indexes) > 0 || len(ir.ForeignKeys) > 0 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}

	pks := ir.primaryKeys()
	if len(pks) > 0 {
		sb.WriteString("  PRIMARY KEY (")
		for i, pk := range pks {
			sb.WriteString(fmt.Sprintf("`%s`", pk))
			if i < len(pks)-1 {
				sb.WriteString(", ")
			}
		}
		sb.WriteString(")")
		if len(ir.Indexes) > 0 || len(ir.ForeignKeys) > 0 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}

	for i, idx := range ir.Indexes {
		idxType := "INDEX"
		if idx.Type == "UNIQUE" {
			idxType = "UNIQUE INDEX"
		}
		sb.WriteString(fmt.Sprintf("  %s `%s` (", idxType, idx.Name))
		for j, col := range idx.Columns {
			sb.WriteString(fmt.Sprintf("`%s`", col.Name))
			if col.Order == "DESC" {
				sb.WriteString(" DESC")
			}
			if j < len(idx.Columns)-1 {
				sb.WriteString(", ")
			}
		}
		sb.WriteString(")")
		if i < len(ir.Indexes)-1 || len(ir.ForeignKeys) > 0 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}

	for i, fk := range ir.ForeignKeys {
		sb.WriteString(fmt.Sprintf("  CONSTRAINT `%s` FOREIGN KEY (`%s`) REFERENCES `%s` (`%s`)",
			fk.Name, fk.ColumnName, fk.RefTableName, fk.RefColumn))
		if i < len(ir.ForeignKeys)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}

	sb.WriteString(") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4")

	if ir.TableComment != "" {
		sb.WriteString(fmt.Sprintf(" COMMENT='%s'", ir.TableComment))
	}
	sb.WriteString(";")
	return sb.String()
}

func (r *MySQLRenderer) renderType(col ColumnIR) string {
	lt := strings.ToUpper(col.LogicalType)
	switch lt {
	case "VARCHAR":
		length := 255
		if col.TypeLength != nil {
			length = *col.TypeLength
		}
		return fmt.Sprintf("VARCHAR(%d)", length)
	case "DECIMAL":
		length := 18
		scale := 2
		if col.TypeLength != nil {
			length = *col.TypeLength
		}
		if col.TypeScale != nil {
			scale = *col.TypeScale
		}
		return fmt.Sprintf("DECIMAL(%d,%d)", length, scale)
	case "BIGINT":
		if col.IsPrimaryKey {
			return "BIGINT AUTO_INCREMENT"
		}
		return "BIGINT"
	case "INT":
		return "INT"
	case "FLOAT":
		return "FLOAT"
	case "DOUBLE":
		return "DOUBLE"
	case "DATE":
		return "DATE"
	case "DATETIME":
		return "DATETIME"
	case "TIMESTAMP":
		return "TIMESTAMP"
	case "TEXT":
		return "TEXT"
	case "BOOLEAN":
		return "TINYINT(1)"
	case "JSON":
		return "JSON"
	default:
		return lt
	}
}

func (t *TableIR) primaryKeys() []string {
	var pks []string
	for _, col := range t.Columns {
		if col.IsPrimaryKey {
			pks = append(pks, col.Name)
		}
	}
	return pks
}
