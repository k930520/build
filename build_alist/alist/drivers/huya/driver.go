package huya

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"

	"github.com/PuerkitoBio/goquery"
	"github.com/Xhofe/rateg"
	"github.com/alist-org/alist/v3/internal/driver"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/pkg/utils"
)

type HuYa struct {
	model.Storage
	Addition
	limitLink func(ctx context.Context, file model.Obj) (*model.Link, error)
}

func (d *HuYa) Config() driver.Config {
	return config
}

func (d *HuYa) GetAddition() driver.Additional {
	return &d.Addition
}

func (d *HuYa) Init(ctx context.Context) error {
	if d.Node == "" {
		d.Node = "AL"
	}
	d.limitLink = rateg.LimitFnCtx(d.link, rateg.LimitFnOption{
		Limit:  1,
		Bucket: 1,
	})
	return nil
}

func (d *HuYa) Drop(ctx context.Context) error {
	return nil
}

func (d *HuYa) List(ctx context.Context, dir model.Obj, args model.ListArgs) ([]model.Obj, error) {
	files := make([]File, 0)
	if dir.GetName() == "root" {
		files = append(files,
			File{
				Name:   "赛事直播",
				RoomId: "https://www.huya.com/m",
				Type:   "folder",
			},
			File{
				Name:   "网游竞技",
				RoomId: "https://www.huya.com/g_ol",
				Type:   "folder",
			},
			File{
				Name:   "单机热游",
				RoomId: "https://www.huya.com/g_pc",
				Type:   "folder",
			},
			File{
				Name:   "娱乐天地",
				RoomId: "https://www.huya.com/g_yl",
				Type:   "folder",
			},
			File{
				Name:   "手游休闲",
				RoomId: "https://www.huya.com/g_sy",
				Type:   "folder",
			},
		)
	}
	if strings.Contains(dir.GetID(), "https://www.huya.com/") {
		doc, err := goquery.NewDocument(dir.GetID())
		if err != nil {
			log.Errorf("%s:%s", dir.GetName(), err)
			return nil, fmt.Errorf("%s:%s", dir.GetName(), err)
		}
		tmp := make([]File, 0)
		if strings.Contains(dir.GetID(), "g_") {
			doc.Find(".g-gameCard-item").Each(func(i int, s *goquery.Selection) {
				name, titleExist := s.Attr("title")
				roomId, hrefExist := s.Find(".g-gameCard-link").Attr("href")
				if titleExist && hrefExist {
					tmp = append(tmp, File{
						Name:   name,
						RoomId: roomId,
						Type:   "folder",
					})
				}
			})
		} else {
			if dir.GetID() != "https://www.huya.com/m" {
				dataStr := doc.Find("body script").FilterFunction(func(i int, s *goquery.Selection) bool {
					return strings.Contains(s.Text(), "CATE_LIBS_DATA")
				})
				reg := regexp.MustCompile("var CATE_LIBS_DATA = (.*?);")
				match := reg.FindStringSubmatch(dataStr.Text())
				if match != nil {
					gjson.Parse(match[1]).ForEach(func(_, item gjson.Result) bool {
						name := item.Get("name").String()
						tid := item.Get("tId").String()
						if tid != "0" {
							tmp = append(tmp, File{
								Name:   name,
								RoomId: "https://live.huya.com/liveHttpUI/getTmpLiveList?iPageSize=120&iTmpId=" + tid,
								Type:   "folder",
							})
						} else {
							item.Get("childs").ForEach(func(_, value gjson.Result) bool {
								if value.Get("name").String() == "全部" {
									tmp = append(tmp, File{
										Name:   name,
										RoomId: "https://live.huya.com/liveHttpUI/getTmpLiveList?iPageSize=120&iTmpId=" + value.Get("tId").String(),
										Type:   "folder",
									})
									return false
								}
								return true
							})
						}
						return true
					})
				}
			}
			doc.Find(".game-live-item").Each(func(i int, s *goquery.Selection) {
				name := s.Find(".txt .nick").Text()
				thumbnail, imgExist := s.Find("a img").Attr("data-original")
				find := s.Find(".title")
				roomId, hrefExist := find.Attr("href")
				if imgExist && hrefExist {
					tmp = append(tmp, File{
						Name:      name + ".flv",
						RoomId:    roomId,
						Type:      "file",
						Thumbnail: strings.Split(thumbnail, "?")[0],
					})
				}
			})
		}
		files = append(files, tmp...)
	}
	if strings.Contains(dir.GetID(), "https://live.huya.com/") {
		res, err := d.get(dir.GetID())
		if err != nil {
			log.Errorf("%s:%s", dir.GetID(), err)
			return nil, err
		}
		tmp := make([]File, 0)
		gjson.GetBytes(res, "vList").ForEach(func(_, item gjson.Result) bool {
			tmp = append(tmp, File{
				Name:   item.Get("sNick").String() + ".flv",
				RoomId: "https://www.huya.com/" + item.Get("lProfileRoom").String(),
				Type:   "file",
			})
			return true
		})
		files = append(files, tmp...)
	}
	return utils.SliceConvert(files, func(src File) (model.Obj, error) {
		return fileToObj(src), nil
	})
}

func (d *HuYa) Link(ctx context.Context, file model.Obj, args model.LinkArgs) (*model.Link, error) {
	if d.limitLink == nil {
		return nil, fmt.Errorf("driver not init")
	}
	return d.limitLink(ctx, file)
}

func (d *HuYa) link(ctx context.Context, file model.Obj) (*model.Link, error) {
	roomId := d.roomIdToString(file.GetID())
	url, err := d.getStreamUrl("https://mp.huya.com/cache.php?do=profileRoom&m=Live&roomid=" + roomId)
	if err != nil {
		log.Errorf("%s:%s", file.GetName(), err)
		return nil, fmt.Errorf("%s:%s", file.GetName(), err)
	}
	exp := 15 * time.Second
	return &model.Link{
		URL:        string(url),
		Expiration: &exp,
	}, nil
}

var _ driver.Driver = (*HuYa)(nil)
