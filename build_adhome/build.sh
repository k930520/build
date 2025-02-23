BuildAdGuardHome() {
sudo sed -i '/"slices"/a\ 	"strings"' AdGuardHome/internal/updater/check.go

sudo sed -i '/return dlURL, key, true/i\
	split := strings.Split(dlURL, "/")\
	dlURL = "http://github.home.local/https://github.com/k930520/build/releases/download/adhome/" + strings.Replace(split[len(split)-1], "AdGuardHome", "AdGuardHome_"+u.channel, 1)\
 ' AdGuardHome/internal/updater/check.go

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

cd AdGuardHome

#tar -czvf ../build/AdGuardHome.tar.gz internal/*

go mod tidy

dnsproxy=$(ls /home/runner/go/pkg/mod/github.com/\!adguard\!team | grep dnsproxy)

echo dnsproxy is $dnsproxy

sudo sed -i '/if withECS {/d' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/cache.go

sudo sed -i -e '/c.itemsWithSubnet = createCache(size)/{s/.*/	c.itemsWithSubnet = c.items/;n;d;}' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/cache.go

sudo sed -i '/"slices"/a\ 	"strings"' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/proxy.go

sudo sed -i '/getUpstreams := (\*UpstreamConfig).getUpstreamsForDomain/i\
	upstreams, found := p.UpstreamConfig.IdentifierToUpstreams.ipToUpstreams[d.Addr.Addr()]\
	if found {\
		return upstreams, false\
	}\
	ipWithoutZone := d.Addr.Addr().WithZone("")\
	for pref, upstreams := range p.UpstreamConfig.IdentifierToUpstreams.subnetToUpstreams {\
		if pref.Contains(ipWithoutZone) {\
			return upstreams, false\
		}\
	}\
' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/proxy.go

sudo sed -i '/dctx\.calcFlagsAndSize()/i\
	for fqdn := dctx.Req.Question[0].Name; fqdn != ""; {\
		addr, ok := p.UpstreamConfig.DomainEDNSAddr["*."+fqdn]\
		if ok && fqdn != dctx.Req.Question[0].Name {\
			dctx.processECS(addr.AsSlice(), p.logger)\
			break\
		}\
		addr, ok = p.UpstreamConfig.DomainEDNSAddr[fqdn]\
		if ok {\
			dctx.processECS(addr.AsSlice(), p.logger)\
			break\
		}\
		_, fqdn, _ = strings.Cut(fqdn, ".")\
	}\
 ' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/proxy.go

sudo sed -i -e '/if p.Config.EnableEDNSClientSubnet && d.ReqECS != nil {/i\
	if d.hasEDNS0 && d.ReqECS != nil {\
 ' -e '//d' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/proxycache.go

sudo sed -i -e '/if !p.EnableEDNSClientSubnet {/i\
	if !d.hasEDNS0 {\
 ' -e '//d' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/proxycache.go

sudo sed -i '/"maps"/a\ 	"net/netip"' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/upstreams.go

sudo sed -i '/type UpstreamConfig struct {/i\
type IdentifierToUpstreams struct {\
	ipToUpstreams map[netip.Addr][]upstream.Upstream\
	subnetToUpstreams map[netip.Prefix][]upstream.Upstream\
}\
' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/upstreams.go

sudo sed -i '/type UpstreamConfig struct {/a\
	DomainEDNSAddr map[string]netip.Addr\
	IdentifierToUpstreams IdentifierToUpstreams\
' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/upstreams.go

sudo sed -i '/logger \*slog.Logger/a\
	domainEDNSAddr map[string]netip.Addr\
	identifierToUpstreams IdentifierToUpstreams\
' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/upstreams.go

sudo sed -i '/logger:                   opts.Logger,/a\
	domainEDNSAddr:           map[string]netip.Addr{},\
	identifierToUpstreams:    IdentifierToUpstreams{ipToUpstreams: map[netip.Addr][]upstream.Upstream{}, subnetToUpstreams: map[netip.Prefix][]upstream.Upstream{}},\
' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/upstreams.go

sudo sed -i '/Upstreams:                p.upstreams,/a\
	DomainEDNSAddr:           p.domainEDNSAddr,\
	IdentifierToUpstreams:    p.identifierToUpstreams,
' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/upstreams.go

sudo sed -i 's/upstreams, domains, err := splitConfigLine(confLine)/upstreams, domains, err := p.splitConfigLine(confLine)/' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/upstreams.go

sudo sed -i 's/func splitConfigLine/func (p *configParser) splitConfigLine/' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/upstreams.go

sudo sed -i '/upstreams\[0\] == "#" && len(domains) > 0/i\
	if upstreams == nil {\
		return nil\
	}\
' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/upstreams.go

sudo sed -i '/strings.HasPrefix(confLine, "\[\/")/i\
	argsLine, upstreamsLine, found := strings.Cut(confLine[len("[@"):], "@]")\
	if found && upstreamsLine != "" {\
		idUpstreams := []upstream.Upstream{}\
		for _, u := range strings.Fields(upstreamsLine) {\
			if u != "#" {\
				dnsUpstream, err := upstream.AddressToUpstream(u, p.options.Clone())\
				if err != nil {\
					return nil, nil, fmt.Errorf("cannot prepare the upstream: %s", err)\
				}\
				idUpstreams = append(idUpstreams, dnsUpstream)\
			}\
		}\
		for _, confID := range strings.Split(argsLine, "@") {\
			if confID == "" {\
				return nil, nil, errors.Error("wrong upstream format")\
			}\
			if ip, err := netip.ParseAddr(confID); err == nil {\
				p.identifierToUpstreams.ipToUpstreams[ip] = idUpstreams\
				continue\
			}\
			if subnet, err := netip.ParsePrefix(confID); err == nil {\
				p.identifierToUpstreams.subnetToUpstreams[subnet] = idUpstreams\
				continue\
			}\
		}\
		return nil, nil, nil\
	}\
' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/upstreams.go

sudo sed -i '/for _, confHost := range strings.Split(domainsLine, "\/") {/a\
		confHost, addrStr, fond := strings.Cut(confHost, "|")\
		if fond && (confHost == "" || addrStr == "") {\
			return nil, nil, errors.New("wrong upstream format")\
		}\
  ' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$dnsproxy/proxy/upstreams.go

sudo sed -i '/domains = append(domains, strings.ToLower(confHost+"\."))/i\
		addr, err := netip.ParseAddr(addrStr)\
		if err != nil {\
			return nil, nil, err\
		}\
		p.domainEDNSAddr[strings.ToLower(confHost+".")] = addr\
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
