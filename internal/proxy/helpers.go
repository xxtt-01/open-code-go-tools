package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

func copyHeaders(dst, src http.Header) {
	for k, values := range src {
		dst.Del(k)
		for _, v := range values {
			dst.Add(k, v)
		}
	}
}

func stripHopByHopHeaders(h http.Header) {
	for _, key := range []string{"Connection", "Keep-Alive", "Proxy-Authenticate", "Proxy-Authorization", "Te", "Trailer", "Transfer-Encoding", "Upgrade"} {
		h.Del(key)
	}
}

func copyResponse(w http.ResponseWriter, body io.Reader) (int64, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return io.Copy(w, body)
	}
	buf := make([]byte, 32*1024)
	var written int64
	for {
		n, readErr := body.Read(buf)
		if n > 0 {
			m, writeErr := w.Write(buf[:n])
			written += int64(m)
			flusher.Flush()
			if writeErr != nil {
				return written, writeErr
			}
			if m != n {
				return written, io.ErrShortWrite
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				return written, nil
			}
			return written, readErr
		}
	}
}

func estimateTokens(payload anthropicRequest) int {
	text := payload.Model + "\n" + blocksToText(payload.System)
	for _, msg := range payload.Messages {
		text += "\n" + msg.Role + ":" + blocksToText(msg.Content)
	}
	for _, tool := range payload.Tools {
		text += "\n" + tool.Name + ":" + tool.Description
	}
	// CJK characters typically use 2-3 tokens each vs ~4 chars per token for ASCII.
	// Count non-ASCII runes more heavily.
	tokenEstimate := 0
	for _, r := range text {
		if r > 127 {
			tokenEstimate += 3 // CJK characters roughly 2-3 tokens each
		} else {
			tokenEstimate++ // ASCII characters, ~4 per token
		}
	}
	return tokenEstimate/4 + 1
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"type": "ocgt_error", "message": err.Error()}})
}

func proxyErrorStatus(err error) int {
	var netErr net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) {
		return http.StatusGatewayTimeout
	}
	return http.StatusBadGateway
}

func writeProxyError(w http.ResponseWriter, err error) {
	writeError(w, proxyErrorStatus(err), err)
}

func writeUpstreamError(w http.ResponseWriter, status int, body []byte) {
	if len(body) == 0 {
		writeError(w, status, fmt.Errorf("upstream returned %d", status))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func upstreamErrorSummary(status int, body []byte) string {
	if len(body) == 0 {
		return fmt.Sprintf("upstream returned %d", status)
	}
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err == nil {
		if msg := nestedString(raw, "error", "message"); msg != "" {
			return msg
		}
		if msg := nestedString(raw, "error", "type"); msg != "" {
			return msg
		}
		if msg, _ := raw["message"].(string); msg != "" {
			return msg
		}
		if msg, _ := raw["error"].(string); msg != "" {
			return msg
		}
	}
	text := strings.TrimSpace(string(body))
	if len(text) > 240 {
		text = text[:240] + "..."
	}
	return text
}

func nestedString(raw map[string]any, keys ...string) string {
	var cur any = raw
	for _, key := range keys {
		obj, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur = obj[key]
	}
	value, _ := cur.(string)
	return value
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Enable CORS for frontend API requests
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Ocgt-Profile, X-Ocgt-Local-Token")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "*")
			w.Header().Set("Access-Control-Allow-Methods", "*")
		}

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Printf("%s %s status=%d %s", r.Method, r.URL.RequestURI(), rec.status, time.Since(start).Round(time.Millisecond))
	})
}

// authMiddleware checks for local auth token if configured
func authMiddleware(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health check and OPTIONS
		if r.URL.Path == "/healthz" || r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}

		// Skip auth for static web assets
		if strings.HasPrefix(r.URL.Path, "/ocgt/web/") || r.URL.Path == "/" {
			next.ServeHTTP(w, r)
			return
		}

		// Check auth token if configured
		if token != "" {
			providedToken := r.Header.Get("X-Ocgt-Local-Token")
			if providedToken == "" {
				// Also check Authorization header with Bearer prefix
				authHeader := r.Header.Get("Authorization")
				if strings.HasPrefix(authHeader, "Bearer ") {
					providedToken = strings.TrimPrefix(authHeader, "Bearer ")
				}
			}
			if providedToken != token {
				writeError(w, http.StatusUnauthorized, errors.New("invalid or missing auth token"))
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
