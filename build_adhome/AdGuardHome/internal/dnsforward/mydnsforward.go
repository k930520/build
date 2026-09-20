package dnsforward

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/miekg/dns"
)

func (s *Server) myProcessFilteringBeforeRequest(
	ctx context.Context,
	l *slog.Logger,
	dctx *dnsContext,
) (rc resultCode) {
	l.DebugContext(ctx, "started processing filtering before request")
	defer l.DebugContext(ctx, "finished processing filtering before request")

	if dctx.proxyCtx.RequestedPrivateRDNS != (netip.Prefix{}) {
		// There is no need to filter request for locally served ARPA hostname
		// so disable redundant filters.
		dctx.setts.ParentalEnabled = false
		dctx.setts.SafeBrowsingEnabled = false
		dctx.setts.SafeSearchEnabled = false
		dctx.setts.ServicesRules = nil
	}

	if dctx.proxyCtx.Res != nil {
		// Go on since the response is already set.
		return resultCodeSuccess
	}

	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	var err error
	if dctx.result, err = s.filterDNSRequest(ctx, l, dctx); err != nil {
		dctx.err = err

		return resultCodeError
	}

	if dctx.result.ReqECS != "" {
		if ip, err := netutil.ParseIP(dctx.result.ReqECS); err == nil {
			if ipAddr, _ := netip.AddrFromSlice(ip); !netutil.IsSpecialPurpose(ipAddr) {
				dctx.proxyCtx.ReqECS = setReqECS(dctx.proxyCtx.Req, ip, 0)
			}
		}
	}

	if dctx.result.TransportOpt != nil {
		err := s.setTransport(ctx, l, dctx)
		if err != nil {
			l.ErrorContext(ctx, "failed to set transport option", err)
		}
	}

	return resultCodeSuccess
}

func setReqECS(m *dns.Msg, ip net.IP, scope uint8) (subnet *net.IPNet) {
	const (
		// defaultECSv4 is the default length of network mask for IPv4 address
		// in ECS option.
		defaultECSv4 = 24

		// defaultECSv6 is the default length of network mask for IPv6 address
		// in ECS.  The size of 7 octets is chosen as a reasonable minimum since
		// at least Google's public DNS refuses requests containing the options
		// with longer network masks.
		defaultECSv6 = 56
	)

	e := &dns.EDNS0_SUBNET{
		Code:        dns.EDNS0SUBNET,
		SourceScope: scope,
	}

	subnet = &net.IPNet{}
	if ip4 := ip.To4(); ip4 != nil {
		e.Family = 1
		e.SourceNetmask = defaultECSv4
		subnet.Mask = net.CIDRMask(defaultECSv4, netutil.IPv4BitLen)
		ip = ip4
	} else {
		// Assume the IP address has already been validated.
		e.Family = 2
		e.SourceNetmask = defaultECSv6
		subnet.Mask = net.CIDRMask(defaultECSv6, netutil.IPv6BitLen)
	}
	subnet.IP = ip.Mask(subnet.Mask)
	e.Address = subnet.IP

	// If OPT record already exists so just add EDNS option inside it.  Note
	// that servers may return FORMERR if they meet several OPT RRs.
	if opt := m.IsEdns0(); opt != nil {
		opt.Option = append(opt.Option, e)

		return subnet
	}

	// Create an OPT record and add EDNS option inside it.
	o := &dns.OPT{
		Hdr: dns.RR_Header{
			Name:   ".",
			Rrtype: dns.TypeOPT,
		},
		Option: []dns.EDNS0{e},
	}
	o.SetUDPSize(4096)
	m.Extra = append(m.Extra, o)

	return subnet
}

func (s *Server) setTransport(ctx context.Context, l *slog.Logger, dctx *dnsContext) error {
	if dctx.result.TransportOpt != nil {
		pctx := dctx.proxyCtx
		s.setCustomUpstream(ctx, l, pctx, "transport")
		if dctx.result.TransportOpt.Mode == "direct" {
			return nil
		}
		if pctx.Res == nil && !dctx.result.IsFiltered && (pctx.Req.Question[0].Qtype == dns.TypeA || pctx.Req.Question[0].Qtype == dns.TypeAAAA) {
			if s.processUpstream(ctx, l, dctx) == resultCodeSuccess {
				if pctx.Res != nil && pctx.Res.Answer != nil {
					host := dctx.origQuestion.Name
					if host == "" {
						host = pctx.Res.Question[0].Name
					}
					host = strings.TrimSuffix(host, ".")
					rule := s.transport.GetMatchRule(host, false)
					switch pctx.Res.Question[0].Qtype {
					case dns.TypeA:
						rule.IPv4 = make([]net.IP, 0)
						for _, rr := range pctx.Res.Answer {
							if a, ok := rr.(*dns.A); ok {
								rule.IPv4 = append(rule.IPv4, a.A)
							}
						}
					case dns.TypeAAAA:
						rule.IPv6 = make([]net.IP, 0)
						for _, rr := range pctx.Res.Answer {
							if aaaa, ok := rr.(*dns.AAAA); ok {
								rule.IPv6 = append(rule.IPv6, aaaa.AAAA)
							}
						}
					}
					switch mode := dctx.result.TransportOpt.Mode; mode {
					case "mitm", "migration", "proxy":
						rule.Args = ""
						rule.Mode, rule.Args = mode, dctx.result.TransportOpt.Args
					default:
						rule.Args = ""
						rule.Mode = mode
						if !rule.HasLookUpECH {
							qtype := pctx.Req.Question[0].Qtype
							pctx.Req.Question[0].Qtype = dns.TypeHTTPS
							pctx.Res = nil
							if s.processUpstream(ctx, l, dctx) == resultCodeSuccess {
								rule.HasLookUpECH = true
								for _, rr := range pctx.Res.Answer {
									if https, ok := rr.(*dns.HTTPS); ok {
										for _, opt := range https.Value {
											if ech, ok := opt.(*dns.SVCBECHConfig); ok {
												rule.ECH = ech.ECH
											}
										}
									}
								}
							}
							pctx.Req.Question[0].Qtype = qtype
						}
					}
					l.Info(host, rule)
				}
			}
		}
		addr, err := netip.ParseAddr(s.conf.TLSConf.ServerName)
		if err != nil {
			return fmt.Errorf("%q is not an ip address %w", s.conf.TLSConf.ServerName, err)
		}
		pctx.Res = s.genResponseWithIPs(ctx, pctx.Req, []netip.Addr{addr})
	}
	return nil
}

func (s *Server) HandleRequest(w http.ResponseWriter, r *http.Request) {
	if matchRule := s.transport.GetMatchRule(r.Host, true); matchRule == nil {
		ch := make(chan resultCode)
		go s.lookupIPAddr(context.Background(), r.Host, r.RemoteAddr, dns.TypeA, ch)
		go s.lookupIPAddr(context.Background(), r.Host, r.RemoteAddr, dns.TypeAAAA, ch)
		for range 2 {
			rc := <-ch
			s.logger.Info("resultCode", rc)
		}
	}
	if r.URL.Host == "" {
		r.URL.Host = r.Host
	}
	if r.URL.Scheme == "" {
		if r.TLS != nil {
			r.URL.Scheme = "https"
		}
	}
	s.transport.HandleRequest(w, r)
}

func (s *Server) lookupIPAddr(ctx context.Context, host, remoteAddr string, qtype uint16, ch chan resultCode) {
	addr, _ := netip.ParseAddrPort(remoteAddr)
	dctx := &dnsContext{
		proxyCtx: &proxy.DNSContext{
			Proto: proxy.ProtoUDP,
			Req:   (&dns.Msg{}).SetQuestion(dns.Fqdn(host), qtype),
			Addr:  addr,
		},
		result:    &filtering.Result{},
		startTime: time.Now(),
	}
	if rc := s.processInitial(ctx, s.logger, dctx); rc == resultCodeSuccess {
		ch <- s.myProcessFilteringBeforeRequest(ctx, s.logger, dctx)
	} else {
		ch <- rc
	}
}

