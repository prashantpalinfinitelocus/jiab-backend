package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/middlewares"
	"github.com/Infinite-Locus-Product/coca_cola_backend/services"
	"github.com/gin-gonic/gin"
)

// fakeMemeService answers Delete only; every other method would panic, which these tests never call.
type fakeMemeService struct {
	services.MemeService
	err       error
	gotUser   string
	gotMeme   string
	wasCalled bool
}

func (f *fakeMemeService) Delete(_ context.Context, userID, memeID string) error {
	f.wasCalled, f.gotUser, f.gotMeme = true, userID, memeID
	return f.err
}

func deleteRouter(svc services.MemeService, loggedInAs string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middlewares.ErrorHandler())
	r.Use(func(c *gin.Context) {
		if loggedInAs != "" {
			c.Set("user", &entities.User{ID: loggedInAs})
		}
	})
	r.DELETE("/memes/:meme_id", NewMemeHandler(svc).Delete)
	return r
}

func doDelete(r *gin.Engine, id string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/memes/"+id, nil))
	return w
}

func TestMemeHandler_Delete_StatusCodes(t *testing.T) {
	cases := []struct {
		name     string
		user     string
		svcErr   error
		wantCode int
		wantCall bool
	}{
		{"owner deletes", "u1", nil, http.StatusOK, true},
		{"someone else's meme", "u2", errs.NewForbiddenError("You can only delete your own memes", nil), http.StatusForbidden, true},
		{"no such meme", "u1", errs.NewNotFoundError("Meme not found", nil), http.StatusNotFound, true},
		{"not logged in", "", nil, http.StatusUnauthorized, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc := &fakeMemeService{err: c.svcErr}
			w := doDelete(deleteRouter(svc, c.user), "abc")
			if w.Code != c.wantCode {
				t.Fatalf("status = %d, want %d (body %s)", w.Code, c.wantCode, w.Body.String())
			}
			if svc.wasCalled != c.wantCall {
				t.Errorf("service called = %v, want %v", svc.wasCalled, c.wantCall)
			}
			if c.wantCall && (svc.gotUser != c.user || svc.gotMeme != "abc") {
				t.Errorf("service got user=%q meme=%q", svc.gotUser, svc.gotMeme)
			}
		})
	}
}
