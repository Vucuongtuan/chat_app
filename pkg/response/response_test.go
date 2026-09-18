package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"chatapp/pkg/i18n"
)

func TestSuccessResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(i18n.Middleware())
	r.GET("/users", func(c *gin.Context) {
		Success(c, http.StatusOK, map[string]string{"id": "123"}, i18n.MsgSuccess)
	})

	// 1. Vietnamese test (default)
	reqVI, _ := http.NewRequest(http.MethodGet, "/users", nil)
	wVI := httptest.NewRecorder()
	r.ServeHTTP(wVI, reqVI)

	if wVI.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", wVI.Code)
	}

	var respVI ResponseBody
	if err := json.Unmarshal(wVI.Body.Bytes(), &respVI); err != nil {
		t.Fatalf("Failed to parse json: %v", err)
	}
	if !respVI.Success || respVI.Message != "Thao tác thành công" {
		t.Errorf("Unexpected VI response: %+v", respVI)
	}

	// 2. English test
	reqEN, _ := http.NewRequest(http.MethodGet, "/users?lang=en", nil)
	wEN := httptest.NewRecorder()
	r.ServeHTTP(wEN, reqEN)

	var respEN ResponseBody
	if err := json.Unmarshal(wEN.Body.Bytes(), &respEN); err != nil {
		t.Fatalf("Failed to parse json: %v", err)
	}
	if !respEN.Success || respEN.Message != "Operation successful" {
		t.Errorf("Unexpected EN response: %+v", respEN)
	}
}

func TestErrorResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(i18n.Middleware())
	r.GET("/error", func(c *gin.Context) {
		Error(c, http.StatusBadRequest, i18n.ErrInvalidFile)
	})

	// Vietnamese
	reqVI, _ := http.NewRequest(http.MethodGet, "/error", nil)
	wVI := httptest.NewRecorder()
	r.ServeHTTP(wVI, reqVI)

	var respVI ResponseBody
	_ = json.Unmarshal(wVI.Body.Bytes(), &respVI)
	if respVI.Success || respVI.Error != i18n.ErrInvalidFile || respVI.Message != "File không hợp lệ" {
		t.Errorf("Unexpected VI error response: %+v", respVI)
	}

	// English
	reqEN, _ := http.NewRequest(http.MethodGet, "/error?lang=en", nil)
	wEN := httptest.NewRecorder()
	r.ServeHTTP(wEN, reqEN)

	var respEN ResponseBody
	_ = json.Unmarshal(wEN.Body.Bytes(), &respEN)
	if respEN.Success || respEN.Error != i18n.ErrInvalidFile || respEN.Message != "Invalid file" {
		t.Errorf("Unexpected EN error response: %+v", respEN)
	}
}

func TestCalculatePagination(t *testing.T) {
	p := CalculatePagination(1, 20, 55)
	if p.TotalPages != 3 {
		t.Errorf("Expected 3 pages, got %d", p.TotalPages)
	}
	if p.Page != 1 || p.PerPage != 20 || p.Total != 55 {
		t.Errorf("Unexpected pagination struct: %+v", p)
	}
}
