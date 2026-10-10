package dtos

type PlagiarizedInfoRequest struct {
	AssetID     string `json:"assetId" binding:"required"`
	Information string `json:"information" binding:"required"`
}

type PlagiarizedInfoResponse struct {
	ID          int    `json:"id"`
	AssetID     string `json:"asset_id"`
	Information string `json:"information"`
	Message     string `json:"message"`
}

type ChangePlagiarizedInfoStatusRequest struct {
	PlagiarizedInfoID int `json:"id" binding:"required"`
}
