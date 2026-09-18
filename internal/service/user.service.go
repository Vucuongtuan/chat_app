package service

import (
	"context"
	"fmt"
	"net/mail"
	"regexp"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"chatapp/internal/dto"
	"chatapp/internal/model"
	"chatapp/internal/repository"
)

type UserService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

var phonePattern = regexp.MustCompile(`^\+?[0-9]{8,15}$`)

func (s *UserService) CreateUser(ctx context.Context, req dto.CreateUserRequest) (*model.User, error) {
	req.Phone = strings.TrimSpace(req.Phone)
	req.Mail = strings.TrimSpace(strings.ToLower(req.Mail))
	req.FullName = strings.TrimSpace(req.FullName)
	req.DisplayName = strings.TrimSpace(req.DisplayName)

	if !phonePattern.MatchString(req.Phone) {
		return nil, fmt.Errorf("số điện thoại không hợp lệ")
	}
	if req.Mail != "" {
		if _, err := mail.ParseAddress(req.Mail); err != nil {
			return nil, fmt.Errorf("email không hợp lệ")
		}
	}
	if len(req.Password) < 8 {
		return nil, fmt.Errorf("mật khẩu phải có ít nhất 8 ký tự")
	}
	if req.FullName == "" || len([]rune(req.FullName)) > 100 {
		return nil, fmt.Errorf("họ tên không hợp lệ")
	}
	if req.DisplayName == "" || len([]rune(req.DisplayName)) > 100 {
		return nil, fmt.Errorf("tên hiển thị không hợp lệ")
	}

	exists, err := s.repo.AccountExistsByPhone(ctx, req.Phone)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("số điện thoại đã được sử dụng")
	}
	if req.Mail != "" {
		exists, err = s.repo.AccountExistsByMail(ctx, req.Mail)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, fmt.Errorf("email đã được sử dụng")
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("không thể mã hóa mật khẩu: %w", err)
	}
	account := &model.Account{Phone: req.Phone, PasswordHash: string(hash), HashAlgo: "bcrypt"}
	if req.Mail != "" {
		account.Mail = &req.Mail
	}
	user := &model.User{FullName: req.FullName, DisplayName: req.DisplayName, Status: "offline"}

	if err := s.repo.CreateUser(ctx, account, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) FindByID(ctx context.Context, id string) (*model.User, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("id không được để trống")
	}
	return s.repo.FindByID(ctx, id)
}

func (s *UserService) FindByDisplayName(ctx context.Context, displayName string) (*model.User, error) {
	if strings.TrimSpace(displayName) == "" {
		return nil, fmt.Errorf("tên hiển thị không được để trống")
	}
	return s.repo.FindByDisplayName(ctx, displayName)
}

func (s *UserService) Search(ctx context.Context, query string, limit int) ([]model.User, error) {
	return s.repo.Search(ctx, query, limit)
}

func (s *UserService) UpdateProfile(ctx context.Context, id string, req dto.UpdateProfileRequest) (*model.User, error) {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.FullName != nil {
		fullName := strings.TrimSpace(*req.FullName)
		if fullName == "" || len([]rune(fullName)) > 100 {
			return nil, fmt.Errorf("họ tên không hợp lệ")
		}
		user.FullName = fullName
	}

	if req.DisplayName != nil {
		displayName := strings.TrimSpace(*req.DisplayName)
		if displayName == "" || len([]rune(displayName)) > 100 {
			return nil, fmt.Errorf("tên hiển thị không hợp lệ")
		}
		user.DisplayName = displayName
	}

	if req.AvatarUrl != nil {
		user.AvatarUrl = strings.TrimSpace(*req.AvatarUrl)
	}

	if req.CoverUrl != nil {
		user.CoverUrl = strings.TrimSpace(*req.CoverUrl)
	}

	if req.Bio != nil {
		user.Bio = strings.TrimSpace(*req.Bio)
	}

	if req.Status != nil {
		status := strings.TrimSpace(*req.Status)
		if status != "" {
			user.Status = status
		}
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}
