package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Memes get random UUID ids, so id order is unrelated to date order. Several share a created_at on purpose.
func seedPagingMemes(t *testing.T, db *gorm.DB, owner string, n int) map[string]bool {
	t.Helper()
	want := map[string]bool{}
	base := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	for i := 0; i < n; i++ {
		id := uuid.NewString()
		m := &entities.Meme{ID: id, UserID: owner, Status: 2, LanguageID: 1, TemplateID: "t", AssetURL: "x/" + id,
			ViewCount: i % 4, CreatedAt: base.Add(time.Duration(i/3) * time.Minute)}
		if err := db.Create(m).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
		want[id] = true
	}
	return want
}

func walk(t *testing.T, want map[string]bool, page func(cursor *string) ([]entities.Meme, *string, error)) {
	t.Helper()
	seen := map[string]int{}
	var cursor *string
	for i := 0; i < 20; i++ {
		memes, next, err := page(cursor)
		if err != nil {
			t.Fatalf("page: %v", err)
		}
		if len(memes) == 0 {
			break
		}
		for _, m := range memes {
			seen[m.ID]++
		}
		cursor = next
	}
	if len(seen) != len(want) {
		t.Fatalf("paging returned %d distinct memes, want %d", len(seen), len(want))
	}
	for id, c := range seen {
		if !want[id] || c != 1 {
			t.Fatalf("meme %s returned %d times (expected once)", id, c)
		}
	}
}

func TestMemeRepository_PagingFollowsTheSortOrder(t *testing.T) {
	db := openScratchDB(t)
	repo := NewMemeRepository()
	ctx := context.Background()
	owner := uuid.NewString()
	want := seedPagingMemes(t, db, owner, 41)
	lang := 1
	trending := true

	t.Run("feed", func(t *testing.T) {
		walk(t, want, func(c *string) ([]entities.Meme, *string, error) {
			return repo.FindApprovedFeed(ctx, db, owner, &lang, nil, 6, c, nil)
		})
	})
	t.Run("trending feed", func(t *testing.T) {
		walk(t, want, func(c *string) ([]entities.Meme, *string, error) {
			return repo.FindApprovedFeed(ctx, db, owner, &lang, nil, 6, c, &trending)
		})
	})
	t.Run("my memes", func(t *testing.T) {
		walk(t, want, func(c *string) ([]entities.Meme, *string, error) {
			return repo.FindByUserID(ctx, db, owner, nil, 6, c)
		})
	})
	t.Run("pending", func(t *testing.T) {
		db.Model(&entities.Meme{}).Where("user_id = ?", owner).Update("status", 1)
		walk(t, want, func(c *string) ([]entities.Meme, *string, error) {
			return repo.FindPending(ctx, db, 6, c)
		})
	})
}
