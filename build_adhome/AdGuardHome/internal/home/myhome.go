package home

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	
	"github.com/AdguardTeam/AdGuardHome/internal/transport"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/netutil/httputil"
)

func (web *webAPI) myServeTLS(ctx context.Context) (next bool) {
	if !web.httpsServer.waitForTLSReady() {
		return false
	}

	var addr string
	func() {
		config.RLock()
		defer config.RUnlock()

		addr = netutil.JoinHostPort(config.TLS.ServerName, config.TLS.PortHTTPS)
	}()

	logger := web.baseLogger.With(loggerKeyServer, "https")

	hdlr := web.myWrapMux(logger)

	web.httpsServer.server = &http.Server{
		Addr:              addr,
		Handler:           hdlr,
		TLSConfig:         web.tlsManager.TLSConfig(),
		ReadTimeout:       web.conf.ReadTimeout,
		ReadHeaderTimeout: web.conf.ReadHeaderTimeout,
		WriteTimeout:      web.conf.WriteTimeout,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	extTLSConf := web.tlsManager.ExtendedTLSConfig()
	printHTTPSAddresses(ctx, web.logger, extTLSConf)

	if web.conf.serveHTTP3 {
		go web.mustStartHTTP3(ctx, addr)
	}

	logger.InfoContext(ctx, "starting https server")
	err := web.httpsServer.server.ListenAndServeTLS("", "")
	if !errors.Is(err, http.ErrServerClosed) {
		cleanupAlways(ctx, logger, web.pidFilePath)

		panic(fmt.Errorf("https: %w", err))
	}

	return true
}

func (web *webAPI) myWrapMux(l *slog.Logger) (h http.Handler) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != web.tlsManager.ExtendedTLSConfig().ServerName {
			logMw := httputil.NewLogMiddleware(l, slog.LevelDebug)
			h = logMw.Wrap(http.HandlerFunc(globalContext.dnsServer.HandleRequest))
		} else {
			h = web.wrapMux(l)
		}
		h.ServeHTTP(w, r)
	})
}
