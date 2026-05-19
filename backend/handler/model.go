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

type ModelHandler struct {
	modelSvc *service.ModelService
}

func NewModelHandler(modelSvc *service.ModelService) *ModelHandler {
	return &ModelHandler{modelSvc: modelSvc}
}

func (h *ModelHandler) List(c *gin.Context) {
	var page service.Pagination
	_ = c.ShouldBindQuery(&page)
	keyword := c.Query("keyword")

	items, total, err := h.modelSvc.List(page, keyword)
	if err != nil {
		utils.InternalError(c, "查询模型列表失败")
		return
	}
	utils.Success(c, gin.H{"items": items, "total": total})
}

func (h *ModelHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的模型ID")
		return
	}

	detail, err := h.modelSvc.GetDetail(id)
	if err != nil {
		utils.InternalError(c, "查询模型详情失败")
		return
	}
	if detail == nil {
		utils.BadRequest(c, "模型不存在")
		return
	}
	utils.Success(c, detail)
}

func (h *ModelHandler) Create(c *gin.Context) {
	var req service.CreateModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请填写表名")
		return
	}

	userID, _ := c.Get(middleware.ContextUserID)
	model, err := h.modelSvc.Create(req, userID.(uint64))
	if err != nil {
		if errors.Is(err, service.ErrTableNameExists) {
			utils.BadRequest(c, "表名已存在")
			return
		}
		utils.InternalError(c, "创建模型失败")
		return
	}
	utils.SuccessWithMessage(c, "创建成功", model)
}

func (h *ModelHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的模型ID")
		return
	}

	var req service.UpdateModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数校验失败")
		return
	}

	userID, _ := c.Get(middleware.ContextUserID)
	model, err := h.modelSvc.Update(id, req, userID.(uint64))
	if err != nil {
		utils.InternalError(c, "更新模型失败")
		return
	}
	utils.Success(c, model)
}

func (h *ModelHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的模型ID")
		return
	}

	if err := h.modelSvc.Delete(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.BadRequest(c, "模型不存在")
			return
		}
		utils.InternalError(c, "删除模型失败")
		return
	}
	utils.Success(c, nil)
}

func (h *ModelHandler) BatchUpdateColumns(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的模型ID")
		return
	}

	var req service.BatchUpdateColumnsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "字段参数无效")
		return
	}

	columns, err := h.modelSvc.BatchUpdateColumns(id, req)
	if err != nil {
		utils.InternalError(c, "更新字段失败")
		return
	}
	utils.Success(c, columns)
}

func (h *ModelHandler) CreateIndex(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的模型ID")
		return
	}

	var req service.CreateIndexRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "索引参数无效")
		return
	}

	idx, err := h.modelSvc.CreateIndex(id, req)
	if err != nil {
		utils.InternalError(c, "创建索引失败")
		return
	}
	utils.SuccessWithMessage(c, "创建成功", idx)
}

func (h *ModelHandler) UpdateIndex(c *gin.Context) {
	modelID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	indexID, _ := strconv.ParseUint(c.Param("indexId"), 10, 64)

	var req service.CreateIndexRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "索引参数无效")
		return
	}

	idx, err := h.modelSvc.UpdateIndex(modelID, indexID, req)
	if err != nil {
		utils.InternalError(c, "更新索引失败")
		return
	}
	utils.Success(c, idx)
}

func (h *ModelHandler) DeleteIndex(c *gin.Context) {
	modelID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	indexID, _ := strconv.ParseUint(c.Param("indexId"), 10, 64)

	if err := h.modelSvc.DeleteIndex(modelID, indexID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.BadRequest(c, "索引不存在")
			return
		}
		utils.InternalError(c, "删除索引失败")
		return
	}
	utils.Success(c, nil)
}

func (h *ModelHandler) CreateForeignKey(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的模型ID")
		return
	}

	var req service.CreateForeignKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "外键参数无效")
		return
	}

	fk, err := h.modelSvc.CreateForeignKey(id, req)
	if err != nil {
		utils.InternalError(c, "创建外键失败")
		return
	}
	utils.SuccessWithMessage(c, "创建成功", fk)
}

func (h *ModelHandler) DeleteForeignKey(c *gin.Context) {
	modelID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	fkID, _ := strconv.ParseUint(c.Param("fkId"), 10, 64)

	if err := h.modelSvc.DeleteForeignKey(modelID, fkID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.BadRequest(c, "外键不存在")
			return
		}
		utils.InternalError(c, "删除外键失败")
		return
	}
	utils.Success(c, nil)
}

func (h *ModelHandler) RenderDDL(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的模型ID")
		return
	}

	dialect := c.Query("dialect")
	if dialect == "" {
		dialect = "MYSQL"
	}

	ddl, err := h.modelSvc.RenderDDL(id, dialect)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.BadRequest(c, "模型不存在")
			return
		}
		utils.InternalError(c, "渲染DDL失败")
		return
	}
	utils.Success(c, gin.H{"dialect": dialect, "ddl": ddl})
}

func (h *ModelHandler) Deploy(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的模型ID")
		return
	}

	var req service.DeployRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请指定目标Schema和方言")
		return
	}

	userID, _ := c.Get(middleware.ContextUserID)
	deploy, err := h.modelSvc.Deploy(id, req, userID.(uint64))
	if err != nil {
		utils.InternalError(c, "部署失败")
		return
	}
	utils.SuccessWithMessage(c, "部署成功", deploy)
}

func (h *ModelHandler) ListDeployments(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的模型ID")
		return
	}

	deps, err := h.modelSvc.ListDeployments(id)
	if err != nil {
		utils.InternalError(c, "查询部署历史失败")
		return
	}
	utils.Success(c, deps)
}

func RegisterModelRoutes(r *gin.RouterGroup, h *ModelHandler, authSecret string) {
	models := r.Group("/models")
	models.Use(middleware.AuthRequired(authSecret))
	{
		models.GET("", h.List)
		models.POST("", h.Create)
		models.GET("/:id", h.Get)
		models.PUT("/:id", h.Update)
		models.DELETE("/:id", h.Delete)

		models.PUT("/:id/columns", h.BatchUpdateColumns)

		models.POST("/:id/indexes", h.CreateIndex)
		models.PUT("/:id/indexes/:indexId", h.UpdateIndex)
		models.DELETE("/:id/indexes/:indexId", h.DeleteIndex)

		models.POST("/:id/foreign-keys", h.CreateForeignKey)
		models.DELETE("/:id/foreign-keys/:fkId", h.DeleteForeignKey)

		models.GET("/:id/ddl", h.RenderDDL)
		models.POST("/:id/deploy", h.Deploy)
		models.GET("/:id/deployments", h.ListDeployments)
	}
}
