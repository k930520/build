package home

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/netutil/httputil"
	"github.com/AdguardTeam/golibs/netutil/urlutil"
)

func (web *webAPI) myServeTLS(ctx context.Context) (next bool) {
	if !web.httpsServer.waitForTLSReady() {
		return false
	}

	logger := web.baseLogger.With(loggerKeyServer, "https")

	hdlr := web.myWrapMux(logger)

	web.httpsServer.server = &http.Server{
		Handler:           hdlr,
		TLSConfig:         web.tlsManager.TLSConfig(),
		ReadTimeout:       web.conf.ReadTimeout,
		ReadHeaderTimeout: web.conf.ReadHeaderTimeout,
		WriteTimeout:      web.conf.WriteTimeout,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	var bindHosts []netip.Addr
	var portHTTPS uint16
	func() {
		config.RLock()
		defer config.RUnlock()

		bindHosts = config.DNS.BindHosts
		portHTTPS = config.TLS.PortHTTPS
	}()
	for _, host := range bindHosts {
		addr := netutil.JoinHostPort(host.String(), portHTTPS)

		if web.conf.serveHTTP3 {
			go web.mustStartHTTP3(ctx, addr)
		}

		ln, err := net.Listen("tcp", addr)
		if err != nil {
			cleanupAlways(ctx, logger, web.pidFilePath)

			panic(fmt.Errorf("tcp listener: %w", err))
		}

		printWebAddrs(ctx, web.logger, urlutil.SchemeHTTPS, host.String(), portHTTPS)

		go func() {
			logger.InfoContext(ctx, "starting https server")
			err = web.httpsServer.server.ServeTLS(ln, "", "")
			if !errors.Is(err, http.ErrServerClosed) {
				cleanupAlways(ctx, logger, web.pidFilePath)

				panic(fmt.Errorf("https: %w", err))
			}
		}()
	}

	return true
}

func (web *webAPI) myWrapMux(l *slog.Logger) (h http.Handler) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if netutil.IsValidIPString(r.Host) || r.Host == web.tlsManager.ExtendedTLSConfig().ServerName {
			h = web.wrapMux(l)
		} else {
			logMw := httputil.NewLogMiddleware(l, slog.LevelDebug)
			h = logMw.Wrap(http.HandlerFunc(globalContext.dnsServer.HandleRequest))
		}
		h.ServeHTTP(w, r)
	})
}
