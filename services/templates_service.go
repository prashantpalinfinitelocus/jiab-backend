package services

import (
	"context"
	"strings"

	"github.com/Infinite-Locus-Product/coca_cola_backend/dtos"
	"github.com/Infinite-Locus-Product/coca_cola_backend/entities"
	"github.com/Infinite-Locus-Product/coca_cola_backend/repository"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
)

// TemplatesService serves meme GIF templates from the internally-hosted
// catalog (GCS-backed, see entities.MemeTemplate) rather than a third-party
// provider. The method shapes/signatures are kept identical to the previous
// Giphy-backed implementation so the frontend's existing /templates/* calls
// need no changes.
type TemplatesService interface {
	Autocomplete(query string, limit, offset int, templateType string) ([]dtos.GiphyAutocompleteItem, error)
	Search(query string, limit, offset int, rating, lang, bundle, templateType string) ([]dtos.GiphyTemplate, error)
	Trending(limit, offset int, rating, bundle, templateType string) ([]dtos.GiphyTemplate, error)
}

type templatesService struct {
	txnManager *utils.TransactionManager
	repo       repository.MemeTemplateRepository
}

func NewTemplatesService(txnManager *utils.TransactionManager, repo repository.MemeTemplateRepository) TemplatesService {
	return &templatesService{txnManager: txnManager, repo: repo}
}

// Autocomplete suggests genre names starting with query, standing in for
// Giphy's free-text autocomplete now that the catalog is a small, fixed set
// of internally-tagged genres rather than an open-ended search index.
func (s *templatesService) Autocomplete(query string, limit, offset int, templateType string) ([]dtos.GiphyAutocompleteItem, error) {
	if limit <= 0 {
		limit = 25
	}

	ctx := context.Background()
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.CommitTxn(tx)

	genres, err := s.repo.DistinctGenres(ctx, tx)
	if err != nil {
		return nil, err
	}

	out := make([]dtos.GiphyAutocompleteItem, 0, len(genres))
	q := strings.ToLower(strings.TrimSpace(query))
	for _, g := range genres {
		if q == "" || strings.HasPrefix(strings.ToLower(g), q) {
			out = append(out, dtos.GiphyAutocompleteItem{Name: g})
		}
	}
	if offset < len(out) {
		out = out[offset:]
	} else {
		out = nil
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// Search matches query against genre (exact, case-insensitive) or filename
// (substring). rating/lang/bundle have no equivalent in the internal catalog
// and are accepted-but-ignored to keep the existing API contract.
func (s *templatesService) Search(query string, limit, offset int, rating, lang, bundle, templateType string) ([]dtos.GiphyTemplate, error) {
	if limit <= 0 {
		limit = 25
	}
	if offset < 0 {
		offset = 0
	}

	ctx := context.Background()
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.CommitTxn(tx)

	templates, err := s.repo.SearchTemplates(ctx, tx, strings.TrimSpace(query), limit, offset)
	if err != nil {
		return nil, err
	}
	return toGiphyTemplates(templates), nil
}

// Trending returns a random sample of the active catalog each call, standing
// in for Giphy's popularity-ranked trending feed - this catalog has no
// engagement signal to rank by, so a fresh random assortment keeps the
// "Trending" tab visually varied instead of returning a static list.
// rating/bundle have no equivalent and are accepted-but-ignored.
func (s *templatesService) Trending(limit, offset int, rating, bundle, templateType string) ([]dtos.GiphyTemplate, error) {
	if limit <= 0 {
		limit = 25
	}

	ctx := context.Background()
	tx, err := s.txnManager.StartTxn()
	if err != nil {
		return nil, err
	}
	defer s.txnManager.CommitTxn(tx)

	templates, err := s.repo.RandomTemplates(ctx, tx, "", limit)
	if err != nil {
		return nil, err
	}
	return toGiphyTemplates(templates), nil
}

func toGiphyTemplates(templates []entities.MemeTemplate) []dtos.GiphyTemplate {
	out := make([]dtos.GiphyTemplate, 0, len(templates))
	for _, t := range templates {
		out = append(out, dtos.GiphyTemplate{ID: t.ID, URL: t.GCSURL, Type: t.Genre})
	}
	return out
}
