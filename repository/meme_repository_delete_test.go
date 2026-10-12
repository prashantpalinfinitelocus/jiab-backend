package repository

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// openScratchDB creates a throwaway database on the local Postgres (unix socket, current user) and
// builds the meme tables from the real entity definitions. The test is skipped when no local Postgres
// is reachable, so it never fails a machine that does not have one.
func openScratchDB(t *testing.T) *gorm.DB {
	t.Helper()
	quiet := &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)}
	user := os.Getenv("USER")
	admin, err := gorm.Open(postgres.Open("host=/tmp dbname=postgres sslmode=disable user="+user), quiet)
	if err != nil {
		t.Skipf("no local postgres: %v", err)
	}
	name := fmt.Sprintf("meme_delete_test_%d", time.Now().UnixNano())
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

// seedMeme creates one meme with a caption, a reaction, a view and a vote, and returns its id.
func seedMeme(t *testing.T, db *gorm.DB, owner string) string {
	t.Helper()
	id := uuid.NewString()
	other := uuid.NewString()
	text := "caption"
	rows := []interface{}{
		&entities.Meme{ID: id, UserID: owner, Status: 2, LanguageID: 1, TemplateID: "t", AssetURL: "Sprite-Meme/x/" + id},
		&entities.MemeText{MemeID: id, Text1: &text},
		&entities.MemeReaction{MemeID: &id, UserID: other, Reaction: "happy"},
		&entities.MemeViewHistory{MemeID: id, UserID: other},
		&entities.MemeVoteHistory{AssetID: id, UserID: other, Language: "en", VotedAt: time.Now()},
	}
	for _, r := range rows {
		if err := db.Create(r).Error; err != nil {
			t.Fatalf("seed %T: %v", r, err)
		}
	}
	return id
}

func countRows(t *testing.T, db *gorm.DB, table, column, id string) int64 {
	t.Helper()
	var n int64
	if err := db.Table(table).Where(column+" = ?", id).Count(&n).Error; err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

func memeRowCounts(t *testing.T, db *gorm.DB, id string) []int64 {
	return []int64{
		countRows(t, db, "memes", "id", id),
		countRows(t, db, "meme_texts", "meme_id", id),
		countRows(t, db, "meme_reactions", "meme_id", id),
		countRows(t, db, "meme_view_history", "meme_id", id),
		countRows(t, db, "meme_vote_history", "asset_id", id),
	}
}

func TestMemeRepository_DeleteWithDependents(t *testing.T) {
	db := openScratchDB(t)
	repo := NewMemeRepository()
	ctx := context.Background()

	doomed := seedMeme(t, db, uuid.NewString())
	neighbour := seedMeme(t, db, uuid.NewString())

	if got := memeRowCounts(t, db, doomed); fmt.Sprint(got) != "[1 1 1 1 1]" {
		t.Fatalf("seed incomplete: %v", got)
	}

	t.Run("a rolled-back delete changes nothing", func(t *testing.T) {
		tx := db.Begin()
		if err := repo.DeleteWithDependents(ctx, tx, doomed); err != nil {
			t.Fatal(err)
		}
		tx.Rollback()
		if got := memeRowCounts(t, db, doomed); fmt.Sprint(got) != "[1 1 1 1 1]" {
			t.Errorf("rollback left %v, want every row back", got)
		}
	})

	t.Run("a committed delete removes the meme and everything attached to it", func(t *testing.T) {
		tx := db.Begin()
		if err := repo.DeleteWithDependents(ctx, tx, doomed); err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit().Error; err != nil {
			t.Fatal(err)
		}
		if got := memeRowCounts(t, db, doomed); fmt.Sprint(got) != "[0 0 0 0 0]" {
			t.Errorf("after delete %v, want all zero", got)
		}
	})

	t.Run("another meme and its rows are untouched", func(t *testing.T) {
		if got := memeRowCounts(t, db, neighbour); fmt.Sprint(got) != "[1 1 1 1 1]" {
			t.Errorf("neighbour has %v, want [1 1 1 1 1]", got)
		}
	})

	t.Run("deleting a meme that does not exist is not an error", func(t *testing.T) {
		tx := db.Begin()
		defer tx.Rollback()
		if err := repo.DeleteWithDependents(ctx, tx, uuid.NewString()); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}
