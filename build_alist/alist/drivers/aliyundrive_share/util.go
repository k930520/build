package aliyundrive_share

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/alist-org/alist/v3/drivers/base"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/op"
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
		id = f.ShareId + "/folder/" + f.FileId
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
		SetHeader("Content-Type", "application/json").
		SetBody(base.Json{"grant_type": "refresh_token", "refresh_token": d.RefreshToken}).SetResult(&resp).SetError(&e).
		Post("https://auth.aliyundrive.com/v2/account/token")
	if err != nil {
		log.Errorf("refreshToken err: %s", err.Error())
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
	var resp OpenTokenResp
	response, err := base.RestyClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(base.Json{
			"client_id":     d.ClientID,
			"client_secret": d.ClientSecret,
			"grant_type":    "refresh_token",
			"refresh_token": d.OpenRefreshToken,
		}).SetResult(&resp).SetError(&e).
		Post(url)
	if err != nil {
		log.Errorf("openRefreshToken err: %s", err.Error())
		return err
	}
	if 200 != response.StatusCode() {
		return fmt.Errorf("failed to get openRefreshToken: %s:%s", e.Code, e.Message)
	}
	log.Infof("openRefreshToken exchange")
	if resp.RefreshToken == "" {
		d.OpenRefreshToken, d.OpenAccessToken = resp.Data.RefreshToken, resp.Data.AccessToken
	} else {
		d.OpenRefreshToken, d.OpenAccessToken = resp.RefreshToken, resp.AccessToken
	}
	op.MustSaveDriverStorage(d)
	return nil
}

func (d *AliyundriveShare) getShareToken(shareId string) error {
	var e ErrorResp
	var resp ShareTokenResp
	var id, pwd string
	if strings.Contains(shareId, "?pwd=") {
		tmp := strings.Split(shareId, "?pwd=")
		id, pwd = tmp[0], tmp[1]
	} else {
		id = shareId
	}
	response, err := base.RestyClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(base.Json{"share_id": id, "share_pwd": pwd}).SetResult(&resp).SetError(&e).
		Post("https://api.aliyundrive.com/v2/share_link/get_share_token")
	if err != nil {
		log.Errorf("shareToken err: %s", err.Error())
		return err
	}
	if 200 != response.StatusCode() {
		return fmt.Errorf("failed to get "+shareId+" shareToken: %s:%s", e.Code, e.Message)
	}
	log.Infof("%s: ShareToken exchange", shareId)
	d.ShareToken[shareId] = resp.ShareToken
	return nil
}

func (d *AliyundriveShare) request(shareId string, flg bool, url string, method string, callback base.ReqCallback) (*resty.Response, error) {
	var e ErrorResp
	req := base.RestyClient.R().SetHeader("Content-Type", "application/json").SetError(&e)
	if shareId != "" {
		if d.ShareToken[shareId] == "" {
			err := d.getShareToken(shareId)
			if err != nil {
				return nil, err
			}
		}
		req.SetHeader("X-Share-Token", d.ShareToken[shareId])
	}
	if flg {
		if d.AccessToken == "" {
			err := d.refreshToken()
			if err != nil {
				return nil, err
			}
		}
		req.SetHeader("Authorization", "Bearer "+d.AccessToken)
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
		switch response.StatusCode() {
		case 429:
			time.Sleep(100 * time.Millisecond)
			return d.request(shareId, flg, url, method, callback)
		case 401:
			if shareId != "" && e.Code == "ShareLinkTokenInvalid" {
				err = d.getShareToken(shareId)
			}
			if flg && e.Code == "AccessTokenInvalid" {
				err = d.refreshToken()
			}
			if err != nil {
				return nil, err
			}
			return d.request(shareId, flg, url, method, callback)
		default:
			return nil, fmt.Errorf("%s:%s", e.Code, e.Message)
		}
	}
	if shareId != "" && response.StatusCode() == 200 && strings.Contains(string(response.Body()), "InvalidParameterNotMatch.ShareId") {
		err = d.getShareToken(shareId)
		if err != nil {
			return nil, err
		}
		return d.request(shareId, flg, url, method, callback)
	}
	return response, nil
}

func (d *AliyundriveShare) openRequest(uri, method string, callback base.ReqCallback) (*resty.Response, error) {
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
		switch response.StatusCode() {
		case 429:
			time.Sleep(500 * time.Millisecond)
			return d.openRequest(uri, method, callback)
		case 401:
			err = d.openRefreshToken()
			if err != nil {
				return nil, err
			}
			return d.openRequest(uri, method, callback)
		default:
			return nil, fmt.Errorf("%s:%s", e.Code, e.Message)
		}
	}
	return response, nil
}

func (d *AliyundriveShare) searchFiles(shareId string, parentFileId string, files []File) ([]File, error) {
	resultFiles := make([]File, 0)
	tmpFiles := make([]File, 0)
	for _, file := range files {
		tmpFiles, _ = d.limitList(base.Json{"id": shareId + "/folder/" + file.FileId, "flg": false})
		for _, tmp := range tmpFiles {
			if tmp.Type == "file" {
				resultFiles = append(resultFiles, tmpFiles...)
				break
			}
		}
		if len(resultFiles) != 0 {
			return resultFiles, nil
		}
	}
	if len(tmpFiles) == 0 {
		return tmpFiles, nil
	} else {
		return d.searchFiles(shareId, parentFileId, tmpFiles)
	}
}

func (d *AliyundriveShare) getFilesRequest(args base.Json) ([]File, error) {
	ids := strings.Split(args["id"].(string), "/folder/")
	if len(ids) != 2 {
		return nil, fmt.Errorf("ali_share fileId is err")
	}
	shareId, fileId := ids[0], ids[1]
	var orderBy string
	if d.OrderBy == "name" {
		orderBy = "name_enhanced"
	} else {
		orderBy = d.OrderBy
	}
	data := base.Json{
		"share_id":        strings.Split(shareId, "?pwd=")[0],
		"parent_file_id":  fileId,
		"limit":           200,
		"order_by":        orderBy,
		"order_direction": d.OrderDirection,
		"marker":          "first",
	}
	files := make([]File, 0)
	isContainsFile := false
	isUpperCaseFirst := false
	for data["marker"] != "" {
		if data["marker"] == "first" {
			data["marker"] = ""
		}
		var resp ListResp
		_, err := d.request(shareId, false, "https://api.aliyundrive.com/adrive/v2/file/list_by_share", http.MethodPost, func(req *resty.Request) { req.SetBody(data).SetResult(&resp) })
		if err != nil {
			return nil, err
		}
		for _, file := range resp.Items {
			file.ShareId = shareId
			if file.Type == "file" {
				isContainsFile = true
				file.Name = strings.Replace(file.Name, "."+file.FileExtension, "_", 1) + file.UpdatedAt.Local().Format("2006-01-02") + "." + file.MimeExtension
				files = append(files, file)
			} else {
				contains := true
				if !args["flg"].(bool) && d.Remark != "" {
					for _, str := range strings.Split(d.Remark, "/") {
						if strings.Contains(file.Name, str) {
							contains = false
							break
						}
					}
				}
				if contains {
					if !args["flg"].(bool) && !isUpperCaseFirst {
						isUpperCaseFirst = regexp.MustCompile(`^[A-Z]`).MatchString(file.Name)
					}
					file.Name = strings.ToUpper(strings.Trim(file.Name, "《【"))
					files = append(files, file)
				}
			}
		}
		data["marker"] = resp.NextMarker
	}
	if !args["flg"].(bool) && isUpperCaseFirst {
		arg := pinyin.NewArgs()
		arg.Style = pinyin.FIRST_LETTER
		for i, file := range files {
			if regexp.MustCompile(`^\p{Han}`).MatchString(file.Name) {
				pinyinStr := pinyin.Pinyin(string([]rune(file.Name)[:1]), arg)
				files[i].Name = strings.ToUpper(pinyinStr[0][0]) + file.Name
			} else {
				files[i].Name = strings.ReplaceAll(strings.ReplaceAll(file.Name, " ", ""), "）", "")
			}
		}
	}
	if args["flg"].(bool) && !isContainsFile && len(files) != 0 {
		return d.searchFiles(shareId, fileId, files)
	}
	return files, nil
}

func (d *AliyundriveShare) getFiles(args map[string]interface{}) ([]File, error) {
	if d.limitList == nil {
		return nil, fmt.Errorf("driver not init")
	}
	result, err := d.limitList(args)
	if err != nil {
		if strings.Contains(err.Error(), "share_link") {
			return nil, fmt.Errorf("%s,%s", err.Error(), "https://www.alipan.com/s/"+args["id"].(string))
		}
		log.Errorf("file list err: %s", err)
		return d.limitList(args)
	}
	return result, err
}

func (d *AliyundriveShare) copyFile(shareId string, fileId string) (string, error) {
	data := `{"requests":[{"body":{"file_id":"` + fileId + `","share_id":"` + strings.Split(shareId, "?pwd=")[0] + `","auto_rename":true,"to_parent_file_id":"` + d.RootID.GetRootId() + `","to_drive_id":"` + d.DriveId + `"},"headers":{"Content-Type":"application/json"},"id":"0","method":"POST","url":"/file/copy"}],"resource":"file"}`
	var result BatchResult
	_, err := d.request(shareId, true, "https://api.aliyundrive.com/adrive/v4/batch", http.MethodPost, func(req *resty.Request) { req.SetBody(data).SetResult(&result) })
	if err != nil {
		return "", err
	}
	return result.Responses[0].Body.FileID, nil
}

func (d *AliyundriveShare) getFileUrl(shareId string, fileId string) (string, error) {
	tmpFileId, err := d.copyFile(shareId, fileId)
	if err != nil || tmpFileId == "" {
		return "", fmt.Errorf("save file failed")
	}
	var result OpenLinkResp
	_, err = d.openRequest("/adrive/v1.0/openFile/getDownloadUrl", http.MethodPost, func(req *resty.Request) {
		req.SetBody(base.Json{"drive_id": d.DriveId, "file_id": tmpFileId, "expire_sec": 14400}).SetResult(&result)
	})
	data := `{"requests":[{"body":{"drive_id":"` + d.DriveId + `","file_id":"` + tmpFileId + `"},"headers":{"Content-Type":"application/json"},"id":"0","method":"POST","url":"/file/delete"}],"resource":"file"}`
	_, _ = d.request("", true, "https://api.aliyundrive.com/adrive/v4/batch", http.MethodPost, func(req *resty.Request) { req.SetBody(data) })
	if err != nil {
		return "", err
	}
	return result.URL, nil
}
