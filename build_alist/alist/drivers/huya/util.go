package huya

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/alist-org/alist/v3/drivers/base"
	"github.com/alist-org/alist/v3/internal/model"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

func isNumber(str string) bool {
	_, err := strconv.Atoi(str)
	return err == nil
}

func fileToObj(f File) *model.ObjThumb {
	return &model.ObjThumb{
		Object: model.Object{
			ID:       f.RoomId,
			Name:     f.Name,
			IsFolder: f.Type == "folder",
		},
		Thumbnail: model.Thumbnail{Thumbnail: f.Thumbnail},
	}
}

func (d *HuYa) roomIdToString(str string) string {
	roomId := strings.Split(str, "https://www.huya.com/")[1]
	if !isNumber(roomId) {
		doc, err := goquery.NewDocument("https://www.huya.com/" + roomId)
		if err != nil {
			log.Errorf("%s:%s", str, err)
		}
		id := doc.Find(".host-rid em").Text()
		if id != "" {
			roomId = id
		}
	}
	return roomId
}

func (d *HuYa) MD5Hex(str string) string {
	hash := md5.New()
	hash.Write([]byte(str))
	return hex.EncodeToString(hash.Sum(nil))
}

func (d *HuYa) get(url string) ([]byte, error) {
	resp, err := base.RestyClient.R().Get(url)
	if err != nil || resp.StatusCode() != 200 {
		return nil, fmt.Errorf("%s:%s", url, err)
	}
	return resp.Body(), nil
}

func (d *HuYa) getStreamUrl(resUrl string, flg bool) ([]byte, []LiveTranscodingTaskList, error) {
	res, err := d.get(resUrl)
	if err != nil {
		log.Errorf("%s:%s", resUrl, err)
		return nil, nil, err
	}
	var streamUrl, urlParams string
	liveTranscodingTaskList := make([]LiveTranscodingTaskList, 0)
	gjson.GetBytes(res, "data.stream.baseSteamInfoList").ForEach(func(_, steamInfo gjson.Result) bool {
		liveTranscodingTaskList = append(liveTranscodingTaskList, LiveTranscodingTaskList{
			TemplateID: steamInfo.Get("sCdnType").String(),
			URL:        strings.ReplaceAll(steamInfo.Get("sFlvUrl").String(), "-game", "") + "/" + steamInfo.Get("sStreamName").String() + ".flv?",
		})
		if steamInfo.Get("sCdnType").String() == d.Node {
			antiCode := steamInfo.Get("newCFlvAntiCode").String()
			urlQuery, _ := url.ParseQuery(antiCode)

			uid := steamInfo.Get("lPresenterUid").Int()
			u := (uid<<8 | uid>>24) & -0x1
			wsTime := urlQuery.Get("wsTime")
			//ct := int64(2147483647)
			ct, _ := strconv.ParseInt(wsTime, 16, 64)
			seqId := uid + ct

			fmBase64Decode, _ := base64.StdEncoding.DecodeString(url.QueryEscape(urlQuery.Get("fm")))

			wsSecretPrefix := strings.Split(string(fmBase64Decode), "_")[0]
			wsSecretHash := d.MD5Hex(strings.Join([]string{strconv.FormatInt(seqId, 10), urlQuery.Get("ctype"), "100"}, "|"))
			sStreamName := steamInfo.Get("sStreamName").String()
			wsSecret := d.MD5Hex(strings.Join([]string{wsSecretPrefix, strconv.FormatInt(u, 10), sStreamName, wsSecretHash, wsTime}, "_"))

			params := url.Values{}
			params.Add("wsSecret", wsSecret)
			params.Add("wsTime", wsTime)
			params.Add("seqid", strconv.FormatInt(seqId, 10))
			params.Add("ctype", urlQuery.Get("ctype"))
			params.Add("fs", urlQuery.Get("fs"))
			params.Add("u", strconv.FormatInt(u, 10))
			params.Add("t", "100")
			params.Add("ver", "1")
			params.Add("uuid", strconv.FormatInt(ct, 10))
			params.Add("sdk_sid", strconv.FormatInt(ct, 10))
			params.Add("codec", "264")
			urlParams = params.Encode()

			sFlvUrl := strings.ReplaceAll(steamInfo.Get("sFlvUrl").String(), "http", "https")
			streamUrl = strings.ReplaceAll(sFlvUrl, "-game", "") + "/" + sStreamName + ".flv?" + urlParams
			return false
		}
		return true
	})
	for i, list := range liveTranscodingTaskList {
		liveTranscodingTaskList[i].URL = list.URL + urlParams
	}
	if flg {
		return nil, liveTranscodingTaskList, nil
	} else {
		return []byte(streamUrl), nil, nil
	}
}
