package service

import (
	"errors"

	"rosetta/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	db *gorm.DB
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

type CreateUserRequest struct {
	Username    string `json:"username" binding:"required,min=2,max=128"`
	Password    string `json:"password" binding:"required,min=6,max=128"`
	DisplayName string `json:"display_name" binding:"required"`
	Email       string `json:"email"`
	RoleIDs     []uint64 `json:"role_ids"`
}

type UpdateUserRequest struct {
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
}

type ResetPasswordRequest struct {
	Password string `json:"password" binding:"required,min=6,max=128"`
}

type AssignRolesRequest struct {
	RoleIDs []uint64 `json:"role_ids" binding:"required"`
}

type Pagination struct {
	Page     int   `form:"page" binding:"omitempty,min=0"`
	PageSize int   `form:"page_size" binding:"omitempty,min=1,max=100"`
}

func (p *Pagination) Normalize() {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PageSize <= 0 || p.PageSize > 100 {
		p.PageSize = 20
	}
}

func (p *Pagination) Offset() int {
	return (p.Page - 1) * p.PageSize
}

var (
	ErrUsernameExists = errors.New("用户名已存在")
)

func (s *UserService) List(page Pagination) ([]models.RbacUser, int64, error) {
	page.Normalize()
	var users []models.RbacUser
	var total int64

	if err := s.db.Model(&models.RbacUser{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := s.db.Offset(page.Offset()).Limit(page.PageSize).
		Omit("password").Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (s *UserService) GetByID(id uint64) (*models.RbacUser, error) {
	var user models.RbacUser
	if err := s.db.Preload("Roles").Omit("password").First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (s *UserService) Create(req CreateUserRequest) (*models.RbacUser, error) {
	var exist models.RbacUser
	if err := s.db.Where("username = ?", req.Username).First(&exist).Error; err == nil {
		return nil, ErrUsernameExists
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := models.RbacUser{
		Username:    req.Username,
		Password:    string(hashed),
		DisplayName: req.DisplayName,
		Email:       req.Email,
		Status:      "ACTIVE",
	}

	tx := s.db.Begin()
	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if len(req.RoleIDs) > 0 {
		userRoles := make([]models.RbacUserRole, len(req.RoleIDs))
		for i, rid := range req.RoleIDs {
			userRoles[i] = models.RbacUserRole{UserID: user.ID, RoleID: rid}
		}
		if err := tx.Create(&userRoles).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	tx.Commit()
	return s.GetByID(user.ID)
}

func (s *UserService) Update(id uint64, req UpdateUserRequest) (*models.RbacUser, error) {
	updates := map[string]interface{}{}
	if req.DisplayName != "" {
		updates["display_name"] = req.DisplayName
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}
	if len(updates) == 0 {
		return s.GetByID(id)
	}
	if err := s.db.Model(&models.RbacUser{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.GetByID(id)
}

func (s *UserService) ResetPassword(id uint64, req ResetPasswordRequest) error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	result := s.db.Model(&models.RbacUser{}).Where("id = ?", id).Update("password", string(hashed))
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

func (s *UserService) SetStatus(id uint64, status string) error {
	if status != "ACTIVE" && status != "DISABLED" {
		return errors.New("invalid status")
	}
	result := s.db.Model(&models.RbacUser{}).Where("id = ?", id).Update("status", status)
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

func (s *UserService) AssignRoles(userID uint64, req AssignRolesRequest) error {
	tx := s.db.Begin()
	if err := tx.Where("user_id = ?", userID).Delete(&models.RbacUserRole{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if len(req.RoleIDs) > 0 {
		userRoles := make([]models.RbacUserRole, len(req.RoleIDs))
		for i, rid := range req.RoleIDs {
			userRoles[i] = models.RbacUserRole{UserID: userID, RoleID: rid}
		}
		if err := tx.Create(&userRoles).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

func (s *UserService) ListRoles() ([]models.RbacRole, error) {
	var roles []models.RbacRole
	if err := s.db.Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

func (s *UserService) GetRoleModelPermissions(roleID uint64) ([]models.RbacRoleModelPermission, error) {
	var perms []models.RbacRoleModelPermission
	if err := s.db.Where("role_id = ?", roleID).Find(&perms).Error; err != nil {
		return nil, err
	}
	return perms, nil
}

type UpdateModelPermissionsRequest struct {
	Permissions []ModelPermissionEntry `json:"permissions"`
}

type ModelPermissionEntry struct {
	ModelID    uint64 `json:"model_id"`
	Permission string `json:"permission"`
}

func (s *UserService) UpdateRoleModelPermissions(roleID uint64, req UpdateModelPermissionsRequest) error {
	tx := s.db.Begin()
	if err := tx.Where("role_id = ?", roleID).Delete(&models.RbacRoleModelPermission{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	for _, p := range req.Permissions {
		perm := models.RbacRoleModelPermission{
			RoleID:     roleID,
			ModelID:    p.ModelID,
			Permission: p.Permission,
		}
		if err := tx.Create(&perm).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}
