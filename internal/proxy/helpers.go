package proxy

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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
	// CJK characters typically use 2-3 tokens each vs ~0.25 for ASCII.
	// Count non-ASCII runes more heavily.
	tokenEstimate := 0
	for _, r := range text {
		if r > 127 {
			tokenEstimate += 2 // CJK characters roughly 2 tokens each
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

func writeUpstreamError(w http.ResponseWriter, status int, body []byte) {
	if len(body) == 0 {
		writeError(w, status, fmt.Errorf("upstream returned %d", status))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Printf("%s %s status=%d %s", r.Method, r.URL.RequestURI(), rec.status, time.Since(start).Round(time.Millisecond))
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