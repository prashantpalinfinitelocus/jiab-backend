package dtos

type MemeGeneratorRequest struct {
	TopText    string `form:"top_text" json:"top_text"`
	BottomText string `form:"bottom_text" json:"bottom_text"`
	// MemeURL is an alternative to the "meme" file field: when set, the
	// backend fetches the template server-side instead of requiring the
	// client to download it and re-upload the same bytes. Restricted to an
	// allowlist of trusted hosts (see utils.FetchRemoteAssetAsFileHeader) to
	// prevent SSRF.
	MemeURL string `form:"meme_url" json:"meme_url"`
}

type MemeGeneratorResponse struct {
	ImageURL string `json:"image_url"`
	Message  string `json:"message"`
}
