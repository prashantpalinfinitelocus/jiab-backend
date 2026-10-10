package entities

import "time"

type UserVideo struct {
	ID             int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	AssetID        string     `json:"asset_id" gorm:"not null"`
	UserID         string     `json:"user_id" gorm:"not null;index"`
	IsComplete     bool       `json:"is_complete" gorm:"type:boolean;default:false;not null"`
	VideoWatchOn   time.Time  `json:"video_watch_on" gorm:"type:timestamp;not null"`
	IsCoinEarned   bool       `json:"is_coin_earned" gorm:"type:boolean;default:false;not null"`
	CoineEarned    int        `json:"coine_earned" gorm:"type:integer;default:0;not null"`
	IsActive       bool       `json:"is_active" gorm:"type:boolean;default:true;not null"`
	IsDeleted      bool       `json:"is_deleted" gorm:"type:boolean;default:false;not null"`
	CreatedOn      time.Time  `json:"created_on" gorm:"type:timestamp;default:CURRENT_TIMESTAMP;not null"`
	LastModifiedOn *time.Time `json:"last_modified_on" gorm:"type:timestamp;default:CURRENT_TIMESTAMP"`
	User           User       `gorm:"foreignKey:UserID;references:ID"`
	CreatedBy      string     `json:"created_by" gorm:"type:varchar(255);not null"`
}

func (UserVideo) TableName() string {
	return "user_video"
}

type JokeType string

const (
	SupriseMeJoke    JokeType = "suprise_me"
	ScrollAndLolJoke JokeType = "scroll_and_lol"
	RefreshRingJoke  JokeType = "refresh_ring"
	SpicyTrayJoke    JokeType = "spicy_tray"
)

type JokeConsumptionByUser struct {
	ID               int        `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID           string     `json:"user_id" gorm:"type:varchar(450);not null;index"`
	JokeID           string     `json:"joke_id" gorm:"type:uuid;not null"`
	JokeDetailID     int        `json:"joke_detail_id" gorm:"not null"`
	ArtistID         string     `json:"artist_id" gorm:"type:uuid;not null"`
	JokeGeneratedAt  time.Time  `json:"joke_generated_at" gorm:"type:timestamp;not null"`
	Platform         int        `json:"platform" gorm:"not null"`
	SimilarJoke      int        `json:"similar_joke" gorm:"not null"`
	SelectedGenre    int        `json:"selected_genre" gorm:"not null"`
	SelectedLanguage int        `json:"selected_language" gorm:"not null"`
	IsActive         bool       `json:"is_active" gorm:"type:boolean;default:false;not null"`
	IsDeleted        bool       `json:"is_deleted" gorm:"type:boolean;default:false;not null"`
	CreatedOn        time.Time  `json:"created_on" gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`
	LastModifiedOn   *time.Time `json:"last_modified_on" gorm:"type:timestamp;default:CURRENT_TIMESTAMP"`
	CreatedBy        string     `json:"created_by" gorm:"type:varchar(255);not null"`
	JokeType         JokeType   `json:"joke_type" gorm:"joke_type_enum;not null"`

	User     User      `gorm:"foreignKey:UserID;references:ID"`
	Language Language  `gorm:"foreignKey:SelectedLanguage;references:ID"`
	Genre    JokeGenre `gorm:"foreignKey:SelectedGenre;references:ID"`
}

func (JokeConsumptionByUser) TableName() string {
	return "joke_consumption_by_user"
}

type JokeDetailsRecieved struct {
	ID              int        `json:"id" gorm:"primaryKey;autoIncrement"`
	JokeTitle       string     `json:"joke_title" gorm:"type:text;not null"`
	JokeContentType string     `json:"joke_content_type" gorm:"type:text;not null"`
	JokeCreator     string     `json:"joke_creator" gorm:"type:text;not null"`
	JokeLanguage    string     `json:"joke_language" gorm:"type:text;not null"`
	JokeGenre       string     `json:"joke_genre" gorm:"type:text;not null"`
	JokeDescription string     `json:"joke_description" gorm:"type:text;not null"`
	IsActive        bool       `json:"is_active" gorm:"type:boolean;not null;default:true"`
	IsDeleted       bool       `json:"is_deleted" gorm:"type:boolean;not null;default:false"`
	CreatedOn       time.Time  `json:"created_on" gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`
	LastModifiedOn  *time.Time `json:"last_modified_on" gorm:"type:timestamp;default:CURRENT_TIMESTAMP"`
	CreatedBy       string     `json:"created_by" gorm:"type:varchar(255);not null"`
}

func (JokeDetailsRecieved) TableName() string {
	return "joke_details_received"
}

type UGCMaster struct {
	ID             int        `json:"id" gorm:"primaryKey;autoIncrement"`
	LanguageID     int        `json:"language_id" gorm:"not null"`
	FormatID       int        `json:"format" gorm:"column:format;not null"`
	Title          string     `json:"title" gorm:"type:text;not null"`
	JokeGenreID    *int       `json:"joke_genre_id"`
	UserID         string     `json:"user_id" gorm:"type:varchar(40);not null"`
	AssetID        string     `json:"asset_id" gorm:"type:uuid;not null"`
	ContentURL     string     `json:"content_url" gorm:"type:text;not null"`
	FileName       string     `json:"file_name" gorm:"type:text;not null"`
	GluedinFill    bool       `json:"gluedin_fill" gorm:"type:boolean;not null"`
	StatusID       int        `json:"status" gorm:"column:status;not null"`
	PlatformID     int        `json:"platform_id" gorm:"not null"`
	IsActive       bool       `json:"is_active" gorm:"type:boolean;not null"`
	IsDeleted      bool       `json:"is_deleted" gorm:"type:boolean;not null"`
	CreatedBy      string     `json:"created_by" gorm:"not null"`
	CreatedOn      time.Time  `json:"created_on" gorm:"type:timestamp;not null"`
	LastModifiedBy *string    `json:"last_modified_by" gorm:"type:text"`
	LastModifiedOn *time.Time `json:"last_modified_on" gorm:"type:timestamp"`

	User     User            `gorm:"foreignKey:UserID;references:ID"`
	Language Language        `gorm:"foreignKey:LanguageID;references:ID"`
	Format   JokeFormat      `gorm:"foreignKey:FormatID;references:ID"`
	Genre    JokeGenre       `gorm:"foreignKey:JokeGenreID;references:ID"`
	Status   UGCStatusLookup `gorm:"foreignKey:StatusID;references:StatusID"`
}

func (UGCMaster) TableName() string {
	return "ugc_master"
}

type UGCStatusLookup struct {
	StatusID int    `json:"status_id" gorm:"primaryKey;autoIncrement"`
	Status   string `json:"status" gorm:"type:text;not null"`
}

func (UGCStatusLookup) TableName() string {
	return "ugc_status_lookup"
}
