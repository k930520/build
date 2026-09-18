package rules

import (
	"fmt"
	"strings"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/netutil"
)

type TransportOpt struct {
	Mode string
	Args string
}

func (r *NetworkRule) IsMatchDNSTypeRewrittenCNAME() (ok bool) {
	if len(r.permittedDNSTypes) != 0 || len(r.restrictedDNSTypes) != 0 {
		return r.DNSRewrite != nil && r.DNSRewrite.NewCNAME != "" && !r.Whitelist
	}
	return false
}

func (r *NetworkRule) myMatchDNSType(rtype uint16) (allowed bool) {
	if len(r.permittedDNSTypes) == 0 && len(r.restrictedDNSTypes) == 0 {
		r.Whitelist = r.DNSRewrite == nil
	} else {
		r.Whitelist = !r.matchDNSType(rtype)
	}
	return true
}

func setECSOptionHandler(r *NetworkRule, value string) (err error) {
	if value == "" {
		return errors.ErrEmptyValue
	}
	if netutil.IsValidIPString(value) || netutil.IsValidIPPrefixString(value) {
		r.ECS = value
		return nil
	}
	return fmt.Errorf("unknown value: %q", value)
}

func setTransportOptionHandler(r *NetworkRule, value string) (err error) {
	if value == "" {
		r.TransportOpt = &TransportOpt{Mode: "auto"}
		return nil
	}
	parts := strings.SplitN(value, ";", 2)
	switch len(parts) {
	case 1:
		switch strings.ToLower(value) {
		case "direct", "mitm", "ech", "quic", "tls-rf", "auto":
			r.TransportOpt = &TransportOpt{Mode: value}
			return nil
		default:
			if netutil.IsValidHostname(value) {
				r.TransportOpt = &TransportOpt{Mode: "mitm", Args: value}
				return nil
			}
			return fmt.Errorf("unknown keyword: %q", value)
		}
	case 2:
		switch strings.ToLower(parts[0]) {
		case "mitm", "migration", "proxy":
			r.TransportOpt = &TransportOpt{Mode: parts[0], Args: parts[1]}
			return nil
		default:
			return fmt.Errorf("unknown keyword: %q", parts[0])
		}
	default:
		return fmt.Errorf("SplitN returned %d parts", len(parts))
	}
	return nil
}

func myRemoveDNSRewriteRules(rules []*NetworkRule) (filtered []*NetworkRule) {
	// Assume that DNS rewrite rules are rare, and return the original slice if
	// there are none.

	var i int
	var found bool
	for i = range rules {
		if rules[i].DNSRewrite != nil && len(rules[i].permittedDNSTypes) == 0 && len(rules[i].restrictedDNSTypes) == 0 {
			found = true

			break
		}
	}

	if !found {
		return rules
	}

	filtered = rules[:i:i]
	for ; i < len(rules); i++ {
		r := rules[i]
		if !(rules[i].DNSRewrite != nil && len(rules[i].permittedDNSTypes) == 0 && len(rules[i].restrictedDNSTypes) == 0) {
			filtered = append(filtered, r)
		}
	}

	return filtered
}
