package i18n

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNormalizeLanguage(t *testing.T) {
	tests := []struct {
		input    string
		expected Language
	}{
		{"", LangVI},
		{"vi", LangVI},
		{"vi-VN", LangVI},
		{"vi_VN", LangVI},
		{"en", LangEN},
		{"en-US", LangEN},
		{"en-GB", LangEN},
		{"en-US,en;q=0.9,vi;q=0.8", LangEN},
		{"vi-VN,vi;q=0.9,en;q=0.8", LangVI},
		{"fr", LangVI}, // unsupported fallback
	}

	for _, tt := range tests {
		got := NormalizeLanguage(tt.input)
		if got != tt.expected {
			t.Errorf("NormalizeLanguage(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestT(t *testing.T) {
	// Vietnamese
	if got := T(LangVI, MsgSuccess); got != "Thao tác thành công" {
		t.Errorf("T(LangVI, MsgSuccess) = %q, want 'Thao tác thành công'", got)
	}

	// English
	if got := T(LangEN, MsgSuccess); got != "Operation successful" {
		t.Errorf("T(LangEN, MsgSuccess) = %q, want 'Operation successful'", got)
	}

	// Unknown key returns key
	if got := T(LangVI, "unknown_key"); got != "unknown_key" {
		t.Errorf("T(LangVI, unknown_key) = %q, want 'unknown_key'", got)
	}

	// Formatted message
	if got := T(LangVI, ErrFileUnsupported, ".xyz"); got != "Định dạng file không được hỗ trợ: .xyz" {
		t.Errorf("T(LangVI, ErrFileUnsupported, .xyz) = %q", got)
	}
	if got := T(LangEN, ErrFileUnsupported, ".xyz"); got != "File format is not supported: .xyz" {
		t.Errorf("T(LangEN, ErrFileUnsupported, .xyz) = %q", got)
	}
}

func TestContext(t *testing.T) {
	ctx := WithContext(context.Background(), LangEN)
	if got := FromContext(ctx); got != LangEN {
		t.Errorf("FromContext() = %v, want %v", got, LangEN)
	}

	// Empty context should return default
	if got := FromContext(context.Background()); got != DefaultLanguage {
		t.Errorf("FromContext(empty) = %v, want %v", got, DefaultLanguage)
	}
}

func TestMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(Middleware())
	r.GET("/test", func(c *gin.Context) {
		lang := GetLanguage(c)
		c.String(http.StatusOK, string(lang))
	})

	// Default
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Body.String() != string(LangVI) {
		t.Errorf("Middleware default = %s, want %s", w.Body.String(), LangVI)
	}

	// Query param ?lang=en
	req, _ = http.NewRequest(http.MethodGet, "/test?lang=en", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Body.String() != string(LangEN) {
		t.Errorf("Middleware query = %s, want %s", w.Body.String(), LangEN)
	}

	// Header Accept-Language: en-US
	req, _ = http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Accept-Language", "en-US")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Body.String() != string(LangEN) {
		t.Errorf("Middleware Accept-Language = %s, want %s", w.Body.String(), LangEN)
	}
}
