package huya

import (
	"github.com/alist-org/alist/v3/internal/driver"
	"github.com/alist-org/alist/v3/internal/op"
)

type Addition struct {
	Node string `json:"node" type:"select" options:"AL,AL13,TX,TX15,HW,HS,HY"`
	driver.RootID
}

var config = driver.Config{
	Name:        "HuYa",
	LocalSort:   false,
	OnlyProxy:   false,
	NoUpload:    true,
	DefaultRoot: "root",
}

func init() {
	op.RegisterDriver(func() driver.Driver {
		return &HuYa{}
	})
}
