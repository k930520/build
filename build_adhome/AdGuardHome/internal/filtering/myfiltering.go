package filtering

import (
	"slices"

	"github.com/AdguardTeam/urlfilter"
	"github.com/AdguardTeam/urlfilter/rules"
)

func (d *DNSFilter) myProcessDNSResultRewrites(
	dnsres *urlfilter.DNSResult,
	host string,
) (dnsRWRes Result) {
	dnsr := dnsres.DNSRewrites()
	if len(dnsr) == 0 {
		return Result{}
	}

	maxLen := len(dnsr[0].Shortcut)
	if dnsres.NetworkRule != nil && len(dnsres.NetworkRule.Shortcut) >= maxLen {
		return Result{}
	}

	dnsr = slices.DeleteFunc(dnsr, func(nr *rules.NetworkRule) bool {
		return len(nr.Shortcut) < maxLen
	})

	res := d.processDNSRewrites(dnsr)
	if res.Reason == RewrittenRule && res.CanonName == host {
		// A rewrite of a host to itself.  Go on and try matching other things.
		return Result{}
	}

	for _, nr := range dnsr {
		if nr.ECS != "" {
			res.ReqECS = nr.ECS
		}
		if nr.TransportOpt != nil {
			res.TransportOpt = nr.TransportOpt
		}
	}

	return res
}

func (d *DNSFilter) myMatchHostProcessDNSResult(
	qtype uint16,
	dnsres *urlfilter.DNSResult,
) (res Result) {
	if dnsres.NetworkRule != nil {
		reason := FilteredBlockList
		if dnsres.NetworkRule.Whitelist {
			reason = NotFilteredAllowList
		}

		if dnsres.NetworkRule.IsMatchDNSTypeRewrittenCNAME() {
			res = d.processDNSRewrites([]*rules.NetworkRule{})
		} else {
			if dnsres.NetworkRule.DNSRewrite != nil {
				res = d.processDNSRewrites([]*rules.NetworkRule{dnsres.NetworkRule})
			} else {
				res = makeResult([]rules.Rule{dnsres.NetworkRule}, reason)
			}
		}
		res.Reason = reason
		res.IsFiltered = reason == FilteredBlockList
		res.ReqECS = dnsres.NetworkRule.ECS
		res.TransportOpt = dnsres.NetworkRule.TransportOpt
		return res
	}

	if result, ok := resultFromHostRules(qtype, dnsres); ok {
		return result
	}

	return hostResultForOtherQType(dnsres)
}
