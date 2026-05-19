package service

import (
	"errors"
	"rosetta/config"
	"rosetta/models"
	"rosetta/utils"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	db     *gorm.DB
	jwtCfg config.JWTConfig
}

func NewAuthService(db *gorm.DB, jwtCfg config.JWTConfig) *AuthService {
	return &AuthService{db: db, jwtCfg: jwtCfg}
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token    string          `json:"token"`
	User     UserInfo        `json:"user"`
}

type UserInfo struct {
	ID          uint64   `json:"id"`
	Username    string   `json:"username"`
	DisplayName string   `json:"display_name"`
	Email       string   `json:"email"`
	Status      string   `json:"status"`
	Roles       []string `json:"roles"`
}

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserDisabled       = errors.New("user is disabled")
	ErrUserNotFound       = errors.New("user not found")
)

func (s *AuthService) Login(req LoginRequest) (*LoginResponse, error) {
	var user models.RbacUser
	if err := s.db.Preload("Roles").Where("username = ?", req.Username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if user.Status != "ACTIVE" {
		return nil, ErrUserDisabled
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := utils.GenerateToken(user.ID, user.Username, s.jwtCfg.Secret, s.jwtCfg.ExpireHours)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	_ = s.db.Model(&user).Update("last_login_at", now)

	roleCodes := make([]string, len(user.Roles))
	for i, role := range user.Roles {
		roleCodes[i] = role.RoleCode
	}

	return &LoginResponse{
		Token: token,
		User: UserInfo{
			ID:          user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
			Email:       user.Email,
			Status:      user.Status,
			Roles:       roleCodes,
		},
	}, nil
}

func (s *AuthService) GetCurrentUser(userID uint64) (*UserInfo, error) {
	var user models.RbacUser
	if err := s.db.Preload("Roles").First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	roleCodes := make([]string, len(user.Roles))
	for i, role := range user.Roles {
		roleCodes[i] = role.RoleCode
	}

	return &UserInfo{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Email:       user.Email,
		Status:      user.Status,
		Roles:       roleCodes,
	}, nil
}
