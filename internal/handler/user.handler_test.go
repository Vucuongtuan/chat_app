package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"chatapp/internal/dto"
	"chatapp/internal/middleware"
	"chatapp/internal/model"
	"chatapp/internal/service"
	"chatapp/pkg/i18n"
	"chatapp/pkg/response"
)

type mockUserRepo struct {
	users map[string]*model.User
}

func (m *mockUserRepo) Create(ctx context.Context, user *model.User) error {
	m.users[user.ID.String()] = user
	return nil
}

func (m *mockUserRepo) CreateUser(ctx context.Context, account *model.Account, user *model.User) error {
	m.users[user.ID.String()] = user
	return nil
}

func (m *mockUserRepo) AccountExistsByPhone(ctx context.Context, phone string) (bool, error) {
	return false, nil
}

func (m *mockUserRepo) AccountExistsByMail(ctx context.Context, mail string) (bool, error) {
	return false, nil
}

func (m *mockUserRepo) FindByID(ctx context.Context, id string) (*model.User, error) {
	user, exists := m.users[id]
	if !exists {
		return nil, gorm.ErrRecordNotFound
	}
	return user, nil
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	return nil, gorm.ErrRecordNotFound
}

func (m *mockUserRepo) FindByPhone(ctx context.Context, phone string) (*model.User, error) {
	return nil, gorm.ErrRecordNotFound
}

func (m *mockUserRepo) FindByDisplayName(ctx context.Context, displayName string) (*model.User, error) {
	for _, u := range m.users {
		if u.DisplayName == displayName {
			return u, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockUserRepo) Search(ctx context.Context, query string, limit int) ([]model.User, error) {
	var list []model.User
	for _, u := range m.users {
		list = append(list, *u)
	}
	return list, nil
}

func (m *mockUserRepo) Update(ctx context.Context, user *model.User) error {
	m.users[user.ID.String()] = user
	return nil
}

func (m *mockUserRepo) Delete(ctx context.Context, id string) error {
	delete(m.users, id)
	return nil
}

func setupTestRouter() (*gin.Engine, *mockUserRepo, uuid.UUID) {
	gin.SetMode(gin.TestMode)

	uid := uuid.New()
	repo := &mockUserRepo{
		users: map[string]*model.User{
			uid.String(): {
				ID:          uid,
				FullName:    "Nguyen Van A",
				DisplayName: "nguyenvana",
				Bio:         "Hello world",
				Status:      "online",
			},
		},
	}

	userService := service.NewUserService(repo)
	userHandler := NewUserHandler(userService)

	r := gin.New()
	r.Use(i18n.Middleware())

	// Fake auth middleware for testing
	authMw := func(c *gin.Context) {
		c.Set(middleware.ContextUserID, uid.String())
		c.Next()
	}

	api := r.Group("/api/v1")
	userHandler.RegisterRoutes(api, authMw)

	return r, repo, uid
}

func TestFindByID(t *testing.T) {
	r, _, uid := setupTestRouter()

	// 1. Success find by ID
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/users/"+uid.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var res response.ResponseBody
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if !res.Success {
		t.Errorf("Expected success true, got false")
	}

	// 2. Not found
	randomUID := uuid.New().String()
	reqNF, _ := http.NewRequest(http.MethodGet, "/api/v1/users/"+randomUID, nil)
	wNF := httptest.NewRecorder()
	r.ServeHTTP(wNF, reqNF)

	if wNF.Code != http.StatusNotFound {
		t.Fatalf("Expected 404, got %d", wNF.Code)
	}

	// 3. Invalid UUID
	reqInv, _ := http.NewRequest(http.MethodGet, "/api/v1/users/not-a-uuid", nil)
	wInv := httptest.NewRecorder()
	r.ServeHTTP(wInv, reqInv)

	if wInv.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400, got %d", wInv.Code)
	}
}

func TestUpdateMe(t *testing.T) {
	r, repo, uid := setupTestRouter()

	newName := "Nguyen Van B"
	newBio := "Updated Bio"
	body, _ := json.Marshal(dto.UpdateProfileRequest{
		FullName: &newName,
		Bio:      &newBio,
	})

	req, _ := http.NewRequest(http.MethodPut, "/api/v1/users/profile", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var res response.ResponseBody
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if !res.Success || res.Message != "Cập nhật thông tin thành công" {
		t.Errorf("Expected update success message, got %v", res.Message)
	}

	// Verify repo
	if repo.users[uid.String()].FullName != newName || repo.users[uid.String()].Bio != newBio {
		t.Errorf("User profile not updated in repository")
	}
}
