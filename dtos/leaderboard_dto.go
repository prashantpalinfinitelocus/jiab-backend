package dtos

type WinnerRewardDTO struct {
	ImageURL    string `json:"image_url"`
	Description string `json:"description"`
}

type LeaderboardEntryDTO struct {
	UserID       string          `json:"user_id"`
	Mobile       string          `json:"mobile"`
	Avatar       string          `json:"avatar"`
	Coins        int             `json:"coins"`
	UserRank     int             `json:"user_rank"`
	WinnerReward WinnerRewardDTO `json:"winner_reward"`
}

type LeaderboardResponseDTO struct {
	TopUsers []LeaderboardEntryDTO `json:"top_users"`
	MyRank   LeaderboardEntryDTO   `json:"my_rank"`
}
