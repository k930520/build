version=$(wget -qO- -t1 -T2 "https://api.github.com/repos/AdguardTeam/AdGuardHome/releases/latest" | grep "tag_name" | head -n 1 | awk -F ":" '{print $2}' | sed 's/\"//g;s/,//g;s/ //g')

git clone -b v0.107.55 https://github.com/AdguardTeam/AdGuardHome

sudo sed -i -e '/	"slices"/a\ 	"strings"' AdGuardHome/internal/updater/check.go

sudo sed -i -e "/		return dlURL, key, true/r build_adhome/adhome/internal/updater/check.txt" -e "//d" AdGuardHome/internal/updater/check.go

sudo sed -i -e '/                return stringutil.FilterOut(conf.UpstreamDNS, IsCommentOrEmpty), nil/{s/.*/             upstreams = conf.UpstreamDNS } else {/;n;d;}' AdGuardHome/internal/dnsforward/config.go

sudo sed -i -e "/	return stringutil.FilterOut(upstreams, IsCommentOrEmpty), nil/r build_adhome/adhome/internal/dnsforward/config_u.txt" -e "//d" AdGuardHome/internal/dnsforward/config.go

sudo sed -i -e "/type ServerConfig struct {/r build_adhome/adhome/internal/dnsforward/config.txt" -e "//d" AdGuardHome/internal/dnsforward/config.go

sudo sed -i -e "/	if dctx.err = prx.Resolve(pctx); dctx.err != nil {/r build_adhome/adhome/internal/dnsforward/process.txt" -e "//d" AdGuardHome/internal/dnsforward/process.go

sudo sed -i -e "/		uc, err = proxy.ParseUpstreamsConfig(\*req.Upstreams, opts)/r build_adhome/adhome/internal/dnsforward/http.txt" -e "//d" AdGuardHome/internal/dnsforward/http.go

cd AdGuardHome

mkdir dist

#make CHANNEL='release' GOOS='linux' GOARCH='arm' GOARM='7' OUT='./dist/AdGuardHome/AdGuardHome'

tar -czvf dist/AdGuardHome_dist.tar.gz internal/*

#upx -9 ./dist/AdGuardHome/AdGuardHome

#tar -C "./dist" -c -f - "./AdGuardHome" | gzip -9 - > "dist/AdGuardHome_linux_armv7.tar.gz"
