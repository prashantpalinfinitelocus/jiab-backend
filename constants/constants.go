package constants

import "time"

const (
	DEFAULT_LIMIT                          = 15
	GET_SUPRISE_ME_JOKE_ENDPOINT           = "/production/jokes"
	GET_SCROLL_AND_LOL_JOKE_ENDPOINT       = "/production/collabs"
	CHANGE_UGC_STATUS_ENDPOINT             = "%v/status"
	GLUEDIN_TEXT_THUMBNAIL_LINK            = "UGCImages/UGC_detail_thumb.png"
	UGC_CONTENT_FOLDER                     = "Sprite-UGC"
	PROGRESSIVE_PROFILING_EXPORT_FOLDER    = "ProgressiveProfilingExports"
	MEME_CONTENT_FOLDER                    = "Sprite-Meme"
	MEME_LOGO_URL                          = "https://drive.google.com/uc?export=download&id=1yuaN55h-ncMTxtlduymIOThhRRi3T9De"
	MEME_BACKDROP_URL                      = "https://drive.google.com/uc?export=download&id=1YsQVvYR5X95IucSlvxPTPvF98nVEU1Ar"
	MEME_TEXT_MAX_LENGTH                   = 40
	DSR_EXPORT_FOLDER                      = "DSRExports"
	COIN_COUNT_INCREMENT_VALUE             = 1
	REWARD_REFERRER_ON_VERIFY              = 5
	REWARD_REFEREE_ON_VERIFY               = 1
	REWARD_PROFILE_COMPLETION              = 10
	REWARD_UNIQUE_CODE                     = 5
	DEFAULT_TOPIC                          = "notifications"
	WEB_PLATFORM                           = 1
	WA_PLATFORM                            = 2
	MAX_UGC_CONTENT_COUNT                  = 5
	MAX_JOKE_COUNT                         = 3
	MISSING_GENRE                          = "Default"
	WA_FORMAT_LOW_RES                      = "512"
	WA_FORMAT_HIGH_RES                     = "1024"
	CONTEST_GCP_PREFIX                     = "ContestAssets/"
	SCROLL_AND_MAX_COIN_COUNT              = 15
	DEFAULT_LANGUAGE_ID                    = 1
	MIXCODE_VALIDATE_CODE_ENDPOINT         = "/pcservices/rest/v7/pincodes/"
	MIXCODE_INVALID_LIMIT                  = 5
	MIXCODE_REDEEM_LIMIT                   = 5
	MIXCODE_REDEEM_COIN_COUNT              = 20
	MAX_SCROLL_AND_LOL_JOKE_COUNT          = 15
	MAX_SPICY_TRAY_JOKE_COUNT              = 15
	UGC_STATUS_APPROVED                    = 2
	UGC_STATUS_PENDING                     = 1
	UGC_STATUS_PENDING_STRING              = "Pending"
	UGC_STATUS_ACCEPTED_STRING             = "Accepted"
	UGC_STATUS_REJECTED_STRING             = "Rejected"
	UGC_STATUS_INACTIVE                    = "INACTIVE"
	UGC_STATUS_ACCEPTED                    = 2
	UGC_STATUS_REJECTED                    = 4
	UGC_STATUS_PLAGIARIZED                 = 3
	UGC_STATUS_PLAGIARIZED_STRING          = "Plagiarized"
	SCROLL_AND_LOL_LABEL                   = "-creator-collab"
	SPICY_TRAY_LABEL                       = "spicy_campaign"
	OTP_MAX_ATTEMPTS                       = 5
	OTP_RATE_LIMIT_DURATION_MINUTES        = 5
	OTP_EXPIRATION_MINUTES                 = 5
	VERIFY_OTP_RATE_LIMIT_DURATION_MINUTES = 5
	VERIFY_OTP_MAX_ATTEMPTS                = 5
	MIN_AGE_REQUIRED                       = 18
	MAX_AGE_ALLOWED                        = 150
	REFRESH_RING_LABEL                     = "RR"
	JOKE_FORMAT_TEXT                       = 1
	WA_TEMPLATE_ENDPOINT                   = "/whatsapp/1/message/template"
	WA_FROM_NUMBER                         = "919650877066"
	WA_PJ_CHALLENGE_WINNER_IMAGE_URL       = "https://017526-content-prod.s3.ap-south-1.amazonaws.com/Assets/Images/PJ-Master.png"
	WA_UGC_REJECTED_TEMPLATE_NAME          = "new_ugc_jokemoderationmessage_rejected3"
	WA_UGC_PLAGIARIZED_TEMPLATE_NAME       = "new1_ugc_jokemoderationmessage_rejected2"
	WA_UGC_SUCCESS                         = "ugc_pjchallengewinnermessage_media"
	CDP_EVENT_ENDPOINT                     = "/v2/datamart/collect"
	HINGLISH_LABEL                         = "Hinglish"
	HINDI_LABEL                            = "hindi"

	PROFILING_QUESTION_MIN_NUMBER     = 1
	PROFILING_QUESTION_MAX_NUMBER     = 3
	PROFILING_QUESTION_MODAL_COOLDOWN = 48 * time.Hour

	CDP_BRAND_NAME_SPRITE = "Sprite"

	// question_master IDs verified against production (tccc-sfa-prod-sql,
	// question_master table) on 2026-08-18. The previous values (3, 4) were
	// inherited from a one-time MSSQL migration and never matched the real
	// active rows, which silently emptied the CDP backfill export.
	PROGRESSIVE_PROFILING_BOTTLE_COUNT_QUESTION_ID = 16
	PROGRESSIVE_PROFILING_DAYPART_QUESTION_ID      = 17
	PROGRESSIVE_PROFILING_JOKE_GENRE_QUESTION_ID   = 18
)

// ProgressiveProfilingCDPEventPrefixes maps a progressive-profiling
// question_master_id to the CDP event_sub_type prefix used for that
// question. Only question IDs present here are included in the CDP backfill
// export (services.ProgressiveProfilingExportService) — anything else is
// silently skipped, so add a row here (and verify the ID against production)
// before a new profiling question can be backfilled.
var ProgressiveProfilingCDPEventPrefixes = map[int]string{
	PROGRESSIVE_PROFILING_DAYPART_QUESTION_ID:      "ConsumptionDaypartPreference",
	PROGRESSIVE_PROFILING_BOTTLE_COUNT_QUESTION_ID: "ConsumptionCount",
	PROGRESSIVE_PROFILING_JOKE_GENRE_QUESTION_ID:   "ComedyGenrePreference",
}

// Notification Constants
const (
	NOTIFICATION_SIGNUP                  = 4
	NOTIFICATION_WIN_COIN                = 2
	NOTIFICATION_SIGNUP_REFERRAL         = 3
	NOTIFICATION_USER_INCOMPLETE_PROFILE = 9
	NOTIFICATION_REFERRAL_SUCCESSFUL     = 1
	NOTIFICATION_UGC_REPORTED            = 13
	NOTIFICATION_UGC_REMOVED             = 10
	// The notification_types rows at ids 7/8 hold copy swapped from what
	// these names imply (row 7 is the rejection copy, row 8 is the
	// acceptance copy) — corrected here to match the actual row content
	// rather than the DB.
	NOTIFICATION_UGC_APPROVED            = 8
	NOTIFICATION_UGC_REJECTED            = 7
	NOTIFICATION_UGC_DUPLICATE           = 14
	NOTIFICATION_COMPLETE_PROFILE        = 15
	NOTIFICATION_CONTEST_ANNOUNCEMENT    = 16
)

const (
	NOTIFICATION_CATEGORY = "coca_cola_notifications"
)

// Firebase/FCM error substrings (lowercase) that indicate a token is permanently
// dead and should be invalidated rather than retried.
var FIREBASE_INVALID_TOKEN_ERROR_SUBSTRINGS = []string{
	"unregistered",                       // "Device unregistered.", "messaging/registration-token-not-registered"
	"notregistered",                      // Firebase Admin SDK literal "NotRegistered"
	"not a valid fcm registration token", // "The registration token is not a valid FCM registration token"
	"invalid-registration-token",         // "messaging/invalid-registration-token"
	"invalidargument",                    // FCM InvalidArgument on malformed token
	"senderid mismatch",                  // token registered to a different Firebase sender/project
	"requested entity was not found",     // token/app instance no longer exists
}

// Profile-completion notification cadence
const (
	PROFILE_COMPLETION_REMINDER_COOLDOWN = 5 * 24 * time.Hour // "Complete your profile" nudge: max once every 5 days per user
	PROFILE_COMPLETION_CAMPAIGN_COOLDOWN = 7 * 24 * time.Hour // Ops bulk campaign w/ coin reward: max once per week per user
)

// Contest announcement broadcast types (for dedup by contest_id + type)
const (
	CONTEST_ANNOUNCEMENT_TYPE_LAUNCH   = "launch"
	CONTEST_ANNOUNCEMENT_TYPE_REMINDER = "reminder"
)

// Bulk-campaign batching: campaigns page through users via keyset pagination
// (id > lastSeenID) rather than loading the whole population at once, and
// send within each batch with bounded concurrency.
const (
	CAMPAIGN_BATCH_SIZE       = 500
	CAMPAIGN_SEND_CONCURRENCY = 15
)

// IST is the timezone the daily notification scheduler runs against.
const IST_TIMEZONE = "Asia/Kolkata"

// SSO Configuration
const (
	SSO_TOKEN_LENGTH          = 64
	SSO_TOKEN_EXPIRY_MINUTES  = 5
	SSO_CREATED_BY            = "SSO"
	DEFAULT_FRONTEND_BASE_URL = "https://sprite-joke-in-a-bottle.coke2home.com"
)

// SSO Handler timeouts
const (
	HANDLER_SSO_INITIATE_TIMEOUT     = 10 * time.Second
	HANDLER_SSO_AUTHENTICATE_TIMEOUT = 30 * time.Second
)

const (
	TEMPLATE_TYPE_GIFS     = "gifs"
	TEMPLATE_TYPE_STICKERS = "stickers"
	TEMPLATE_TYPE_ALL      = "all"
)

// Meme Repository Constants
const (
	STATUS_WHERE_CLAUSE = "status = ?"
	ID_LESS_THAN_CLAUSE = "id < ?"
	ORDER_BY_CREATED_AT = "created_at DESC"
	ORDER_BY_VIEW_COUNT = "view_count DESC"
)

// CORS_ALLOWED_ORIGINS are the only origins the API sends back in
// Access-Control-Allow-Origin. Previously this was an unconditional wildcard
// ("*"), which a StackHawk DAST scan flagged as a Cross-Domain Misconfiguration
// (CWE-264) - any site could read responses from unauthenticated endpoints.
// Add a new frontend origin here deliberately, not by widening to "*" again.
var CORS_ALLOWED_ORIGINS = map[string]bool{
	"https://jiab-staging.coke2home.com":          true,
	"https://sprite-joke-in-a-bottle.coke2home.com": true,
}

// Reaction Types
const (
	REACTION_HAPPY   = "happy"
	REACTION_SAD     = "sad"
	REACTION_NEUTRAL = "neutral"
)

var LANGUAGE_MNEMONICS = map[string]string{
	"English":  "eng",
	"Hindi":    "hin",
	"Marathi":  "marathi",
	"Maithili": "maithili",
	"Bengali":  "bangla",
	"Tamil":    "tamil",
	"tulu":     "tullu",
	"Telugu":   "telugu",
	"Bhojpuri": "bhojpuri",
	"Oriya":    "oriya",
	"Kannada":  "kannada",
}
