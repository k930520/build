BuildAdGuardHome() {

sudo cp -r build_adhome/AdGuardHome/* AdGuardHome

sudo sed -i '/type DefaultManager struct {/a\
	CAPair\
	' AdGuardHome/internal/aghtls/defaultmanager.go

sudo sed -i '/func (mgr *DefaultManager) onGetCertificate(\
	chi *tls.ClientHelloInfo) (cert *tls.Certificate, err error,\
) {/i\
	return mgr.myOnGetCertificate(chi)\
	' AdGuardHome/internal/aghtls/defaultmanager.go

sudo sed -i '/	err = validateCertificates(/c\	err = myValidateCertificates(' AdGuardHome/internal/aghtls/defaultmanager.go

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

make CHANNEL=$1 GOOS=linux GOARCH=arm GOARM=7 OUT=dist/AdGuardHome/AdGuardHome

sudo tar -czvf ../build/$1_static.tar.gz ./build/*
sudo tar -czvf ../build/$1_internal.tar.gz ./internal/*
sudo tar -czvf ../build/$1_urlfilter.tar.gz /home/runner/go/pkg/mod/github.com/\!adguard\!team/$urlfilter/rules/*

sudo upx -9 dist/AdGuardHome/AdGuardHome

sudo tar -C "dist" -c -f - "./AdGuardHome" | gzip -9 - > "../build/AdGuardHome_$1_linux_armv7.tar.gz"

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
