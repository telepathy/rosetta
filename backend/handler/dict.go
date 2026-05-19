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

type DictHandler struct {
	dictSvc *service.DictService
}

func NewDictHandler(dictSvc *service.DictService) *DictHandler {
	return &DictHandler{dictSvc: dictSvc}
}

func (h *DictHandler) ListDefinitions(c *gin.Context) {
	var page service.Pagination
	if err := c.ShouldBindQuery(&page); err != nil {
		utils.BadRequest(c, "分页参数无效")
		return
	}

	dicts, total, err := h.dictSvc.ListDefinitions(page)
	if err != nil {
		utils.InternalError(c, "查询字典列表失败")
		return
	}
	utils.Success(c, gin.H{"items": dicts, "total": total})
}

func (h *DictHandler) GetDefinition(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的字典ID")
		return
	}

	dict, err := h.dictSvc.GetDefinition(id)
	if err != nil {
		utils.InternalError(c, "查询字典失败")
		return
	}
	if dict == nil {
		utils.BadRequest(c, "字典不存在")
		return
	}
	utils.Success(c, dict)
}

func (h *DictHandler) CreateDefinition(c *gin.Context) {
	var req service.CreateDictRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请填写字典名称和编码")
		return
	}

	dict, err := h.dictSvc.CreateDefinition(req)
	if err != nil {
		if errors.Is(err, service.ErrDictCodeExists) {
			utils.BadRequest(c, "字典编码已存在")
			return
		}
		utils.InternalError(c, "创建字典失败")
		return
	}
	utils.SuccessWithMessage(c, "创建成功", dict)
}

func (h *DictHandler) UpdateDefinition(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的字典ID")
		return
	}

	var req service.UpdateDictRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数校验失败")
		return
	}

	dict, err := h.dictSvc.UpdateDefinition(id, req)
	if err != nil {
		utils.InternalError(c, "更新字典失败")
		return
	}
	utils.Success(c, dict)
}

func (h *DictHandler) DeleteDefinition(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的字典ID")
		return
	}

	if err := h.dictSvc.DeleteDefinition(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.BadRequest(c, "字典不存在")
			return
		}
		utils.InternalError(c, "删除字典失败")
		return
	}
	utils.Success(c, nil)
}

func (h *DictHandler) ListItems(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的字典ID")
		return
	}

	items, err := h.dictSvc.ListItems(id)
	if err != nil {
		utils.InternalError(c, "查询字典条目失败")
		return
	}
	utils.Success(c, items)
}

func (h *DictHandler) CreateItem(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的字典ID")
		return
	}

	var req service.CreateDictItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请填写条目编码和名称")
		return
	}

	item, err := h.dictSvc.CreateItem(id, req)
	if err != nil {
		if errors.Is(err, service.ErrItemCodeExists) {
			utils.BadRequest(c, "条目编码已存在")
			return
		}
		utils.InternalError(c, "创建条目失败")
		return
	}
	utils.SuccessWithMessage(c, "创建成功", item)
}

func (h *DictHandler) UpdateItem(c *gin.Context) {
	dictID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的字典ID")
		return
	}
	itemID, err := strconv.ParseUint(c.Param("itemId"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的条目ID")
		return
	}

	var req service.UpdateDictItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数校验失败")
		return
	}

	item, err := h.dictSvc.UpdateItem(dictID, itemID, req)
	if err != nil {
		utils.InternalError(c, "更新条目失败")
		return
	}
	utils.Success(c, item)
}

func (h *DictHandler) DeleteItem(c *gin.Context) {
	dictID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的字典ID")
		return
	}
	itemID, err := strconv.ParseUint(c.Param("itemId"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的条目ID")
		return
	}

	if err := h.dictSvc.DeleteItem(dictID, itemID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.BadRequest(c, "条目不存在")
			return
		}
		utils.InternalError(c, "删除条目失败")
		return
	}
	utils.Success(c, nil)
}

func (h *DictHandler) UpdateItemsOrder(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的字典ID")
		return
	}

	var req service.UpdateDictItemOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "排序参数无效")
		return
	}

	if err := h.dictSvc.UpdateItemsOrder(id, req); err != nil {
		utils.InternalError(c, "更新排序失败")
		return
	}
	utils.Success(c, nil)
}

func RegisterDictRoutes(r *gin.RouterGroup, h *DictHandler, authSecret string) {
	dicts := r.Group("/dicts")
	dicts.Use(middleware.AuthRequired(authSecret))
	{
		dicts.GET("", h.ListDefinitions)
		dicts.POST("", h.CreateDefinition)
		dicts.GET("/:id", h.GetDefinition)
		dicts.PUT("/:id", h.UpdateDefinition)
		dicts.DELETE("/:id", h.DeleteDefinition)

		dicts.GET("/:id/items", h.ListItems)
		dicts.POST("/:id/items", h.CreateItem)
		dicts.PUT("/:id/items/:itemId", h.UpdateItem)
		dicts.DELETE("/:id/items/:itemId", h.DeleteItem)
		dicts.PUT("/:id/items/order", h.UpdateItemsOrder)
	}
}
