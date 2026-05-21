package handler

import (
	"errors"
	"strconv"

	"rosetta/middleware"
	"rosetta/service"
	"rosetta/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DatabaseHandler struct {
	dbSvc *service.DatabaseService
}

func NewDatabaseHandler(dbSvc *service.DatabaseService) *DatabaseHandler {
	return &DatabaseHandler{dbSvc: dbSvc}
}

func (h *DatabaseHandler) ListDatabases(c *gin.Context) {
	dbs, err := h.dbSvc.ListDatabases()
	if err != nil {
		utils.InternalError(c, "查询逻辑库列表失败")
		return
	}
	utils.Success(c, dbs)
}

func (h *DatabaseHandler) GetDatabase(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的ID")
		return
	}
	db, err := h.dbSvc.GetDatabase(id)
	if err != nil {
		utils.InternalError(c, "查询失败")
		return
	}
	if db == nil {
		utils.BadRequest(c, "逻辑库不存在")
		return
	}
	utils.Success(c, db)
}

func (h *DatabaseHandler) CreateDatabase(c *gin.Context) {
	var req service.CreateDatabaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请输入逻辑库名称")
		return
	}
	db, err := h.dbSvc.CreateDatabase(req)
	if err != nil {
		utils.InternalError(c, "创建失败")
		return
	}
	utils.SuccessWithMessage(c, "创建成功", db)
}

func (h *DatabaseHandler) UpdateDatabase(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的ID")
		return
	}
	var req service.CreateDatabaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数无效")
		return
	}
	db, err := h.dbSvc.UpdateDatabase(id, req)
	if err != nil {
		utils.InternalError(c, "更新失败")
		return
	}
	utils.Success(c, db)
}

func (h *DatabaseHandler) DeleteDatabase(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的ID")
		return
	}
	if err := h.dbSvc.DeleteDatabase(id); err != nil {
		utils.InternalError(c, "删除失败")
		return
	}
	utils.Success(c, nil)
}

func (h *DatabaseHandler) ListSchemas(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的ID")
		return
	}
	schemas, err := h.dbSvc.ListSchemas(id)
	if err != nil {
		utils.InternalError(c, "查询Schema列表失败")
		return
	}
	utils.Success(c, schemas)
}

func (h *DatabaseHandler) CreateSchema(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的ID")
		return
	}
	var req service.CreateLogicalSchemaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请输入Schema名称")
		return
	}
	schema, err := h.dbSvc.CreateSchema(id, req)
	if err != nil {
		utils.InternalError(c, "创建Schema失败")
		return
	}
	utils.SuccessWithMessage(c, "创建成功", schema)
}

func (h *DatabaseHandler) DeleteSchema(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("schemaId"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的ID")
		return
	}
	if err := h.dbSvc.DeleteSchema(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.BadRequest(c, "Schema不存在")
			return
		}
		utils.InternalError(c, "删除失败")
		return
	}
	utils.Success(c, nil)
}

func (h *DatabaseHandler) ListMappedInstances(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的ID")
		return
	}
	instances, err := h.dbSvc.ListMappedInstances(id)
	if err != nil {
		utils.InternalError(c, "查询失败")
		return
	}
	utils.Success(c, instances)
}

func (h *DatabaseHandler) MapInstance(c *gin.Context) {
	dbID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的ID")
		return
	}
	var req struct {
		InstanceID uint64 `json:"instance_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请选择实例")
		return
	}
	if err := h.dbSvc.MapInstance(dbID, req.InstanceID); err != nil {
		utils.InternalError(c, "映射失败")
		return
	}
	utils.SuccessWithMessage(c, "映射成功", nil)
}

func (h *DatabaseHandler) UnmapInstance(c *gin.Context) {
	dbID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的ID")
		return
	}
	instID, err := strconv.ParseUint(c.Param("instanceId"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的实例ID")
		return
	}
	if err := h.dbSvc.UnmapInstance(dbID, instID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.BadRequest(c, "映射不存在")
			return
		}
		utils.InternalError(c, "解除映射失败")
		return
	}
	utils.Success(c, nil)
}

func RegisterDatabaseRoutes(r *gin.RouterGroup, h *DatabaseHandler, authSecret string) {
	dbs := r.Group("/databases")
	dbs.Use(middleware.AuthRequired(authSecret))
	{
		dbs.GET("", h.ListDatabases)
		dbs.POST("", h.CreateDatabase)
		dbs.GET("/:id", h.GetDatabase)
		dbs.PUT("/:id", h.UpdateDatabase)
		dbs.DELETE("/:id", h.DeleteDatabase)

		dbs.GET("/:id/schemas", h.ListSchemas)
		dbs.POST("/:id/schemas", h.CreateSchema)
		dbs.DELETE("/:id/schemas/:schemaId", h.DeleteSchema)

		dbs.GET("/:id/instances", h.ListMappedInstances)
		dbs.POST("/:id/instances", h.MapInstance)
		dbs.DELETE("/:id/instances/:instanceId", h.UnmapInstance)
	}
}
