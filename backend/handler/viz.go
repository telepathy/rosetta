package handler

import (
	"strconv"

	"rosetta/database"
	"rosetta/middleware"
	"rosetta/service"
	"rosetta/utils"

	"github.com/gin-gonic/gin"
)

type VizHandler struct {
	modelSvc *service.ModelService
}

func NewVizHandler(modelSvc *service.ModelService) *VizHandler {
	return &VizHandler{modelSvc: modelSvc}
}

func (h *VizHandler) StructureDiagram(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的模型ID")
		return
	}

	detail, err := h.modelSvc.GetDetail(id)
	if err != nil || detail == nil {
		utils.BadRequest(c, "模型不存在")
		return
	}

	type columnNode struct {
		Name         string `json:"name"`
		LogicalType  string `json:"logical_type"`
		IsPrimaryKey bool   `json:"is_primary_key"`
		IsForeignKey bool   `json:"is_foreign_key"`
		Nullable     bool   `json:"nullable"`
		Comment      string `json:"comment"`
	}

	fkCols := make(map[string]bool)
	for _, fk := range detail.ForeignKeys {
		fkCols[fk.ColumnName] = true
	}

	nodes := make([]columnNode, len(detail.Columns))
	for i, col := range detail.Columns {
		nodes[i] = columnNode{
			Name:         col.ColumnName,
			LogicalType:  col.LogicalType,
			IsPrimaryKey: col.IsPrimaryKey,
			IsForeignKey: fkCols[col.ColumnName],
			Nullable:     col.Nullable,
			Comment:      col.Comment,
		}
	}

	utils.Success(c, gin.H{
		"table_name":    detail.TabName,
		"table_comment": detail.TableComment,
		"columns":       nodes,
		"indexes":       detail.Indexes,
		"foreign_keys":  detail.ForeignKeys,
	})
}

func (h *VizHandler) ERDiagram(c *gin.Context) {
	schemaID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的Schema ID")
		return
	}

	type erTable struct {
		ID           uint64 `json:"id"`
		TableName    string `json:"table_name"`
		TableComment string `json:"table_comment"`
		ColumnCount  int    `json:"column_count"`
	}

	type erEdge struct {
		Source       uint64 `json:"source"`
		Target       uint64 `json:"target"`
		SourceCol    string `json:"source_col"`
		TargetCol    string `json:"target_col"`
		FkName       string `json:"fk_name"`
		Virtual      bool   `json:"virtual"`
	}

	type erResponse struct {
		Tables []erTable `json:"tables"`
		Edges  []erEdge  `json:"edges"`
	}

	result := erResponse{
		Tables: []erTable{},
		Edges:  []erEdge{},
	}

	rows, err := database.DB.Raw(
		"SELECT lm.id, lm.table_name, lm.table_comment, COUNT(mc.id) as col_count FROM logical_model lm LEFT JOIN model_column mc ON mc.model_id = lm.id WHERE lm.schema_id = ? GROUP BY lm.id, lm.table_name, lm.table_comment",
		schemaID,
	).Rows()
	if err != nil {
		utils.InternalError(c, "查询失败")
		return
	}
	defer rows.Close()

	tableIDs := make(map[uint64]bool)
	for rows.Next() {
		var t erTable
		if err := rows.Scan(&t.ID, &t.TableName, &t.TableComment, &t.ColumnCount); err != nil {
			continue
		}
		tableIDs[t.ID] = true
		result.Tables = append(result.Tables, t)
	}

	for tid := range tableIDs {
		var fks []struct {
			ID            uint64
			FkName        string
			ColumnName    string
			RefModelID    uint64
			RefColumnName string
		}
		database.DB.Raw(
			"SELECT id, fk_name, column_name, ref_model_id, ref_column_name FROM model_foreign_key WHERE model_id = ?", tid,
		).Scan(&fks)

		for _, fk := range fks {
			if tableIDs[fk.RefModelID] {
				result.Edges = append(result.Edges, erEdge{
					Source:    tid,
					Target:    fk.RefModelID,
					SourceCol: fk.ColumnName,
					TargetCol: fk.RefColumnName,
					FkName:    fk.FkName,
				})
			}
		}
	}

	// Also fetch virtual FK edges
	if len(tableIDs) > 0 {
		var vfks []struct {
			ModelID       uint64
			ColumnName    string
			RefModelID    uint64
			RefColumnName string
			FkName        string
		}
		ids := make([]uint64, 0, len(tableIDs))
		for tid := range tableIDs {
			ids = append(ids, tid)
		}
		database.DB.Raw(
			"SELECT model_id, column_name, ref_model_id, ref_column_name, fk_name FROM virtual_foreign_key WHERE model_id IN ?", ids,
		).Scan(&vfks)

		for _, vfk := range vfks {
			if tableIDs[vfk.RefModelID] {
				result.Edges = append(result.Edges, erEdge{
					Source:    vfk.ModelID,
					Target:    vfk.RefModelID,
					SourceCol: vfk.ColumnName,
					TargetCol: vfk.RefColumnName,
					FkName:    vfk.FkName,
					Virtual:   true,
				})
			}
		}
	}

	utils.Success(c, result)
}

func (h *VizHandler) ListVirtualForeignKeys(c *gin.Context) {
	schemaID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的Schema ID")
		return
	}

	vfks, err := h.modelSvc.ListVirtualForeignKeys(schemaID)
	if err != nil {
		utils.InternalError(c, "查询虚拟外键列表失败")
		return
	}
	utils.Success(c, vfks)
}

func RegisterVizRoutes(r *gin.RouterGroup, h *VizHandler, authSecret string) {
	r.Use(middleware.AuthRequired(authSecret))
	{
		r.GET("/models/:id/structure-diagram", h.StructureDiagram)
		r.GET("/logical-schemas/:id/er-diagram", h.ERDiagram)
		r.GET("/logical-schemas/:id/virtual-foreign-keys", h.ListVirtualForeignKeys)
	}
}
