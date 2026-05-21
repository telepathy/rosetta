package handler

import (
	"strconv"

	"rosetta/middleware"
	"rosetta/service"
	"rosetta/utils"

	"github.com/gin-gonic/gin"
)

type ReverseEngHandler struct {
	revEngSvc *service.ReverseEngService
}

func NewReverseEngHandler(revEngSvc *service.ReverseEngService) *ReverseEngHandler {
	return &ReverseEngHandler{revEngSvc: revEngSvc}
}

func (h *ReverseEngHandler) ListTables(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的实例ID")
		return
	}

	tables, err := h.revEngSvc.ListTables(id, c.Query("schema"))
	if err != nil {
		utils.Error(c, 422, 422, err.Error())
		return
	}
	utils.Success(c, tables)
}

func (h *ReverseEngHandler) Preview(c *gin.Context) {
	var req struct {
		InstanceID uint64 `json:"instance_id" binding:"required"`
		SchemaName string `json:"schema_name" binding:"required"`
		TableName  string `json:"table_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请填写实例ID、Schema和表名")
		return
	}

	preview, err := h.revEngSvc.Preview(req.InstanceID, req.SchemaName, req.TableName)
	if err != nil {
		utils.Error(c, 422, 422, err.Error())
		return
	}
	utils.Success(c, preview)
}

func (h *ReverseEngHandler) Import(c *gin.Context) {
	var req service.ImportRevEngRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请填写必填字段")
		return
	}

	userID, _ := c.Get(middleware.ContextUserID)
	req.CreatedBy = userID.(uint64)

	model, err := h.revEngSvc.Import(req)
	if err != nil {
		utils.Error(c, 422, 422, err.Error())
		return
	}
	utils.SuccessWithMessage(c, "导入成功", model)
}

func RegisterReverseEngRoutes(r *gin.RouterGroup, h *ReverseEngHandler, authSecret string) {
	re := r.Group("")
	re.Use(middleware.AuthRequired(authSecret))
	{
		re.GET("/instances/:id/tables", h.ListTables)
		re.POST("/models/reverse-engineer", h.Preview)
		re.POST("/models/reverse-engineer/import", h.Import)
	}
}
