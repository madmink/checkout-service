// log/export.go
package log

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type LogsExportConfig struct {
	Endpoint string
	APIKey   string
	Timeout  time.Duration
}

type NRLogsHook struct {
	endpoint string
	apiKey   string
	hdrName  string
	client   *http.Client
}

func headerForNRKey(k string) (string, error) {
	if k == "" {
		return "", errors.New("empty key")
	}
	if strings.HasPrefix(k, "NRAL-") || strings.HasPrefix(k, "NRII-") {
		return "X-Insert-Key", nil
	}
	if strings.HasPrefix(k, "NRAK-") || len(k) == 40 {
		return "X-License-Key", nil
	}
	return "X-Insert-Key", nil
}

func NewNRLogsHook(c LogsExportConfig) *NRLogsHook {
	if c.Timeout == 0 {
		c.Timeout = 3 * time.Second
	}

	hdr, _ := headerForNRKey(c.APIKey)

	tr := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			d := &net.Dialer{Timeout: c.Timeout}
			return d.DialContext(ctx, "tcp4", addr)
		},
		TLSHandshakeTimeout: c.Timeout,
		MaxIdleConns:        50,
		IdleConnTimeout:     30 * time.Second,
	}

	return &NRLogsHook{
		endpoint: c.Endpoint,
		apiKey:   c.APIKey,
		hdrName:  hdr,
		client:   &http.Client{Transport: tr, Timeout: c.Timeout + 2*time.Second},
	}
}

func (h *NRLogsHook) Levels() []logrus.Level { return logrus.AllLevels }

func (h *NRLogsHook) Fire(e *logrus.Entry) error {
	if h.endpoint == "" || h.apiKey == "" || h.hdrName == "" {
		return nil // disabled
	}

	payload := []map[string]any{{
		"timestamp":  e.Time.UnixMilli(),
		"message":    e.Message,
		"level":      e.Level.String(),
		"attributes": e.Data,
	}}

	b, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", h.endpoint, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(h.hdrName, h.apiKey)

	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return errors.New("nr logs http " + resp.Status)
	}
	return nil
}
