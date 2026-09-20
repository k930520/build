package transport

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"golang.org/x/net/proxy"
)

type Protocol int

const (
	ProtocolAuto Protocol = iota
	ProtocolHTTP2HTTP1
	ProtocolHTTP3
)

type Rule struct {
	Domain          string
	Mode            string
	Args            string
	HasLookUpECH    bool
	ECH             []byte
	IPv4            []net.IP
	IPv6            []net.IP
	protocolSupport Protocol
}
type Transport struct {
	logger      *slog.Logger
	mu          *sync.Mutex
	rules       map[string]*Rule
	h2Transport *http.Transport
	h3Transport *http3.Transport
}

func (t *Transport) RoundTrip(r *http.Request) (resp *http.Response, err error) {
	matchRule := t.GetMatchRule(r.Host, true)
	if matchRule == nil {
		t.logger.ErrorContext(r.Context(), r.Host, slogutil.KeyError, fmt.Errorf("rule not found"))
		return nil, fmt.Errorf("%q rule not found", r.Host)
	}
	t.logger.Info("host and mode", r.Host, matchRule.Mode)
	switch matchRule.Mode {
	case "mitm", "tls-rf", "proxy":
		return t.h2Transport.RoundTrip(r)
	case "quic":
		return t.h3Transport.RoundTrip(r)
	default:
		switch matchRule.protocolSupport {
		case ProtocolHTTP3:
			resp, err = t.h3Transport.RoundTrip(r)
		case ProtocolHTTP2HTTP1:
			resp, err = t.h2Transport.RoundTrip(r)
		case ProtocolAuto:
			resp, err = t.h3Transport.RoundTrip(r.Clone(r.Context()))
			if err == nil {
				matchRule.protocolSupport = ProtocolHTTP3
			} else {
				resp, err = t.h2Transport.RoundTrip(r.Clone(r.Context()))
				if err == nil {
					matchRule.protocolSupport = ProtocolHTTP2HTTP1
				}
			}
		}
		return resp, err
	}
}

func NewTransport(l *slog.Logger) *Transport {
	tr := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: 10 * time.Second,
			KeepAliveConfig: net.KeepAliveConfig{
				Enable:   true,
				Idle:     30 * time.Second,
				Interval: 10 * time.Second,
				Count:    3,
			},
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          60,
		MaxIdleConnsPerHost:   3,
		MaxConnsPerHost:       6,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	p := &Transport{
		logger:      l.With(slogutil.KeyPrefix, "transport"),
		mu:          &sync.Mutex{},
		rules:       make(map[string]*Rule),
		h2Transport: tr,
		h3Transport: &http3.Transport{TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS13,
			MaxVersion: tls.VersionTLS13,
			NextProtos: []string{"h3"},
		}},
	}

	p.h2Transport.DialTLSContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		logger := p.logger.With(slogutil.KeyPrefix, "h2-h1")
		host, port, matchRule, err := p.GetMatch(addr, logger)
		if err != nil {
			logger.ErrorContext(ctx, addr, slogutil.KeyError, err)
			return nil, err
		}
		rawConn, err := func() (net.Conn, error) {
			if matchRule.Mode == "proxy" {
				dialer, err := proxy.SOCKS5(network, matchRule.Args, nil, proxy.Direct)
				if err != nil {
					logger.ErrorContext(ctx, "failed to create proxy dialer", slogutil.KeyError, err)
					return nil, fmt.Errorf("failed to create proxy dialer: %w", err)
				}
				conn, err := dialer.Dial(network, addr)
				if err != nil {
					logger.ErrorContext(ctx, "failed to create proxy Conn", slogutil.KeyError, err)
					return nil, fmt.Errorf("failed to create proxy Conn: %w", err)
				}
				return conn, err
			}
			dialer := &net.Dialer{
				Timeout: 10 * time.Second,
				KeepAliveConfig: net.KeepAliveConfig{
					Enable:   true,
					Idle:     30 * time.Second,
					Interval: 10 * time.Second,
					Count:    3,
				},
			}
			ips := append(matchRule.IPv4, matchRule.IPv6...)
			conn, err := staggeredRace(ctx, len(ips), 250*time.Millisecond,
				func(rctx context.Context, idx int) (net.Conn, error) {
					return dialer.DialContext(rctx, network, netutil.JoinHostPort(ips[idx].String(), port))
				},
				func(c net.Conn) { _ = c.Close() },
			)
			if err != nil {
				logger.ErrorContext(ctx, "failed to create Conn", slogutil.KeyError, err)
				return nil, fmt.Errorf("failed to create Conn: %w", err)
			}
			if len(matchRule.ECH) > 0 {
				return conn, err
			}
			return &recordHandshakeFragConn{
				Conn: conn,
			}, err
		}()
		if err != nil {
			return nil, err
		}
		logger.Info("Conn remote network address", host, rawConn.RemoteAddr().String())
		tlsConfig := &tls.Config{
			ServerName:                     host,
			MinVersion:                     tls.VersionTLS13,
			MaxVersion:                     tls.VersionTLS13,
			NextProtos:                     []string{"h2", "http/1.1"},
			EncryptedClientHelloConfigList: matchRule.ECH,
		}
		if matchRule.Mode == "mitm" && matchRule.Args != host {
			tlsConfig.ServerName = matchRule.Args
			tlsConfig.InsecureSkipVerify = true
		}
		tlsConn := tls.Client(rawConn, tlsConfig)
		err = tlsConn.Handshake()
		if err != nil {
			_ = rawConn.Close()
			logger.ErrorContext(ctx, "TLS handshake failed", slogutil.KeyError, err)
			return nil, err
		}
		state := tlsConn.ConnectionState()
		logger.Info("ConnectionState", host, state.NegotiatedProtocol, host, state.Version)
		return tlsConn, nil
	}

	p.h3Transport.Dial = func(ctx context.Context, addr string, tlsCfg *tls.Config, cfg *quic.Config) (*quic.Conn, error) {
		logger := p.logger.With(slogutil.KeyPrefix, "h3")
		host, port, matchRule, err := p.GetMatch(addr, logger)
		if err != nil {
			logger.ErrorContext(ctx, addr, slogutil.KeyError, err)
			return nil, err
		}
		tlsCfg.EncryptedClientHelloConfigList = matchRule.ECH
		ips := append(matchRule.IPv4, matchRule.IPv6...)
		conn, err := staggeredRace(ctx, len(ips), 250*time.Millisecond,
			func(rctx context.Context, idx int) (*quic.Conn, error) {
				return quic.DialAddrEarly(rctx, netutil.JoinHostPort(ips[idx].String(), port), tlsCfg, cfg)
			},
			func(c *quic.Conn) { _ = c.CloseWithError(0, "lost happy-eyeballs race") },
		)
		if err != nil {
			logger.ErrorContext(ctx, "failed to create Conn", slogutil.KeyError, err)
			return nil, err
		}
		logger.Info("Conn remote network address", host, conn.RemoteAddr().String())
		return conn, nil
	}
	return p
}

func (t *Transport) GetMatch(addr string, logger *slog.Logger) (string, uint16, *Rule, error) {
	host, port, err := netutil.SplitHostPort(addr)
	if err != nil {
		return "", 0, nil, err
	}
	matchRule := t.GetMatchRule(host, true)
	if matchRule == nil {
		return "", 0, nil, fmt.Errorf("%q rule not found", host)
	}
	logger.Info("host rule", host, matchRule)
	return host, port, matchRule, nil
}

func (t *Transport) GetMatchRule(host string, canNil bool) *Rule {
	t.mu.Lock()
	defer t.mu.Unlock()

	_, ok := t.rules[host]
	if !ok {
		if canNil {
			return nil
		}
		t.rules[host] = &Rule{Domain: host}
	}
	return t.rules[host]
}

func staggeredRace[T any](ctx context.Context, n int, attemptDelay time.Duration, dial func(ctx context.Context, idx int) (T, error), closeLoser func(T)) (T, error) {
	var zero T
	if n <= 0 {
		return zero, fmt.Errorf("no candidates to dial")
	}

	raceCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	type result struct {
		val T
		err error
	}
	results := make(chan result, n)

	launched, pending := 0, 0
	launchNext := func() {
		idx := launched
		go func() {
			v, err := dial(raceCtx, idx)
			results <- result{val: v, err: err}
		}()
		launched++
		pending++
	}

	timer := time.NewTimer(attemptDelay)
	defer timer.Stop()
	safeReset := func(d time.Duration) {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(d)
	}

	launchNext()

	var lastErr error
	for {
		select {
		case <-ctx.Done():
			// The parent cancelled the race. defer cancel() stops the losing
			// dials, but a dial that already succeeded (its result sitting in
			// the buffered channel, or landing right after cancellation) would
			// otherwise leak its *net.Conn / *quic.Conn. Drain the outstanding
			// results and close any that connected — mirror the winner-path
			// drain below. results is buffered to n, so this can't block forever.
			if pending > 0 {
				go func(rem int) {
					for range rem {
						lr := <-results
						if lr.err == nil {
							closeLoser(lr.val)
						}
					}
				}(pending)
			}
			return zero, ctx.Err()

		case <-timer.C:
			if launched < n {
				launchNext()
			}
			if launched < n {
				safeReset(attemptDelay)
			}

		case r := <-results:
			pending--
			if r.err == nil {
				cancel() // stop the losing dials
				if pending > 0 {
					go func(rem int) {
						for range rem {
							lr := <-results
							if lr.err == nil {
								closeLoser(lr.val)
							}
						}
					}(pending)
				}
				return r.val, nil
			}
			lastErr = r.err
			// Fail-fast: start the next candidate immediately rather than
			// waiting for the timer.
			if launched < n {
				launchNext()
				if launched < n {
					safeReset(attemptDelay)
				}
			} else if pending == 0 {
				if lastErr == nil {
					lastErr = fmt.Errorf("all dial attempts failed")
				}
				return zero, lastErr
			}
		}
	}
}
