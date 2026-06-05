package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/indugate/gateway/internal/config"
	"github.com/indugate/gateway/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserNotFound       = errors.New("user not found")
	ErrLastAdmin          = errors.New("cannot delete the last admin user")
	ErrUsernameExists     = errors.New("username already exists")
)

type JWTClaims struct {
	UserID   uint           `json:"user_id"`
	Username string         `json:"username"`
	Role     model.UserRole `json:"role"`
	jwt.RegisteredClaims
}

type AuthService struct {
	db  *gorm.DB
	cfg config.AuthConfig
}

func NewAuthService(db *gorm.DB, cfg config.AuthConfig) *AuthService {
	return &AuthService{db: db, cfg: cfg}
}

func (s *AuthService) Enabled() bool {
	return s.cfg.Enabled
}

func (s *AuthService) JWTEnabled() bool {
	return s.cfg.Enabled && s.cfg.JWTSecret != ""
}

func (s *AuthService) DeviceACLEnabled() bool {
	return s.cfg.Enabled && s.cfg.DeviceACLEnabled
}

func (s *AuthService) EnsureDefaultAdmin(ctx context.Context) error {
	if !s.JWTEnabled() {
		return nil
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(&model.User{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	username := s.cfg.DefaultAdminUser
	if username == "" {
		username = "admin"
	}
	password := s.cfg.DefaultAdminPass
	if password == "" {
		password = "admin123"
	}
	_, err := s.CreateUser(ctx, username, password, model.RoleAdmin)
	return err
}

func (s *AuthService) CreateUser(ctx context.Context, username, password string, role model.UserRole) (*model.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	if role == "" {
		role = model.RoleOperator
	}
	user := &model.User{
		Username:     username,
		PasswordHash: string(hash),
		Role:         role,
	}
	if err := s.db.WithContext(ctx).Create(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func (s *AuthService) Login(ctx context.Context, username, password string) (string, *model.User, error) {
	var user model.User
	if err := s.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, ErrInvalidCredentials
		}
		return "", nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", nil, ErrInvalidCredentials
	}
	token, err := s.issueToken(&user)
	if err != nil {
		return "", nil, err
	}
	return token, &user, nil
}

func (s *AuthService) issueToken(user *model.User) (string, error) {
	expireHours := s.cfg.JWTExpireHours
	if expireHours <= 0 {
		expireHours = 24
	}
	now := time.Now()
	claims := JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.Username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(expireHours) * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWTSecret))
}

func (s *AuthService) ValidateToken(tokenStr string) (*JWTClaims, error) {
	if tokenStr == "" || s.cfg.JWTSecret == "" {
		return nil, errors.New("missing token")
	}
	token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func (s *AuthService) GetUser(ctx context.Context, id uint) (*model.User, error) {
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, id).Error; err != nil {
		return nil, ErrUserNotFound
	}
	return &user, nil
}

func (s *AuthService) ListUsers(ctx context.Context) ([]model.User, error) {
	var users []model.User
	if err := s.db.WithContext(ctx).Order("id asc").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (s *AuthService) UpdateUser(ctx context.Context, id uint, role model.UserRole) (*model.User, error) {
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, id).Error; err != nil {
		return nil, ErrUserNotFound
	}
	if role != "" {
		user.Role = role
	}
	if err := s.db.WithContext(ctx).Save(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *AuthService) ChangePassword(ctx context.Context, id uint, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	result := s.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Update("password_hash", string(hash))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (s *AuthService) DeleteUser(ctx context.Context, id uint) error {
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, id).Error; err != nil {
		return ErrUserNotFound
	}
	if user.Role == model.RoleAdmin {
		var adminCount int64
		s.db.WithContext(ctx).Model(&model.User{}).Where("role = ?", model.RoleAdmin).Count(&adminCount)
		if adminCount <= 1 {
			return ErrLastAdmin
		}
	}
	return s.db.WithContext(ctx).Delete(&model.User{}, id).Error
}
