package dtos

import "time"

type GetComicCoinResponseDTO struct {
	ComicCoin int        `json:"comic_coin"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

type IncrementCoinCountRequestDTO struct {
	Genre    string  `form:"genre"`
	Language string  `form:"language"`
	Type     *string `form:"type"`
	AssetId  string  `form:"asset_id"`
}

type IncrementCoinCountResponseDTO struct {
	Message           string `json:"message"`
	ShowCoinAnimation bool   `json:"show_coin_animation"`
}
