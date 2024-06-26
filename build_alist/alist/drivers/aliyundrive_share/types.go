package aliyundrive_share

import (
	"time"
)

type ErrorResp struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ShareTokenResp struct {
	ShareToken string    `json:"share_token"`
	ExpireTime time.Time `json:"expire_time"`
	ExpiresIn  int       `json:"expires_in"`
}

type ListResp struct {
	Items      []File `json:"items"`
	NextMarker string `json:"next_marker"`
}

type File struct {
	FileId        string    `json:"file_id"`
	ShareId       string    `json:"share_id"`
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	Category      string    `json:"category"`
	FileExtension string    `json:"file_extension"`
	Thumbnail     string    `json:"thumbnail"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Size          int64     `json:"size"`
}

type OpenTokenResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Data         struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	} `json:"data"`
}

type ShareLinkResp struct {
	DownloadUrl string `json:"download_url"`
	Url         string `json:"url"`
	Thumbnail   string `json:"thumbnail"`
}

type BatchResult struct {
	Responses []struct {
		Body struct {
			DriveID string `json:"drive_id"`
			FileID  string `json:"file_id"`
		} `json:"body"`
		Status int64 `json:"status"`
	} `json:"responses"`
}
