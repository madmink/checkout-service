// Package log
package log

import (
	"context"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/sirupsen/logrus"
)

const TimeStampFormat = time.RFC3339

// Logger ---- public logger handles ----
type Logger struct {
	Access *logrus.Logger
	Error  *logrus.Logger
}

var (
	Logging  *Logger
	LogLevel = logrus.InfoLevel
)

// Config ---- config for rotating files ----
type Config struct {
	FileName    string
	MaxSize     int // MB
	MaxBackups  int // files
	MaxAge      int // days
	PrettyPrint bool
}

// InitLog log/log.go
// ...
// InitLog builds two rotating JSON loggers (access & error).
// If nrApp is provided, logs include New Relic linking metadata.
// If logsCfg is provided, logs are ALSO shipped to New Relic Logs API.
func InitLog(nrApp *newrelic.Application, logsCfg *LogsExportConfig) {
	cfg := [2]Config{ /* ... same as before ... */ }

	access := newLogger(cfg[0], LogLevel, nrApp, logsCfg)          // Info+
	errorL := newLogger(cfg[1], logrus.ErrorLevel, nrApp, logsCfg) // Error+

	Logging = &Logger{Access: access, Error: errorL}
}

// ---- New Relic-linking hook (adds entity/trace/span ids to each entry) ----
type nrLinkHook struct{ app *newrelic.Application }

func (h nrLinkHook) Levels() []logrus.Level { return logrus.AllLevels }
func (h nrLinkHook) Fire(e *logrus.Entry) error {
	if h.app == nil {
		return nil
	}
	md := h.app.GetLinkingMetadata()
	if md.TraceID != "" {
		e.Data["trace.id"] = md.TraceID
	}
	if md.SpanID != "" {
		e.Data["span.id"] = md.SpanID
	}
	if md.EntityName != "" {
		e.Data["entity.name"] = md.EntityName
	}
	if md.EntityType != "" {
		e.Data["entity.type"] = md.EntityType
	}
	if md.EntityGUID != "" {
		e.Data["entity.guid"] = md.EntityGUID
	}
	if md.Hostname != "" {
		e.Data["hostname"] = md.Hostname
	}
	return nil
}

// ---- internal builder ----
// ---- internal builder ----
func newLogger(cfg Config, minLevel logrus.Level, nrApp *newrelic.Application, logsCfg *LogsExportConfig) *logrus.Logger {
	l := logrus.New()
	l.SetLevel(minLevel)
	// ... formatter + rotate hook + stdout (unchanged) ...

	// Add New Relic linking metadata (existing hook)
	if nrApp != nil {
		l.AddHook(nrLinkHook{app: nrApp})
	}

	// If logs export is configured, add the NR Logs API hook
	if logsCfg != nil && logsCfg.Endpoint != "" && logsCfg.APIKey != "" {
		l.AddHook(NewNRLogsHook(*logsCfg))
	}

	l.SetReportCaller(false)
	return l
}

// SetLevel --------- helpers ----------
func SetLevel(level logrus.Level) {
	LogLevel = level
	if Logging != nil && Logging.Access != nil {
		Logging.Access.SetLevel(level)
	}
}

func Info(args ...any) {
	if Logging != nil {
		Logging.Access.Info(args...)
	}
}
func Warn(args ...any) {
	if Logging != nil {
		Logging.Access.Warn(args...)
	}
}
func Error(args ...any) {
	if Logging != nil {
		Logging.Error.Error(args...)
	}
}
func Warnln(args ...any) {
	if Logging != nil {
		Logging.Access.Warnln(args...)
	}
}
func Errorln(args ...any) {
	if Logging != nil {
		Logging.Error.Errorln(args...)
	}
}

// Transaction ---------- transaction shim ----------
type Transaction interface {
	End()
	NoticeError(error)
	AddAttribute(string, any)
	AddAttributes(map[string]any)
	Info(msg string, kv ...any)
	Warn(msg string, kv ...any)
	Error(msg string, kv ...any)
}

type Option func(*stdTxn)

func With(k string, v any) Option       { return func(t *stdTxn) { t.AddAttribute(k, v) } }
func WithAttrs(m map[string]any) Option { return func(t *stdTxn) { t.AddAttributes(m) } }

type ctxKey string

var requestIDKey ctxKey = "request_id"

func RequestIDFromCtx(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

func StartTransaction(ctx context.Context, name string, opts ...Option) Transaction {
	t := &stdTxn{name: name, start: time.Now(), reqID: RequestIDFromCtx(ctx), fields: logrus.Fields{}}
	for _, o := range opts {
		o(t)
	}
	if Logging != nil {
		Logging.Access.WithFields(logrus.Fields{"name": name, "req_id": t.reqID}).Info("txn_start")
	}
	return t
}
func StratTransaction(ctx context.Context, name string, opts ...Option) Transaction {
	return StartTransaction(ctx, name, opts...)
}

type stdTxn struct {
	name   string
	start  time.Time
	reqID  string
	fields logrus.Fields
	err    error
}

func (t *stdTxn) NoticeError(err error) { t.err = err }
func (t *stdTxn) AddAttribute(k string, v any) {
	if t.fields == nil {
		t.fields = logrus.Fields{}
	}
	t.fields[k] = v
}
func (t *stdTxn) AddAttributes(m map[string]any) {
	if t.fields == nil {
		t.fields = logrus.Fields{}
	}
	for k, v := range m {
		t.fields[k] = v
	}
}

func (t *stdTxn) End() {
	if Logging == nil {
		return
	}
	f := logrus.Fields{"name": t.name, "dur_ms": time.Since(t.start).Milliseconds(), "status": "ok"}
	if t.reqID != "" {
		f["req_id"] = t.reqID
	}
	for k, v := range t.fields {
		f[k] = v
	}
	if t.err != nil {
		f["status"] = "error"
		f["error"] = t.err.Error()
		Logging.Error.WithFields(f).Error("txn_end")
	} else {
		Logging.Access.WithFields(f).Info("txn_end")
	}
}

func (t *stdTxn) Info(msg string, kv ...any)  { t.note("info", msg, kv...) }
func (t *stdTxn) Warn(msg string, kv ...any)  { t.note("warn", msg, kv...) }
func (t *stdTxn) Error(msg string, kv ...any) { t.note("error", msg, kv...) }

func (t *stdTxn) note(level, msg string, kv ...any) {
	if Logging == nil {
		return
	}
	f := logrus.Fields{"name": t.name, "at_ms": time.Since(t.start).Milliseconds(), "msg": msg}
	if t.reqID != "" {
		f["req_id"] = t.reqID
	}
	for i := 0; i+1 < len(kv); i += 2 {
		if k, ok := kv[i].(string); ok {
			f[k] = kv[i+1]
		}
	}
	switch level {
	case "warn":
		Logging.Access.WithFields(f).Warn("txn_note")
	case "error":
		Logging.Error.WithFields(f).Error("txn_note")
	default:
		Logging.Access.WithFields(f).Info("txn_note")
	}
}
