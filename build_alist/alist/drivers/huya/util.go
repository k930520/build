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

func containsString(slice []gjson.Result, str string) int {
	for index, s := range slice {
		if s.String() == str {
			return index
		}
	}
	return -1
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

func (d *HuYa) getAntiCode(steamInfo gjson.Result) string {
	antiCode := steamInfo.Get("newCFlvAntiCode").String()
	urlQuery, _ := url.ParseQuery(antiCode)

	wsTime := urlQuery.Get("wsTime")
	ct, _ := strconv.ParseInt(wsTime, 16, 64)
	//ct := int64(((time.Now().UnixMilli()%1e10)*1e3 + int64(rand.Float64()*1e3)) % 4294967295)

	uid := steamInfo.Get("lPresenterUid").Int()
	//u := (uid<<8 | uid>>24) & -0x1
	seqId := uid + ct

	fmBase64Decode, _ := base64.StdEncoding.DecodeString(url.QueryEscape(urlQuery.Get("fm")))

	wsSecretPrefix := strings.Split(string(fmBase64Decode), "_")[0]
	wsSecretHash := d.MD5Hex(strings.Join([]string{strconv.FormatInt(seqId, 10), urlQuery.Get("ctype"), "100"}, "|"))
	sStreamName := steamInfo.Get("sStreamName").String()
	wsSecret := d.MD5Hex(strings.Join([]string{wsSecretPrefix, strconv.FormatInt(uid, 10), sStreamName, wsSecretHash, wsTime}, "_"))

	params := url.Values{}
	params.Add("wsSecret", wsSecret)
	params.Add("wsTime", wsTime)
	params.Add("seqid", strconv.FormatInt(seqId, 10))
	params.Add("ctype", urlQuery.Get("ctype"))
	params.Add("fs", urlQuery.Get("fs"))
	params.Add("u", strconv.FormatInt(uid, 10))
	params.Add("t", "100")
	params.Add("ver", "1")
	params.Add("uuid", strconv.FormatInt(ct, 10))
	params.Add("sdk_sid", strconv.FormatInt(ct, 10))
	params.Add("codec", "264")
	urlParams := params.Encode()

	sFlvUrl := strings.ReplaceAll(steamInfo.Get("sFlvUrl").String(), "http", "https")

	return sFlvUrl + "/" + sStreamName + ".flv?" + urlParams
}

func (d *HuYa) get(url string) ([]byte, error) {
	resp, err := base.RestyClient.R().Get(url)
	if err != nil || resp.StatusCode() != 200 {
		return nil, fmt.Errorf("%s:%s", url, err)
	}
	return resp.Body(), nil
}

func (d *HuYa) getStreamUrl(resUrl string) ([]byte, error) {
	res, err := d.get(resUrl)
	if err != nil {
		log.Errorf("%s:%s", resUrl, err)
		return nil, err
	}
	var streamUrl string
	steamInfoList := gjson.GetBytes(res, "data.stream.baseSteamInfoList")
	index := containsString(steamInfoList.Get("..#.sCdnType").Array(), d.Node)
	if index == -1 {
		streamUrl = d.getAntiCode(steamInfoList.Array()[0])
	} else {
		streamUrl = d.getAntiCode(steamInfoList.Array()[index])
	}
	return []byte(streamUrl), nil
}
