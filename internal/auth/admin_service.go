package auth

import (
	"context"
	"errors"
	"time"

	"github.com/IanShaw027/sub2api-plugin-market/ent"
	"github.com/IanShaw027/sub2api-plugin-market/ent/adminuser"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserNotActive      = errors.New("user is not active")
	ErrUserNotFound       = errors.New("user not found")
)

// AdminService 管理员服务
type AdminService struct {
	client *ent.Client
}

// NewAdminService 创建管理员服务
func NewAdminService(client *ent.Client) *AdminService {
	return &AdminService{client: client}
}

// Authenticate 认证管理员
func (s *AdminService) Authenticate(ctx context.Context, username, password string) (*ent.AdminUser, error) {
	user, err := s.client.AdminUser.Query().
		Where(adminuser.UsernameEQ(username)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if !user.IsActive {
		return nil, ErrUserNotActive
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// 更新最后登录时间
	now := time.Now()
	user, err = user.Update().
		SetLastLoginAt(now).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// CreateAdmin 创建管理员
func (s *AdminService) CreateAdmin(ctx context.Context, username, email, password, role string) (*ent.AdminUser, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user, err := s.client.AdminUser.Create().
		SetID(uuid.New()).
		SetUsername(username).
		SetEmail(email).
		SetPasswordHash(string(hashedPassword)).
		SetRole(adminuser.Role(role)).
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetByID 根据 ID 获取管理员
func (s *AdminService) GetByID(ctx context.Context, id uuid.UUID) (*ent.AdminUser, error) {
	user, err := s.client.AdminUser.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// GetByUsername 根据用户名获取管理员
func (s *AdminService) GetByUsername(ctx context.Context, username string) (*ent.AdminUser, error) {
	user, err := s.client.AdminUser.Query().
		Where(adminuser.UsernameEQ(username)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}
