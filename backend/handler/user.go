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

type UserHandler struct {
	userSvc *service.UserService
}

func NewUserHandler(userSvc *service.UserService) *UserHandler {
	return &UserHandler{userSvc: userSvc}
}

func (h *UserHandler) List(c *gin.Context) {
	var page service.Pagination
	if err := c.ShouldBindQuery(&page); err != nil {
		utils.BadRequest(c, "分页参数无效")
		return
	}

	users, total, err := h.userSvc.List(page)
	if err != nil {
		utils.InternalError(c, "查询用户列表失败")
		return
	}
	utils.Success(c, gin.H{"items": users, "total": total})
}

func (h *UserHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的用户ID")
		return
	}

	user, err := h.userSvc.GetByID(id)
	if err != nil {
		utils.InternalError(c, "查询用户失败")
		return
	}
	if user == nil {
		utils.BadRequest(c, "用户不存在")
		return
	}
	utils.Success(c, user)
}

func (h *UserHandler) Create(c *gin.Context) {
	var req service.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数校验失败，请检查用户名、密码、显示名")
		return
	}

	user, err := h.userSvc.Create(req)
	if err != nil {
		if errors.Is(err, service.ErrUsernameExists) {
			utils.BadRequest(c, "用户名已存在")
			return
		}
		utils.InternalError(c, "创建用户失败")
		return
	}
	utils.SuccessWithMessage(c, "创建成功", user)
}

func (h *UserHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的用户ID")
		return
	}

	var req service.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "参数校验失败")
		return
	}

	user, err := h.userSvc.Update(id, req)
	if err != nil {
		utils.InternalError(c, "更新用户失败")
		return
	}
	if user == nil {
		utils.BadRequest(c, "用户不存在")
		return
	}
	utils.Success(c, user)
}

func (h *UserHandler) ResetPassword(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的用户ID")
		return
	}

	var req service.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "密码长度至少6位")
		return
	}

	if err := h.userSvc.ResetPassword(id, req); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.BadRequest(c, "用户不存在")
			return
		}
		utils.InternalError(c, "重置密码失败")
		return
	}
	utils.Success(c, nil)
}

func (h *UserHandler) SetStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的用户ID")
		return
	}

	var req struct {
		Status string `json:"status" binding:"required,oneof=ACTIVE DISABLED"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "状态值必须为 ACTIVE 或 DISABLED")
		return
	}

	if err := h.userSvc.SetStatus(id, req.Status); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.BadRequest(c, "用户不存在")
			return
		}
		utils.InternalError(c, "更新用户状态失败")
		return
	}
	utils.Success(c, nil)
}

func (h *UserHandler) AssignRoles(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的用户ID")
		return
	}

	var req service.AssignRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请提供角色ID列表")
		return
	}

	if err := h.userSvc.AssignRoles(id, req); err != nil {
		utils.InternalError(c, "分配角色失败")
		return
	}
	utils.Success(c, nil)
}

func (h *UserHandler) ListRoles(c *gin.Context) {
	roles, err := h.userSvc.ListRoles()
	if err != nil {
		utils.InternalError(c, "查询角色列表失败")
		return
	}
	utils.Success(c, roles)
}

func (h *UserHandler) GetRoleModelPermissions(c *gin.Context) {
	roleID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的角色ID")
		return
	}

	perms, err := h.userSvc.GetRoleModelPermissions(roleID)
	if err != nil {
		utils.InternalError(c, "查询角色权限失败")
		return
	}
	utils.Success(c, perms)
}

func (h *UserHandler) UpdateRoleModelPermissions(c *gin.Context) {
	roleID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的角色ID")
		return
	}

	var req service.UpdateModelPermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "权限参数无效")
		return
	}

	if err := h.userSvc.UpdateRoleModelPermissions(roleID, req); err != nil {
		utils.InternalError(c, "更新角色权限失败")
		return
	}
	utils.Success(c, nil)
}

func RegisterUserRoutes(r *gin.RouterGroup, h *UserHandler, authSecret string) {
	users := r.Group("/users")
	users.Use(middleware.AuthRequired(authSecret))
	{
		users.GET("", h.List)
		users.POST("", h.Create)
		users.GET("/:id", h.Get)
		users.PUT("/:id", h.Update)
		users.PUT("/:id/password", h.ResetPassword)
		users.PUT("/:id/status", h.SetStatus)
		users.PUT("/:id/roles", h.AssignRoles)
	}

	roles := r.Group("/roles")
	roles.Use(middleware.AuthRequired(authSecret))
	{
		roles.GET("", h.ListRoles)
		roles.GET("/:id/model-permissions", h.GetRoleModelPermissions)
		roles.PUT("/:id/model-permissions", h.UpdateRoleModelPermissions)
	}
}
