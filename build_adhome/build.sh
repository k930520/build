BuildAdGuardHome() {
sudo sed -i '/"slices"/a\ 	"strings"' AdGuardHome/internal/updater/check.go

sudo sed -i '/return dlURL, key, true/i\
	split := strings.Split(dlURL, "/")\
	dlURL = "http://github.home.local/https://github.com/k930520/build/releases/download/adhome/" + strings.Replace(split[len(split)-1], "AdGuardHome", "AdGuardHome_"+u.channel, 1)\
 ' AdGuardHome/internal/updater/check.go

sudo sed -i '/for _, ups := range conf.DomainReservedUpstreams {/i\
	for _, ups := range conf.IdentifierUpstreams {\
		insertListResults(ups, results, true)\
	}\
 ' AdGuardHome/internal/dnsforward/configvalidator.go

sudo sed -i '/func (s \*Server) Resolve(ctx context.Context, net, host string) (addr \[\]netip.Addr, err error) {/a\
	for _, u := range []uint16{dns.TypeA, dns.TypeAAAA} {\
		resVal, err := s.dnsFilter.CheckHost(strings.TrimSuffix(host,"."), u, &filtering.Settings{FilteringEnabled: true})\
		if err == nil {\
			if resVal.Reason.In(filtering.Rewritten) && resVal.CanonName != "" && len(resVal.IPList) == 0 {\
				host = dns.Fqdn(resVal.CanonName)\
				break\
			}\
			if resVal.Reason.In(filtering.Rewritten) {\
				addr = append(addr, resVal.IPList...)\
			}\
		}\
	}\
	if len(addr) > 0 {\
		return addr, nil\
	}\
 ' AdGuardHome/internal/dnsforward/dnsforward.go

sudo sed -i '/if s\.conf\.AAAADisabled && qt == dns\.TypeAAAA {/i\
	if err := netutil.ValidateDomainName(strings.TrimSuffix(q.Name, ".")); err != nil {\
		pctx.Res = s.NewMsgNODATA(pctx.Req)\
		return resultCodeFinish\
	}\
 ' AdGuardHome/internal/dnsforward/process.go

sudo sed -i '/if dctx\.err = prx\.Resolve(pctx); dctx\.err != nil {/i\
	prx.AAAADisabled = s.conf.AAAADisabled\
 /' AdGuardHome/internal/dnsforward/process.go

cd AdGuardHome

#tar -czvf ../build/AdGuardHome.tar.gz internal/*

go mod tidy

dnsproxy=$(ls /home/runner/go/pkg/mod/github.com/\!adguard\!team | grep dnsproxy)

echo dnsproxy is $dnsproxy

sudo sed -i '/if withECS {/d' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/cache.go

sudo sed -i -e '/c.itemsWithSubnet = createCache(size)/{s/.*/	c.itemsWithSubnet = c.items/;n;d;}' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/cache.go

sudo sed -i '/PreferIPv6 bool/a\	AAAADisabled bool' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/config.go

sudo sed -i '/"slices"/a\ 	"strings"' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/proxy.go

sudo sed -i '/getUpstreams := (\*UpstreamConfig).getUpstreamsForDomain/i\
	upstreams, found := p.UpstreamConfig.IdentifierUpstreams[d.Addr.Addr()]\
	if found {\
		return upstreams, false\
	}\
	ipWithoutZone := d.Addr.Addr().WithZone("")\
	for id, upstreams := range p.UpstreamConfig.IdentifierUpstreams {\
		switch id.(type) {\
		case netip.Prefix:\
			if id.(netip.Prefix).Contains(ipWithoutZone) {\
				return upstreams, false\
			}\
		}\
	}\
' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/proxy.go

sudo sed -i '/dctx.calcFlagsAndSize()/i\
	addr, found := p.UpstreamConfig.ToEDNSAddr[dctx.Addr.Addr()]\
	if found {\
		dctx.processECS(addr.AsSlice(), p.logger)\
	}\
	ipWithoutZone := dctx.Addr.Addr().WithZone("")\
	for key, addr := range p.UpstreamConfig.ToEDNSAddr {\
		switch key.(type) {\
		case netip.Prefix:\
			if key.(netip.Prefix).Contains(ipWithoutZone) {\
				dctx.processECS(addr.AsSlice(), p.logger)\
			}\
		}\
	}\
	for fqdn := dctx.Req.Question[0].Name; fqdn != ""; {\
		addr, found = p.UpstreamConfig.ToEDNSAddr["*."+fqdn]\
		if found && fqdn != dctx.Req.Question[0].Name {\
			dctx.processECS(addr.AsSlice(), p.logger)\
			break\
		}\
		addr, found = p.UpstreamConfig.ToEDNSAddr[fqdn]\
		if found {\
			dctx.processECS(addr.AsSlice(), p.logger)\
			break\
		}\
		_, fqdn, _ = strings.Cut(fqdn, ".")\
	}\
 ' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/proxy.go

sudo sed -i '/resp, u, err := p\.exchangeUpstreams(req, wrapped)/i\
	if p.AAAADisabled && req.Question[0].Qtype == dns.TypeA {\
		req.Question[0].Qtype = dns.TypeAAAA\
	}\
' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/proxy.go

sudo sed -i '/resp, u, err := p\.exchangeUpstreams(req, wrapped)/a\
	if p.AAAADisabled && req.Question[0].Qtype == dns.TypeAAAA {\
		if ok := func() bool {\
			for _, rr := range resp.Answer {\
				if _, ok := rr.(*dns.AAAA); ok {\
					return true\
				}\
			}\
			return false\
		}(); !ok || resp.Answer == nil {\
			req.Question[0].Qtype = dns.TypeA\
			resp, u, err = p.exchangeUpstreams(req, wrapped)\
		}\
	}\
' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/proxy.go

sudo sed -i -e '/if p.Config.EnableEDNSClientSubnet && d.ReqECS != nil {/i\
	if d.hasEDNS0 && d.ReqECS != nil {\
 ' -e '//d' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/proxycache.go

sudo sed -i -e '/if !p.EnableEDNSClientSubnet {/i\
	if !d.hasEDNS0 {\
 ' -e '//d' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/proxycache.go

sudo sed -i '/"maps"/a\ 	"net/netip"' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/upstreams.go

sudo sed -i '/type UpstreamConfig struct {/a\
	ToEDNSAddr map[any]netip.Addr\
	IdentifierUpstreams map[any][]upstream.Upstream\
' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/upstreams.go

sudo sed -i '/logger \*slog.Logger/a\
	toEDNSAddr map[any]netip.Addr\
	identifierUpstreams map[any][]upstream.Upstream\
' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/upstreams.go

sudo sed -i '/logger:                   opts.Logger,/a\
	toEDNSAddr:               map[any]netip.Addr{},\
 	identifierUpstreams:      map[any][]upstream.Upstream{},\
' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/upstreams.go

sudo sed -i '/Upstreams:                p.upstreams,/a\
	ToEDNSAddr:               p.toEDNSAddr,\
	IdentifierUpstreams:      p.identifierUpstreams,\
' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/upstreams.go

sudo sed -i 's/upstreams, domains, err := splitConfigLine(confLine)/upstreams, domains, err := p.splitConfigLine(confLine)/' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/upstreams.go

sudo sed -i 's/func splitConfigLine(confLine string) (upstreams, domains \[\]string, err error)/func (p *configParser) splitConfigLine(confLine string) (upstreams []string, domains []any, err error)/' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/upstreams.go

sudo sed -i 's/strings.HasPrefix(confLine, "\[\/")/strings.HasPrefix(confLine, "[\\\\")/' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/upstreams.go

sudo sed -i 's/domainsLine, upstreamsLine, found := strings.Cut(confLine\[len("\[\/"):\], "\/\]")/domainsLine, upstreamsLine, found := strings.Cut(confLine[len("[\\\\"):], "\\\\]")/' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/upstreams.go

sudo sed -i 's/_, confHost := range strings.Split(domainsLine, "\/")/_, confHost := range strings.Split(domainsLine, "\\\\")/' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/upstreams.go

sudo sed -i '/for _, confHost := range strings.Split(domainsLine, "\\\\") {/a\
		confHost, addrStr, found := strings.Cut(confHost, "|")\
		if found && (confHost == "" || addrStr == "") {\
			return nil, nil, errors.New("wrong upstream format")\
		}\
  ' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/upstreams.go

sudo sed -i '/host := strings.TrimPrefix(confHost, "\*.")/i\
		var addr netip.Addr\
		if found {\
			addr, err = netip.ParseAddr(addrStr)\
			if err != nil {\
				return nil, nil, err\
			}\
		}\
		if ip, err := netip.ParseAddr(confHost); err == nil {\
			if found {\
				p.toEDNSAddr[ip] = addr\
			}\
			domains = append(domains, ip)\
			continue\
		}\
		if subnet, err := netip.ParsePrefix(confHost); err == nil {\
			if found {\
				p.toEDNSAddr[subnet] = addr\
			}\
			domains = append(domains, subnet)\
			continue\
		}\
  ' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/upstreams.go

sudo sed -i '/domains = append(domains, strings.ToLower(confHost+"."))/i\
		if found {\
			p.toEDNSAddr[strings.ToLower(confHost+".")] = addr\
		}\
  ' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/upstreams.go

sudo sed -i 's/func (p \*configParser) specifyUpstream(domains \[\]string, u string, idx int) (err error)/func (p *configParser) specifyUpstream(domains []any, u string, idx int) (err error)/' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/upstreams.go

sudo sed -i 's/func (p \*configParser) excludeFromReserved(domains \[\]string)/func (p *configParser) excludeFromReserved(domains []any)/' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/upstreams.go

sudo sed -i '/if trimmed := strings.TrimPrefix(host, "\*."); trimmed != host {/i\
		switch host.(type) {\
		case string:\
			host := host.(string)\
  ' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/upstreams.go

sudo sed -i '/p.specifiedDomainUpstreams\[host\] = nil/a\
		}\
  ' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/upstreams.go

sudo sed -i 's/func (p \*configParser) includeToReserved(dnsUpstream upstream.Upstream, domains \[\]string)/func (p *configParser) includeToReserved(dnsUpstream upstream.Upstream, domains []any)/' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/upstreams.go

sudo sed -i '/if strings.HasPrefix(host, "\*.") {/i\
		switch host.(type) {\
		case string:\
			host := host.(string)\
  ' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/upstreams.go

sudo sed -i '/p.domainReservedUpstreams\[host\] = append(p.domainReservedUpstreams\[host\], dnsUpstream)/a\
		case netip.Addr, netip.Prefix:\
			p.identifierUpstreams[host] = append(p.identifierUpstreams[host], dnsUpstream)\
		}\
  ' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/upstreams.go

make CHANNEL=$1 GOOS=linux GOARCH=arm GOARM=7 OUT=dist/AdGuardHome/AdGuardHome

upx -9 dist/AdGuardHome/AdGuardHome

tar -C "dist" -c -f - "./AdGuardHome" | gzip -9 - > "../build/AdGuardHome_$1_linux_armv7.tar.gz"

cd ../

echo clean for $1

rm -rf AdGuardHome

#tar -czvf ./build/dnsproxy.tar.gz /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/*

go clean -modcache
}

mkdir build

CHANNEL=(edge beta release)
for i in "${CHANNEL[@]}"; do
	echo building for ${i}
	if [ "${i}" == "edge" ]; then
		git clone https://github.com/AdguardTeam/AdGuardHome
	else
		version=$(wget -qO- -t1 -T2 "https://static.adtidy.org/adguardhome/${i}/version.json" | grep "version" | head -n 1 | awk -F ":" '{print $2}' | sed 's/\"//g;s/,//g;s/ //g')
		git clone -b $version https://github.com/AdguardTeam/AdGuardHome
	fi
	BuildAdGuardHome ${i}
done
