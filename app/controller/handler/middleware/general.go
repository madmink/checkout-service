package middleware

import (
	"bufio"
	"bytes"
	httpresponse "checkout-service/app/controller/handler/httpResponse"
	"checkout-service/app/util"
	"checkout-service/config"
	"checkout-service/log"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"runtime/debug"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
)

func Middleware(app *newrelic.Application) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			txnName := path

			var txn *newrelic.Transaction
			if app != nil {
				txn = app.StartTransaction(txnName)
				txn.SetWebRequestHTTP(r)
				w = txn.SetWebResponse(w)
			}

			ctx := r.Context()

			if txn != nil {
				ctx = newrelic.NewContext(ctx, txn)
				ctx = context.WithValue(ctx, "nrtxn", txn)
			}

			r = r.WithContext(ctx)

			defer func() {
				if txn != nil {
					txn.End()
				}

				if rec := recover(); rec != nil {
					if log.Logging != nil && log.Logging.Error != nil {
						log.Logging.Error.Errorf(
							util.RequestIDLog(r.Context())+
								"recovered. Error: %v; stack trace: %s",
							rec,
							string(debug.Stack()),
						)
					}

					httpresponse.Response(w, httpresponse.ServiceErrorDefault, http.StatusInternalServerError)
					return
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	body       []byte
}

func (rw *responseWriter) Write(p []byte) (int, error) {
	rw.body = append(rw.body, p...)
	return rw.ResponseWriter.Write(p)
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}

	return nil, nil, fmt.Errorf("hijacker not supported")
}

func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (rw *responseWriter) Push(target string, opts *http.PushOptions) error {
	if p, ok := rw.ResponseWriter.(http.Pusher); ok {
		return p.Push(target, opts)
	}

	return http.ErrNotSupported
}

func GetLogMiddleware(cfg config.ApplicationConfig) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return LogMiddleware(cfg, next)
	}
}

func LogMiddleware(cfg config.ApplicationConfig, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		r, requestID := util.EnsureRequestIDFromRequest(r)
		w.Header().Set(util.RequestIDHeader, requestID)

		var bodyBuf bytes.Buffer
		if r.Body != nil {
			if _, err := io.Copy(&bodyBuf, r.Body); err != nil {
				if log.Logging != nil && log.Logging.Access != nil {
					log.Logging.Access.Errorf(
						util.RequestIDLog(r.Context())+"Error copying request body: %v",
						err,
					)
				}
			}
		}

		r.Body = io.NopCloser(bytes.NewReader(bodyBuf.Bytes()))

		if cfg.Log.Request {
			reqb, err := httputil.DumpRequest(r, true)
			if err != nil {
				if log.Logging != nil && log.Logging.Access != nil {
					log.Logging.Access.Errorf(
						util.RequestIDLog(r.Context())+"Error dumping request: %v",
						err,
					)
				}
			} else {
				if log.Logging != nil && log.Logging.Access != nil {
					log.Logging.Access.Infof(
						util.RequestIDLog(r.Context())+"Request : %s",
						string(reqb),
					)
				}
			}
		}

		// Restore again after DumpRequest, so handler can still read the body.
		r.Body = io.NopCloser(bytes.NewReader(bodyBuf.Bytes()))

		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		shouldLogSuccess := rw.statusCode >= http.StatusOK &&
			rw.statusCode < http.StatusBadRequest &&
			cfg.Log.SuccessResponse

		shouldLogFailed := rw.statusCode >= http.StatusBadRequest &&
			cfg.Log.FailedResponse

		if shouldLogSuccess || shouldLogFailed {
			if log.Logging != nil && log.Logging.Access != nil {
				log.Logging.Access.Infof(
					util.RequestIDLog(r.Context())+
						"Response: Status Code: %d, Duration: %s, Body: %s",
					rw.statusCode,
					duration,
					string(rw.body),
				)
			}
		}
	})
}
