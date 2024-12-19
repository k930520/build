package aliyundrive_share

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Xhofe/rateg"
	"github.com/alist-org/alist/v3/drivers/base"
	"github.com/alist-org/alist/v3/internal/driver"
	"github.com/alist-org/alist/v3/internal/errs"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/go-resty/resty/v2"
	"github.com/maruel/natural"
)

type AliyundriveShare struct {
	model.Storage
	Addition
	DriveId         string
	AccessToken     string
	OpenAccessToken string
	ShareToken      map[string]string
	PathMap         map[string][]string
	limitList       func(args base.Json) ([]File, error)
	limitLink       func(file model.Obj) (*model.Link, error)
}

func (d *AliyundriveShare) Config() driver.Config {
	return config
}

func (d *AliyundriveShare) GetAddition() driver.Additional {
	return &d.Addition
}

func (d *AliyundriveShare) Init(ctx context.Context) error {
	var result DriveID
	_, err := d.request("", true, "https://user.aliyundrive.com/v2/user/get", http.MethodPost, func(req *resty.Request) {
		req.SetResult(&result)
	})
	if err != nil {
		return err
	}
	d.DriveId = result.ResourceDriveID
	d.ShareToken = make(map[string]string, 0)
	d.setPathMap()
	d.limitList = rateg.LimitFn(d.getFilesRequest, rateg.LimitFnOption{
		Limit:  10,
		Bucket: 1,
	})
	d.limitLink = rateg.LimitFn(d.link, rateg.LimitFnOption{
		Limit:  2,
		Bucket: 1,
	})
	return nil
}

func (d *AliyundriveShare) Drop(ctx context.Context) error {
	return nil
}

func (d *AliyundriveShare) List(ctx context.Context, dir model.Obj, args model.ListArgs) ([]model.Obj, error) {
	files := make([]File, 0)
	if dir.GetName() == "root" {
		for _, v := range d.PathMap["root"] {
			if !strings.Contains(v, "/folder/") {
				file := File{
					FileId:    v + "++",
					Name:      v,
					UpdatedAt: dir.ModTime(),
					CreatedAt: dir.CreateTime(),
					Type:      "folder",
				}
				files = append(files, file)
			} else {
				resp, err := d.getFiles(base.Json{"id": v, "flg": false})
				if err != nil {
					return nil, err
				}
				files = append(files, resp...)
			}
		}
	}
	if strings.Contains(dir.GetID(), "++") {
		k := strings.Trim(dir.GetID(), "++")
		for _, s := range d.PathMap[k] {
			if !strings.Contains(s, "/folder/") {
				file := File{
					FileId:    k + "/" + s + "++",
					Name:      s,
					UpdatedAt: dir.ModTime(),
					CreatedAt: dir.CreateTime(),
					Type:      "folder",
				}
				files = append(files, file)
			} else {
				resp, err := d.getFiles(base.Json{"id": s, "flg": false})
				if err != nil {
					return nil, err
				}
				files = append(files, resp...)
			}
		}
	}
	if strings.Contains(dir.GetID(), "/folder/") {
		reqPath := strings.Split(args.ReqPath, "/")
		if regexp.MustCompile(`^[A-Z0-9]`).MatchString(reqPath[len(reqPath)-1]) || regexp.MustCompile(`^[A-Z0-9]`).MatchString(reqPath[len(reqPath)-2]) {
			resp, err := d.getFiles(base.Json{"id": dir.GetID(), "flg": true})
			if err != nil {
				return nil, err
			}
			files = append(files, resp...)
		} else {
			resp, err := d.getFiles(base.Json{"id": dir.GetID(), "flg": false})
			if err != nil {
				return nil, err
			}
			files = append(files, resp...)
		}
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].Name == files[j].Name {
			files[i].Name += "+"
		}
		c := natural.Less(files[i].Name, files[j].Name)
		if d.OrderDirection == "DESC" {
			return !c
		}
		return c
	})
	return utils.SliceConvert(files, func(src File) (model.Obj, error) {
		return fileToObj(src), nil
	})
}

func (d *AliyundriveShare) Link(ctx context.Context, file model.Obj, args model.LinkArgs) (*model.Link, error) {
	if d.limitLink == nil {
		return nil, fmt.Errorf("driver not init")
	}
	return d.limitLink(file)
}

func (d *AliyundriveShare) link(file model.Obj) (*model.Link, error) {
	split := strings.Split(file.GetID(), "/folder/")
	if len(split) != 2 {
		return nil, fmt.Errorf("ali_share fileId is err")
	}
	shareId, fileId := split[0], split[1]
	if file.GetSize()/1024 > 10*1024 {
		url, err := d.getFileUrl(shareId, fileId)
		if err != nil || url == "" {
			return nil, fmt.Errorf("get file url failed")
		}
		exp := 30 * time.Minute
		return &model.Link{
			URL:        url,
			Expiration: &exp,
		}, nil
	} else {
		var resp ShareLinkResp
		_, err := d.request(shareId, true, "https://api.aliyundrive.com/v2/file/get_share_link_download_url", http.MethodPost, func(req *resty.Request) {
			req.SetBody(base.Json{"share_id": shareId, "file_id": fileId, "expire_sec": 600}).SetResult(&resp)
		})
		if err != nil {
			return nil, err
		}
		url := resp.DownloadUrl
		if url == "" {
			return nil, fmt.Errorf("get file url failed")
		}
		exp := 600 * time.Second
		return &model.Link{
			URL:        url,
			Expiration: &exp,
		}, nil
	}
}

func (d *AliyundriveShare) Other(ctx context.Context, args model.OtherArgs) (interface{}, error) {
	split := strings.Split(args.Obj.GetID(), "/folder/")
	if len(split) != 2 {
		return nil, fmt.Errorf("ali_share fileId is err")
	}
	shareId, fileId := split[0], split[1]
	var resp base.Json
	switch args.Method {
	case "doc_preview":
		data := base.Json{
			"share_id": shareId,
			"file_id":  fileId,
		}
		_, err := d.request(shareId, false, "https://api.aliyundrive.com/v2/file/get_office_preview_url", http.MethodPost, func(req *resty.Request) {
			req.SetBody(data).SetResult(&resp)
		})
		if err != nil {
			return nil, err
		}
		return resp, nil
	case "video_preview":
		tmpFileId, err := d.copyFile(shareId, fileId)
		if err != nil || tmpFileId == "" {
			return nil, fmt.Errorf("save file failed")
		}
		data := base.Json{
			"drive_id":       d.DriveId,
			"file_id":        tmpFileId,
			"category":       "live_transcoding",
			"url_expire_sec": 14400,
		}
		_, err = d.openRequest("/adrive/v1.0/openFile/getVideoPreviewPlayInfo", http.MethodPost, func(req *resty.Request) {
			req.SetBody(data).SetResult(&resp)
		})
		if err != nil {
			return nil, err
		}
		return resp, nil
	default:
		return nil, errs.NotSupport
	}
}

var _ driver.Driver = (*AliyundriveShare)(nil)
