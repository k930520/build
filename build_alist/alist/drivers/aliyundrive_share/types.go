package aliyundrive_share

import (
	"time"
)

type ErrorResp struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type DriveID struct {
	BackupDriveID   string `json:"backup_drive_id"`
	ResourceDriveID string `json:"resource_drive_id"`
	DefaultDriveID  string `json:"default_drive_id"`
}

type File struct {
	FileId        string    `json:"file_id"`
	ShareId       string    `json:"share_id"`
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	FileExtension string    `json:"file_extension"`
	MimeExtension string    `json:"mime_extension"`
	Thumbnail     string    `json:"thumbnail"`
	Size          int64     `json:"size"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type ListResp struct {
	Items      []File `json:"items"`
	NextMarker string `json:"next_marker"`
}

type ShareTokenResp struct {
	ShareToken string    `json:"share_token"`
	ExpireTime time.Time `json:"expire_time"`
	ExpiresIn  int       `json:"expires_in"`
}

type ShareLinkResp struct {
	DownloadUrl string `json:"download_url"`
	Url         string `json:"url"`
	Thumbnail   string `json:"thumbnail"`
}

type OpenTokenResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Data         struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	} `json:"data"`
}

type OpenLinkResp struct {
	URL        string `json:"url"`
	StreamsURL any    `json:"streamsUrl"`
}

type BatchResult struct {
	Responses []struct {
		Body struct {
			Code    string `json:"code"`
			DriveID string `json:"drive_id"`
			FileID  string `json:"file_id"`
		} `json:"body"`
		Status int64 `json:"status"`
	} `json:"responses"`
}
