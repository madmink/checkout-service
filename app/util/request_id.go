package util

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
)

type requestIDKey string

const RequestIDHeader = "Log-Request-ID"

func GenerateRequestID() string {
	bytes := make([]byte, 32)

	if _, err := rand.Read(bytes); err != nil {
		return ""
	}

	return base64.URLEncoding.EncodeToString(bytes)
}

func GetRequestID(ctx context.Context) string {
	requestID, ok := ctx.Value(requestIDKey(RequestIDHeader)).(string)
	if !ok {
		return ""
	}

	return requestID
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey(RequestIDHeader), requestID)
}

func EnsureRequestIDFromRequest(r *http.Request) (*http.Request, string) {
	requestID := r.Header.Get(RequestIDHeader)

	if requestID == "" {
		requestID = GetRequestID(r.Context())
	}

	if requestID == "" {
		requestID = GenerateRequestID()
	}

	ctx := WithRequestID(r.Context(), requestID)

	return r.WithContext(ctx), requestID
}

func RequestIDLog(ctx context.Context) string {
	return fmt.Sprintf("RequestID : %s : ", GetRequestID(ctx))
}
