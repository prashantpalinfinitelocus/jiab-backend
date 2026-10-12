package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	"github.com/Infinite-Locus-Product/coca_cola_backend/middlewares"
	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
	"github.com/gin-gonic/gin"
)

type fakeFeedService struct {
	services.MemeService
	results []services.MemeFeedResult
}

func (f *fakeFeedService) Feed(context.Context, string, *int, *string, int, *string, *bool) ([]services.MemeFeedResult, *string, error) {
	return f.results, nil, nil
}

func (f *fakeFeedService) GetUserMemes(context.Context, string, *int, int, *string) ([]services.MemeFeedResult, *string, error) {
	return f.results, nil, nil
}

func (f *fakeFeedService) Detail(context.Context, string, string) (*services.MemeFeedResult, error) {
	return &f.results[0], nil
}

func TestMemeResponses_CarryTheCreatorIDAndDisplayName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	name := "Prashant Pal"
	svc := &fakeFeedService{results: []services.MemeFeedResult{{
		Meme:     entities.Meme{ID: "meme-1", UserID: "creator-1"},
		Username: &name,
	}}}
	r := gin.New()
	r.Use(middlewares.ErrorHandler())
	r.Use(func(c *gin.Context) { c.Set("user", &entities.User{ID: "viewer"}) })
	h := NewMemeHandler(svc)
	r.GET("/memes/feed", h.Feed)
	r.GET("/memes/user", h.GetUserMemes)
	r.GET("/memes/:meme_id", h.Detail)

	get := func(path string) map[string]interface{} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("%s: status %d, body %s", path, w.Code, w.Body.String())
		}
		var body map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if items, ok := body["items"].([]interface{}); ok {
			return items[0].(map[string]interface{})
		}
		return body
	}

	for _, path := range []string{"/memes/feed", "/memes/user"} {
		item := get(path)
		if item["userId"] != "creator-1" {
			t.Errorf("%s: userId = %v, want creator-1", path, item["userId"])
		}
		if item["username"] != "Prashant Pal" {
			t.Errorf("%s: username = %v, want Prashant Pal", path, item["username"])
		}
	}
	if d := get("/memes/meme-1"); d["userId"] != "creator-1" {
		t.Errorf("detail: userId = %v, want creator-1", d["userId"])
	}
}
