package aliyundrive_share

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/alist-org/alist/v3/drivers/base"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/op"
	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/elliotchance/pie/v2"
	"github.com/go-resty/resty/v2"
	"github.com/mozillazg/go-pinyin"
	log "github.com/sirupsen/logrus"
)

func fileToObj(f File) *model.ObjThumb {
	var id string
	if strings.Contains(f.FileId, "++") {
		id = f.FileId
	} else {
		if f.Type == "file" {
			id = f.ShareId + "/folder/" + f.FileId + "/folder/" + f.Category
		} else {
			id = f.ShareId + "/folder/" + f.FileId
		}
	}
	return &model.ObjThumb{
		Object: model.Object{
			ID:       id,
			Name:     f.Name,
			Size:     f.Size,
			Modified: f.UpdatedAt,
			Ctime:    f.CreatedAt,
			IsFolder: f.Type == "folder",
		},
		Thumbnail: model.Thumbnail{Thumbnail: f.Thumbnail},
	}
}

func (d *AliyundriveShare) setPathMap() {
	d.PathMap = make(map[string][]string, 0)
	for _, share := range strings.Split(d.Shares, "\n") {
		share = strings.TrimSpace(share)
		if share != "" {
			str := strings.Split(share, "https://")
			id := strings.Split(str[1], "/s/")[1]
			if !strings.Contains(id, "/folder/") {
				id += "/folder/root"
			}
			if strings.TrimSpace(str[0]) == "" {
				d.PathMap["root"] = append(d.PathMap["root"], id)
			} else {
				paths := strings.Split(str[0], "/")
				for index, path := range paths {
					if index == 0 {
						if !pie.Contains(d.PathMap["root"], path) {
							d.PathMap["root"] = append(d.PathMap["root"], path)
						}
						if len(paths) > 1 && !pie.Contains(d.PathMap[path], paths[index+1]) {
							d.PathMap[path] = append(d.PathMap[path], paths[index+1])
						}
					}
					if index == len(paths)-1 {
						d.PathMap[str[0]] = append(d.PathMap[str[0]], id)
					} else {
						k := strings.Join(paths[:index+1], "/")
						if !pie.Contains(d.PathMap[k], paths[index+1]) {
							d.PathMap[k] = append(d.PathMap[k], paths[index+1])
						}
					}
				}
			}
		}
	}
}

func (d *AliyundriveShare) refreshToken() error {
	var e ErrorResp
	var resp base.TokenResp
	response, err := base.RestyClient.R().
		ForceContentType("application/json").
		SetBody(base.Json{"grant_type": "refresh_token", "refresh_token": d.RefreshToken}).SetResult(&resp).SetError(&e).
		Post("https://auth.aliyundrive.com/v2/account/token")
	if err != nil {
		log.Errorf("refreshToken response: %s", response.String())
		return err
	}
	if 200 != response.StatusCode() {
		return fmt.Errorf("failed to get refreshToken: %s:%s", e.Code, e.Message)
	}
	log.Infof("refreshToken exchange")
	d.RefreshToken, d.AccessToken = resp.RefreshToken, resp.AccessToken
	op.MustSaveDriverStorage(d)
	return nil
}

func (d *AliyundriveShare) openRefreshToken() error {
	url := "https://openapi.alipan.com/oauth/access_token"
	if d.OauthTokenURL != "" {
		url = d.OauthTokenURL
	}
	var e ErrorResp
	var resp base.TokenResp
	response, err := base.RestyClient.R().
		ForceContentType("application/json").
		SetBody(base.Json{
			"client_id":     d.ClientID,
			"client_secret": d.ClientSecret,
			"grant_type":    "refresh_token",
			"refresh_token": d.OpenRefreshToken,
		}).SetResult(&resp).SetError(&e).
		Post(url)
	if err != nil {
		log.Errorf("openRefreshToken response: %s", response.String())
		return err
	}
	if 200 != response.StatusCode() {
		return fmt.Errorf("failed to get openRefreshToken: %s:%s", e.Code, e.Message)
	}
	log.Infof("openRefreshToken exchange")
	d.OpenRefreshToken, d.OpenAccessToken = resp.RefreshToken, resp.AccessToken
	op.MustSaveDriverStorage(d)
	return nil
}

func (d *AliyundriveShare) getShareToken(shareId string) error {
	var e ErrorResp
	var resp ShareTokenResp
	response, err := base.RestyClient.R().
		ForceContentType("application/json").
		SetBody(base.Json{"share_id": shareId, "share_pwd": ""}).SetResult(&resp).SetError(&e).
		Post("https://api.aliyundrive.com/v2/share_link/get_share_token")
	if err != nil {
		log.Errorf("shareToken response: %s", response.String())
		return err
	}
	if 200 != response.StatusCode() {
		return fmt.Errorf("failed to get "+shareId+" shareToken: %s:%s", e.Code, e.Message)
	}
	log.Infof("%s: ShareToken exchange", shareId)
	d.ShareToken[shareId] = resp.ShareToken
	return nil
}

func (d *AliyundriveShare) request(shareId string, url string, method string, callback base.ReqCallback) ([]byte, error) {
	if d.AccessToken == "" {
		err := d.refreshToken()
		if err != nil {
			return nil, err
		}
	}
	var e ErrorResp
	req := base.RestyClient.R().SetError(&e).
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", "Bearer "+d.AccessToken)
	if shareId != "" {
		if d.ShareToken[shareId] == "" {
			err := d.getShareToken(shareId)
			if err != nil {
				return nil, err
			}
		}
		req.SetHeader("X-Share-Token", d.ShareToken[shareId])
	}
	if callback != nil {
		callback(req)
	} else {
		req.SetBody("{}")
	}
	response, err := req.Execute(method, url)
	if err != nil {
		log.Errorf("%s response: %s", url, response.String())
		return nil, err
	}
	if response.StatusCode() != 200 {
		if response.StatusCode() == 401 {
			err = d.refreshToken()
			if err != nil {
				return nil, err
			}
			if shareId != "" {
				err = d.getShareToken(shareId)
				if err != nil {
					return nil, err
				}
			}
			return d.request(shareId, url, method, callback)
		} else {
			return nil, fmt.Errorf("%s:%s", e.Code, e.Message)
		}
	}
	return response.Body(), nil
}

func (d *AliyundriveShare) openRequest(uri, method string, callback base.ReqCallback) ([]byte, error) {
	if d.OpenAccessToken == "" {
		err := d.openRefreshToken()
		if err != nil {
			return nil, err
		}
	}
	var e ErrorResp
	req := base.RestyClient.R().SetError(&e).
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", "Bearer "+d.OpenAccessToken)
	if callback != nil {
		callback(req)
	}
	response, err := req.Execute(method, "https://openapi.alipan.com"+uri)
	if err != nil {
		log.Errorf("%s response: %s", uri, response.String())
		return nil, err
	}
	if response.StatusCode() != 200 {
		if response.StatusCode() == 401 || response.StatusCode() == 429 {
			if response.StatusCode() == 429 {
				time.Sleep(500 * time.Millisecond)
			} else {
				err = d.openRefreshToken()
				if err != nil {
					return nil, err
				}
			}
			return d.openRequest(uri, method, callback)
		} else {
			return nil, fmt.Errorf("%s:%s", e.Code, e.Message)
		}
	}
	return response.Body(), nil
}

func (d *AliyundriveShare) getFilesRequest(ctx context.Context, args map[string]string) ([]File, error) {
	split := strings.Split(args["id"], "/folder/")
	if len(split) != 2 {
		return nil, fmt.Errorf("ali_share fileId is err")
	}
	shareId, fileId := split[0], split[1]
	if d.ShareToken[shareId] == "" {
		err := d.getShareToken(shareId)
		if err != nil {
			return nil, err
		}
	}
	files := make([]File, 0)
	data := base.Json{
		"share_id":        shareId,
		"parent_file_id":  fileId,
		"limit":           200,
		"order_by":        d.OrderBy,
		"order_direction": d.OrderDirection,
		"marker":          "first",
	}
	arg := pinyin.NewArgs()
	arg.Style = pinyin.FIRST_LETTER
	regex := regexp.MustCompile("^[\\p{Han}]")
	for data["marker"] != "" {
		if data["marker"] == "first" {
			data["marker"] = ""
		}
		var e ErrorResp
		var resp ListResp
		response, err := base.RestyClient.R().
			ForceContentType("application/json").
			SetBody(data).SetResult(&resp).SetError(&e).
			SetHeader("X-Share-Token", d.ShareToken[shareId]).
			Post("https://api.aliyundrive.com/adrive/v3/file/list")
		if err != nil {
			log.Errorf("file list response: %s", response.String())
			return nil, err
		}
		if response.StatusCode() != 200 {
			if response.StatusCode() == 401 || response.StatusCode() == 429 {
				if response.StatusCode() == 429 {
					time.Sleep(500 * time.Millisecond)
				} else {
					err = d.getShareToken(shareId)
					if err != nil {
						return nil, err
					}
				}
				return d.getFilesRequest(ctx, args)
			} else {
				return nil, fmt.Errorf("%s:%s", e.Code, e.Message)
			}
		}
		for i, file := range resp.Items {
			if file.Type == "folder" {
				if regex.MatchString(file.Name) {
					pinyinStr := pinyin.Pinyin(string([]rune(file.Name)[:1]), arg)
					resp.Items[i].Name = strings.ToUpper(pinyinStr[0][0]) + " " + file.Name
					if strings.Contains(file.Name, "完结") || strings.Contains(file.Name, "欧美日韩") {
						resp.Items[i].Name = "0" + file.Name
					}
				} else {
					resp.Items[i].Name = strings.Replace(file.Name, "）", " ", 1)
				}
				if strings.HasPrefix(file.Name, "【") {
					resp.Items[i].Name = "0" + file.Name
				}
			}
			if file.Category == "video" {
				resp.Items[i].Name = file.UpdatedAt.Local().Format("2006-01-02") + "_" + file.Name
			}
		}
		data["marker"] = resp.NextMarker
		files = append(files, resp.Items...)
	}
	return files, nil
}

func (d *AliyundriveShare) getFiles(ctx context.Context, args map[string]string) ([]File, error) {
	if d.limitList == nil {
		return nil, fmt.Errorf("driver not init")
	}
	resp, err := d.limitList(ctx, args)
	if err != nil {
		if strings.Contains(err.Error(), "share_link") {
			return nil, fmt.Errorf("%s,%s", err.Error(), "https://www.aliyundrive.com/s/"+args["id"])
		}
		log.Errorf("file list err: %s", err)
		return d.getFilesRequest(ctx, args)
	}
	return resp, nil
}

func (d *AliyundriveShare) copyFile(shareId string, fileId string) (string, error) {
	data := `{"requests":[{"body":{"file_id":"` + fileId + `","share_id":"` + shareId + `","auto_rename":true,"to_parent_file_id":"` + d.RootID.GetRootId() + `","to_drive_id":"` + d.DriveId + `"},"headers":{"Content-Type":"application/json"},"id":"0","method":"POST","url":"/file/copy"}],"resource":"file"}`
	var result BatchResult
	_, err := d.request(shareId, "https://api.aliyundrive.com/adrive/v2/batch", http.MethodPost, func(req *resty.Request) {
		req.SetBody(data).SetResult(&result)
	})
	if err != nil {
		return "", err
	}
	return result.Responses[0].Body.FileID, nil
}

func (d *AliyundriveShare) getFileUrl(fileId string) (string, error) {
	resp, err := d.openRequest("/adrive/v1.0/openFile/getDownloadUrl", http.MethodPost, func(req *resty.Request) {
		req.SetBody(base.Json{
			"drive_id":   d.DriveId,
			"file_id":    fileId,
			"expire_sec": 900,
		})
	})
	_, _ = d.openRequest("/adrive/v1.0/openFile/delete", http.MethodPost, func(req *resty.Request) {
		req.SetBody(base.Json{
			"drive_id": d.DriveId,
			"file_id":  fileId,
		})
	})
	if err != nil {
		return "", err
	}
	return utils.Json.Get(resp, "url").ToString(), nil
}
