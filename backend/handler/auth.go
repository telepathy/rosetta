package handler

import (
	"errors"
	"net/http"

	"rosetta/service"
	"rosetta/utils"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authSvc *service.AuthService
}

func NewAuthHandler(authSvc *service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req service.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请输入用户名和密码")
		return
	}

	resp, err := h.authSvc.Login(req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			utils.Error(c, http.StatusUnauthorized, 401, "用户名或密码错误")
			return
		}
		if errors.Is(err, service.ErrUserDisabled) {
			utils.Error(c, http.StatusForbidden, 403, "账号已被禁用")
			return
		}
		utils.InternalError(c, "登录失败")
		return
	}

	utils.SuccessWithMessage(c, "登录成功", resp)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	utils.Success(c, nil)
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Unauthorized(c, "未登录")
		return
	}

	userInfo, err := h.authSvc.GetCurrentUser(userID.(uint64))
	if err != nil {
		utils.InternalError(c, "获取用户信息失败")
		return
	}

	utils.Success(c, userInfo)
}
