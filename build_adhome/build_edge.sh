BuildAdGuardHome() {

sudo cp -r build_adhome/AdGuardHome/* AdGuardHome

sudo sed -i '/type DefaultManager struct {/a\
	rootPair\
	certs       map[string]*tls.Certificate\
	' AdGuardHome/internal/aghtls/defaultmanager.go

sudo sed -i '/GetCertificate:/ s/mgr\.onGetCertificate/mgr.myOnGetCertificate/g' AdGuardHome/internal/aghtls/defaultmanager.go

sudo sed -i '/	err = validateCertificates(/c\	err = myValidateCertificates(' AdGuardHome/internal/aghtls/defaultmanager.go

sudo sed -i '/"github.com\/miekg\/dns"/a\
	"github.com/AdguardTeam/AdGuardHome/internal/transport"\
	' AdGuardHome/internal/dnsforward/dnsforward.go

sudo sed -i '/type Server struct {/a\
	transport *transport.Transport\
	' AdGuardHome/internal/dnsforward/dnsforward.go

sudo sed -i '/	s = &Server{/a\
		transport:   transport.NewTransport(p.Logger),' AdGuardHome/internal/dnsforward/dnsforward.go

sudo sed -i '/func (s \*Server) Resolve(ctx context.Context, net, host string) (addr \[\]netip.Addr, err error) {/a\
	for _, u := range []uint16{dns.TypeA, dns.TypeAAAA} {\
		resVal, err := s.dnsFilter.CheckHost(strings.TrimSuffix(host,"."), u, &filtering.Settings{FilteringEnabled: true})\
		if err == nil {\
			if filtering.Rewritten ==resVal.Reason && resVal.CanonName != "" && len(resVal.IPList) == 0 {\
				host = dns.Fqdn(resVal.CanonName)\
				break\
			}\
			if filtering.Rewritten ==resVal.Reason {\
				addr = append(addr, resVal.IPList...)\
			}\
		}\
	}\
	if len(addr) > 0 {\
		return addr, nil\
	}\
 ' AdGuardHome/internal/dnsforward/dnsforward.go

sudo sed -i '/		s.processFilteringBeforeRequest,/c\		s.myProcessFilteringBeforeRequest,' AdGuardHome/internal/dnsforward/requesthandler.go

sudo sed -i '/type Result struct {/a\
	ReqECS       string\
	TransportOpt \*rules.TransportOpt\
	' AdGuardHome/internal/filtering/result.go

sudo sed -i '/	dnsRWRes := d.processDNSResultRewrites(dnsres, host)/c\	dnsRWRes := d.myProcessDNSResultRewrites(dnsres, host)' AdGuardHome/internal/filtering/filtering.go

sudo sed -i '/	res = d.matchHostProcessDNSResult(rrtype, dnsres)/c\	res = d.myMatchHostProcessDNSResult(rrtype, dnsres)' AdGuardHome/internal/filtering/filtering.go

sudo sed -i '/		shouldContinue := web.serveTLS(ctx)/c\		shouldContinue := web.myServeTLS(ctx)' AdGuardHome/internal/home/web.go

sudo sed -i '/		return dlURL, key, true/c\		return u.getDlURL(dlURL), key, true' AdGuardHome/internal/updater/check.go

cd AdGuardHome

go mod tidy

urlfilter=$(ls /home/runner/go/pkg/mod/github.com/\!adguard\!team | grep urlfilter)

echo urlfilter is $urlfilter

sudo cp -r ../build_adhome/urlfilter/rules/* /home/runner/go/pkg/mod/github.com/\!adguard\!team/$urlfilter/rules

sudo sed -i '/type NetworkRule struct {/a\
	ECS          string\
	TransportOpt \*TransportOpt\
	' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$urlfilter/rules/network.go

sudo sed -i '/		!r.matchDNSType(req.DNSType),/c\		!r.myMatchDNSType(req.DNSType),' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$urlfilter/rules/network.go

sudo sed -i '/"respgeo": setRespGeoOptionHandler,/a\
	"ecs":       setECSOptionHandler,\
	"transport": setTransportOptionHandler,\
	' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$urlfilter/rules/network.go

sudo sed -i '/func GetDNSBasicRule(rules \[\]\*NetworkRule) (basicRule \*NetworkRule) {/{n;d;}' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$urlfilter/rules/match.go
sudo sed -i '/func GetDNSBasicRule(rules \[\]\*NetworkRule) (basicRule \*NetworkRule) {/{n;d;}' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$urlfilter/rules/match.go

sudo sed -i '/func GetDNSBasicRule(rules \[\]\*NetworkRule) (basicRule \*NetworkRule) {/a\
	rules = removeBadfilterRules(rules)\
	rules = myRemoveDNSRewriteRules(rules)\
' /home/runner/go/pkg/mod/github.com/\!adguard\!team/$urlfilter/rules/match.go

make CHANNEL=$1 GOOS=linux GOARCH=arm GOARM=7 OUT=dist/AdGuardHome/AdGuardHome

sudo upx -9 dist/AdGuardHome/AdGuardHome

sudo tar -C "dist" -c -f - "./AdGuardHome" | gzip -9 - > "../build/AdGuardHome_$1_linux_armv7.tar.gz"

sudo tar -czvf ../build/$1_static.tar.gz build/*
sudo tar -czvf ../build/$1_internal.tar.gz internal/*
sudo tar -czvf ../build/$1_urlfilter.tar.gz /home/runner/go/pkg/mod/github.com/\!adguard\!team/$urlfilter/rules/*

cd ../

echo clean for $1

sudo rm -rf AdGuardHome

go clean -modcache
}

mkdir build

CHANNEL=(edge)
for i in "${CHANNEL[@]}"; do
	echo building for ${i}
	version=$(wget -qO- -t1 -T2 "https://static.adtidy.org/adguardhome/${i}/version.json" | grep "version" | head -n 1 | awk -F ":" '{print $2}' | sed 's/\"//g;s/,//g;s/ //g')
	if [ "${i}" == "edge" ]; then
		git clone https://github.com/AdguardTeam/AdGuardHome
	else
		git clone -b $version https://github.com/AdguardTeam/AdGuardHome
	fi
	BuildAdGuardHome ${i}
done
