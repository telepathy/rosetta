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

type InstanceHandler struct {
	instSvc *service.InstanceService
}

func NewInstanceHandler(instSvc *service.InstanceService) *InstanceHandler {
	return &InstanceHandler{instSvc: instSvc}
}

func (h *InstanceHandler) List(c *gin.Context) {
	var page service.Pagination
	if err := c.ShouldBindQuery(&page); err != nil {
		utils.BadRequest(c, "分页参数无效")
		return
	}

	instances, total, err := h.instSvc.List(page)
	if err != nil {
		utils.InternalError(c, "查询实例列表失败")
		return
	}
	utils.Success(c, gin.H{"items": instances, "total": total})
}

func (h *InstanceHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的实例ID")
		return
	}

	inst, err := h.instSvc.GetByID(id)
	if err != nil {
		utils.InternalError(c, "查询实例失败")
		return
	}
	if inst == nil {
		utils.BadRequest(c, "实例不存在")
		return
	}
	utils.Success(c, inst)
}

func (h *InstanceHandler) Create(c *gin.Context) {
	var req service.CreateInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请填写必填字段：名称、类型、主机、端口")
		return
	}

	inst, err := h.instSvc.Create(req)
	if err != nil {
		utils.InternalError(c, "创建实例失败")
		return
	}
	utils.SuccessWithMessage(c, "创建成功", inst)
}

func (h *InstanceHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的实例ID")
		return
	}

	var req service.UpdateInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数校验失败")
		return
	}

	inst, err := h.instSvc.Update(id, req)
	if err != nil {
		utils.InternalError(c, "更新实例失败")
		return
	}
	if inst == nil {
		utils.BadRequest(c, "实例不存在")
		return
	}
	utils.Success(c, inst)
}

func (h *InstanceHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的实例ID")
		return
	}

	if err := h.instSvc.Delete(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.BadRequest(c, "实例不存在")
			return
		}
		utils.InternalError(c, "删除实例失败")
		return
	}
	utils.Success(c, nil)
}

func (h *InstanceHandler) TestConnection(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的实例ID")
		return
	}

	if err := h.instSvc.TestConnection(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.BadRequest(c, "实例不存在")
			return
		}
		utils.Error(c, 422, 422, err.Error())
		return
	}
	utils.SuccessWithMessage(c, "连接成功", nil)
}

func (h *InstanceHandler) ListSchemas(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的实例ID")
		return
	}

	schemas, err := h.instSvc.ListSchemas(id)
	if err != nil {
		utils.InternalError(c, "查询Schema列表失败")
		return
	}
	utils.Success(c, schemas)
}

func (h *InstanceHandler) CreateSchema(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的实例ID")
		return
	}

	var req service.CreateSchemaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请填写Schema名称和数据层级")
		return
	}

	schema, err := h.instSvc.CreateSchema(id, req)
	if err != nil {
		utils.InternalError(c, "创建Schema失败")
		return
	}
	utils.SuccessWithMessage(c, "创建成功", schema)
}

func RegisterInstanceRoutes(r *gin.RouterGroup, h *InstanceHandler, authSecret string) {
	instances := r.Group("/instances")
	instances.Use(middleware.AuthRequired(authSecret))
	{
		instances.GET("", h.List)
		instances.POST("", h.Create)
		instances.GET("/:id", h.Get)
		instances.PUT("/:id", h.Update)
		instances.DELETE("/:id", h.Delete)
		instances.POST("/:id/test", h.TestConnection)
		instances.GET("/:id/schemas", h.ListSchemas)
		instances.POST("/:id/schemas", h.CreateSchema)
	}
}
