#rm -rf AdGuardHome

#version=$(wget -qO- -t1 -T2 "https://api.github.com/repos/AdguardTeam/AdGuardHome/releases/latest" | grep "tag_name" | head -n 1 | awk -F ":" '{print $2}' | sed 's/\"//g;s/,//g;s/ //g')

#git clone -b $version https://github.com/AdguardTeam/AdGuardHome

sudo sed -i -e '/"slices"/a\ 	"strings"' AdGuardHome/internal/updater/check.go

sudo sed -i -e "/return dlURL, key, true/r build_adhome/adhome/internal/updater/check.txt" -e "//d" AdGuardHome/internal/updater/check.go

sudo sed -i -e '/return stringutil.FilterOut(conf.UpstreamDNS, IsCommentOrEmpty), nil/{s/.*/		upstreams = conf.UpstreamDNS } else {/;n;d;}' AdGuardHome/internal/dnsforward/config.go

sudo sed -i -e "/return stringutil.FilterOut(upstreams, IsCommentOrEmpty), nil/r build_adhome/adhome/internal/dnsforward/config_u.txt" -e "//d" AdGuardHome/internal/dnsforward/config.go

sudo sed -i -e "/type ServerConfig struct {/r build_adhome/adhome/internal/dnsforward/config.txt" -e "//d" AdGuardHome/internal/dnsforward/config.go

sudo sed -i -e "/if dctx.err = prx.Resolve(pctx); dctx.err != nil {/r build_adhome/adhome/internal/dnsforward/process.txt" -e "//d" AdGuardHome/internal/dnsforward/process.go

sudo sed -i -e "/uc, err = proxy.ParseUpstreamsConfig(\*req.Upstreams, opts)/r build_adhome/adhome/internal/dnsforward/http_s.txt" -e "//d" AdGuardHome/internal/dnsforward/http.go

sudo sed -i -e "/cv := newUpstreamConfigValidator(req.Upstreams, req.FallbackDNS, req.PrivateUpstreams, opts)/r build_adhome/adhome/internal/dnsforward/http_t.txt" -e "//d" AdGuardHome/internal/dnsforward/http.go

cd AdGuardHome

go mod tidy

sudo sed -i -e '/if withECS {/{s/.*/	c.itemsWithSubnet = c.items/;n;d;n;d;}' $GOMODCACHE/github.com/!adguard!team/dnsproxy@v0.75.0/proxy/cache.go

make CHANNEL='edge' GOOS='linux' GOARCH='arm' GOARM='7' OUT='./dist/AdGuardHome/AdGuardHome'

#tar -czvf dist/AdGuardHome_dist.tar.gz internal/*

upx -9 ./dist/AdGuardHome/AdGuardHome

tar -C "./dist" -c -f - "./AdGuardHome" | gzip -9 - > "dist/AdGuardHome_linux_armv7.tar.gz"
