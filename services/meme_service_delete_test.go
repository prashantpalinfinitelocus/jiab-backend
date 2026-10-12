package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/repository"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// fakeStorage stands in for Cloud Storage: it records which objects were asked to be deleted.
type fakeStorage struct {
	utils.GCSService // every method this test does not use is left nil on purpose
	deleted          []string
	failWith         error
}

func (f *fakeStorage) DeleteObject(_ context.Context, objectPath string) error {
	f.deleted = append(f.deleted, objectPath)
	return f.failWith
}

func scratchMemeDB(t *testing.T) *gorm.DB {
	t.Helper()
	quiet := &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)}
	user := os.Getenv("USER")
	admin, err := gorm.Open(postgres.Open("host=/tmp dbname=postgres sslmode=disable user="+user), quiet)
	if err != nil {
		t.Skipf("no local postgres: %v", err)
	}
	name := fmt.Sprintf("meme_service_test_%d", time.Now().UnixNano())
	if err := admin.Exec("CREATE DATABASE " + name).Error; err != nil {
		t.Skipf("cannot create a scratch database: %v", err)
	}
	t.Cleanup(func() { admin.Exec("DROP DATABASE IF EXISTS " + name + " WITH (FORCE)") })
	db, err := gorm.Open(postgres.Open("host=/tmp dbname="+name+" sslmode=disable user="+user), quiet)
	if err != nil {
		t.Fatalf("open scratch db: %v", err)
	}
	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`).Error; err != nil {
		t.Skipf("uuid-ossp not available: %v", err)
	}
	if err := db.AutoMigrate(&entities.Meme{}, &entities.MemeText{}, &entities.MemeReaction{}, &entities.MemeViewHistory{}, &entities.MemeVoteHistory{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func statusOf(err error) int {
	var app *errs.AppError
	if errors.As(err, &app) {
		return app.Status
	}
	return 0
}

func TestMemeService_Delete(t *testing.T) {
	db := scratchMemeDB(t)
	storage := &fakeStorage{}
	svc := NewMemeService(utils.NewTransactionManager(db), repository.NewMemeRepository(), nil, nil, nil, nil, nil, nil, nil, storage, nil)
	ctx := context.Background()

	owner, stranger := uuid.NewString(), uuid.NewString()
	mk := func(assetURL string) string {
		m := &entities.Meme{ID: uuid.NewString(), UserID: owner, Status: 2, LanguageID: 1, TemplateID: "t", AssetURL: assetURL}
		if err := db.Create(m).Error; err != nil {
			t.Fatal(err)
		}
		return m.ID
	}
	exists := func(id string) bool {
		var n int64
		db.Model(&entities.Meme{}).Where("id = ?", id).Count(&n)
		return n == 1
	}

	t.Run("the owner can delete, and the stored file is removed", func(t *testing.T) {
		id := mk("Sprite-Meme/owner/file-1")
		if err := svc.Delete(ctx, owner, id); err != nil {
			t.Fatalf("owner delete failed: %v", err)
		}
		if exists(id) {
			t.Error("meme still exists")
		}
		if len(storage.deleted) != 1 || storage.deleted[0] != "Sprite-Meme/owner/file-1" {
			t.Errorf("file deletions = %v", storage.deleted)
		}
	})

	t.Run("someone else gets 403 and nothing is deleted", func(t *testing.T) {
		before := len(storage.deleted)
		id := mk("Sprite-Meme/owner/file-2")
		err := svc.Delete(ctx, stranger, id)
		if statusOf(err) != http.StatusForbidden {
			t.Fatalf("status = %d (%v), want 403", statusOf(err), err)
		}
		if !exists(id) {
			t.Error("a stranger managed to delete the meme")
		}
		if len(storage.deleted) != before {
			t.Error("a file was removed although the delete was refused")
		}
	})

	t.Run("a meme that does not exist gives 404", func(t *testing.T) {
		if got := statusOf(svc.Delete(ctx, owner, uuid.NewString())); got != http.StatusNotFound {
			t.Errorf("status = %d, want 404", got)
		}
	})

	t.Run("an id that is not a UUID gives 404, not a database error", func(t *testing.T) {
		for _, bad := range []string{"not-a-uuid", "", "1' OR '1'='1"} {
			if got := statusOf(svc.Delete(ctx, owner, bad)); got != http.StatusNotFound {
				t.Errorf("id %q: status = %d, want 404", bad, got)
			}
		}
	})

	t.Run("no logged-in user gives 401", func(t *testing.T) {
		if got := statusOf(svc.Delete(ctx, "", uuid.NewString())); got != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", got)
		}
	})

	t.Run("a failure to remove the file does not fail the delete", func(t *testing.T) {
		storage.failWith = errors.New("storage unavailable")
		defer func() { storage.failWith = nil }()
		id := mk("Sprite-Meme/owner/file-3")
		if err := svc.Delete(ctx, owner, id); err != nil {
			t.Fatalf("delete failed because of storage: %v", err)
		}
		if exists(id) {
			t.Error("meme still exists")
		}
	})

	t.Run("a meme whose asset is an external link leaves storage alone", func(t *testing.T) {
		before := len(storage.deleted)
		id := mk("https://example.com/hosted.gif")
		if err := svc.Delete(ctx, owner, id); err != nil {
			t.Fatal(err)
		}
		if len(storage.deleted) != before {
			t.Error("tried to delete a file that is not in our bucket")
		}
	})
}
