version=$(wget -qO- -t1 -T2 "https://api.github.com/repos/AdguardTeam/AdGuardHome/releases/latest" | grep "tag_name" | head -n 1 | awk -F ":" '{print $2}' | sed 's/\"//g;s/,//g;s/ //g')

git clone -b $version https://github.com/AdguardTeam/AdGuardHome

sudo sed -i -e "/		return dlURL, key, true/r build_adhome/adhome/internal/updater/check.txt" -e "//d" AdGuardHome/internal/updater/check.go

sudo sed -i -e "/type Server struct {/r build_adhome/adhome/internal/dnsforward/dnsforward.txt" -e "//d" AdGuardHome/internal/dnsforward/dnsforward.go

sudo sed -i -e "/	uc, err := newUpstreamConfig(upstreams, defaultDNS, &upstream.Options{/r build_adhome/adhome/internal/dnsforward/dnsforward_u.txt" -e "//d" AdGuardHome/internal/dnsforward/dnsforward.go

sudo sed -i -e "/	fallbacks := s.conf.FallbackDNS/r build_adhome/adhome/internal/dnsforward/dnsforward_f.txt" -e "//d" AdGuardHome/internal/dnsforward/dnsforward.go

sudo sed -i -e "/	if dctx.err = prx.Resolve(pctx); dctx.err != nil {/r build_adhome/adhome/internal/dnsforward/process.txt" -e "//d" AdGuardHome/internal/dnsforward/process.go

sudo sed -i -e "/		uc, err = proxy.ParseUpstreamsConfig(*req.Upstreams, opts)/r build_adhome/adhome/internal/dnsforward/http_u.txt" -e "//d" AdGuardHome/internal/dnsforward/http.go

sudo sed -i -e "/		uc, err = proxy.ParseUpstreamsConfig(*req.Fallbacks, opts)/r build_adhome/adhome/internal/dnsforward/http_f.txt" -e "//d" AdGuardHome/internal/dnsforward/http.go

cd AdGuardHome

make CHANNEL='release' GOOS='linux' GOARCH='arm' GOARM='7' OUT='./dist/AdGuardHome/AdGuardHome'

upx -9 ./dist/AdGuardHome/AdGuardHome

tar -C "./dist" -c -f - "./AdGuardHome" | gzip -9 - > "dist/AdGuardHome_linux_armv7.tar.gz"
