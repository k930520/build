package aliyundrive_share

import (
	"context"
	"fmt"
	"net/http"
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
	AccessToken     string
	OpenAccessToken string
	DriveId         string
	ShareToken      map[string]string
	PathMap         map[string][]string
	limitList       func(ctx context.Context, args map[string]string) ([]File, error)
	limitLink       func(ctx context.Context, file model.Obj) (*model.Link, error)
}

func (d *AliyundriveShare) Config() driver.Config {
	return config
}

func (d *AliyundriveShare) GetAddition() driver.Additional {
	return &d.Addition
}

func (d *AliyundriveShare) Init(ctx context.Context) error {
	res, err := d.request("", "https://user.aliyundrive.com/v2/user/get", http.MethodPost, nil)
	if err != nil {
		return err
	}
	d.DriveId = utils.Json.Get(res, "resource_drive_id").ToString()
	d.setPathMap()
	d.ShareToken = make(map[string]string, 0)
	d.limitList = rateg.LimitFnCtx(d.getFilesRequest, rateg.LimitFnOption{
		Limit:  4,
		Bucket: 1,
	})
	d.limitLink = rateg.LimitFnCtx(d.link, rateg.LimitFnOption{
		Limit:  1,
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
		tmp := make([]File, 0)
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
				resp, err := d.getFiles(ctx, map[string]string{"id": v, "path": args.ReqPath})
				if err != nil {
					return nil, err
				}
				tmp = append(tmp, resp...)
			}
		}
		files = append(files, tmp...)
	}
	if strings.Contains(dir.GetID(), "++") {
		tmp := make([]File, 0)
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
				resp, err := d.getFiles(ctx, map[string]string{"id": s, "path": args.ReqPath})
				if err != nil {
					return nil, err
				}
				tmp = append(tmp, resp...)
			}
		}
		files = append(files, tmp...)
	}
	if strings.Contains(dir.GetID(), "/folder/") {
		resp, err := d.getFiles(ctx, map[string]string{"id": dir.GetID(), "path": args.ReqPath})
		if err != nil {
			return nil, err
		}
		files = append(files, resp...)
	}
	sort.Slice(files, func(i, j int) bool {
		switch d.OrderBy {
		case "name":
			{
				if files[i].Name == files[j].Name {
					files[i].Name += "*"
				}
				c := natural.Less(files[i].Name, files[j].Name)
				if d.OrderDirection == "DESC" {
					return !c
				}
				return c
			}
		case "size":
			{
				if d.OrderDirection == "DESC" {
					return files[i].Size >= files[j].Size
				}
				return files[i].Size <= files[j].Size
			}
		case "modified":
			if d.OrderDirection == "DESC" {
				return files[i].UpdatedAt.After(files[j].UpdatedAt)
			}
			return files[i].UpdatedAt.Before(files[j].UpdatedAt)
		}
		return false
	})
	return utils.SliceConvert(files, func(src File) (model.Obj, error) {
		return fileToObj(src), nil
	})
}

func (d *AliyundriveShare) Link(ctx context.Context, file model.Obj, args model.LinkArgs) (*model.Link, error) {
	if d.limitLink == nil {
		return nil, fmt.Errorf("driver not init")
	}
	return d.limitLink(ctx, file)
}

func (d *AliyundriveShare) link(ctx context.Context, file model.Obj) (*model.Link, error) {
	split := strings.Split(file.GetID(), "/folder/")
	if len(split) != 3 {
		return nil, fmt.Errorf("ali_share fileId is err")
	}
	shareId, fileId, category := split[0], split[1], split[2]
	switch category {
	case "video":
		tmpFileId, err := d.copyFile(shareId, fileId)
		if err != nil {
			return nil, err
		}
		if tmpFileId == "" {
			return nil, fmt.Errorf("save file failed")
		}
		url, err := d.getFileUrl(tmpFileId)
		if err != nil {
			return nil, err
		}
		if url == "" {
			return nil, fmt.Errorf("get file url failed")
		}
		exp := 30 * time.Minute
		return &model.Link{
			URL:        url,
			Expiration: &exp,
		}, nil
	default:
		var resp ShareLinkResp
		_, err := d.request(shareId, "https://api.aliyundrive.com/v2/file/get_share_link_download_url", http.MethodPost, func(req *resty.Request) {
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
	if len(split) != 3 {
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
		url := "https://api.aliyundrive.com/v2/file/get_office_preview_url"
		_, err := d.request(shareId, url, http.MethodPost, func(req *resty.Request) {
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
