package aliyundrive_share

import (
	"github.com/alist-org/alist/v3/internal/driver"
	"github.com/alist-org/alist/v3/internal/op"
)

type Addition struct {
	OrderBy        string `json:"order_by" type:"select" options:"name,size,updated_at,created_at"`
	OrderDirection string `json:"order_direction" type:"select" options:"ASC,DESC"`
	driver.RootID
	OauthTokenURL    string `json:"oauth_token_url" default:"https://api.nn.ci/alist/ali_open/token"`
	ClientID         string `json:"client_id" required:"false" help:"Keep it empty if you don't have one"`
	ClientSecret     string `json:"client_secret" required:"false" help:"Keep it empty if you don't have one"`
	Shares           string `json:"shares" required:"true" type:"text"`
	RefreshToken     string `json:"refresh_token" required:"true"`
	OpenRefreshToken string `json:"open_refresh_token" required:"true"`
}

var config = driver.Config{
	Name:        "AliyundriveShare",
	LocalSort:   false,
	OnlyProxy:   false,
	NoUpload:    true,
	DefaultRoot: "root",
}

func init() {
	op.RegisterDriver(func() driver.Driver {
		return &AliyundriveShare{}
	})
}
