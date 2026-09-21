package transport

import (
	"io"
	"net/http"

	"github.com/AdguardTeam/golibs/logutil/slogutil"
)

func (t *Transport) HandleRequest(w http.ResponseWriter, r *http.Request) {
	logger := t.logger.With(slogutil.KeyPrefix, "handle")
	resp, err := t.RoundTrip(r)
	if err != nil {
		logger.ErrorContext(r.Context(), "transport roundtrip is failed", slogutil.KeyError, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
	return
}
