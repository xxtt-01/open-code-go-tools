package proxy

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/ethan-blue/open-code-go-tools/internal/config"
)

const maxReasoningEntries = 1024

// MaxBodySize is the maximum allowed request body size (10 MB).
const MaxBodySize int64 = 10 << 20

type requestLogEntry struct {
	ID       string    `json:"id"`
	Time     time.Time `json:"time"`
	Method   string    `json:"method"`
	Path     string    `json:"path"`
	Status   int       `json:"status"`
	Duration string    `json:"duration"`
	Model    string    `json:"model"`
	Route    string    `json:"route"`
	Error    string    `json:"error,omitempty"`
}

type Server struct {
	config     config.Config
	configPath string
	client     *http.Client
	upstream   string

	configMu sync.RWMutex // Protects config, upstream, and client.Timeout

	reasoningMu     sync.Mutex
	reasoningByTool map[string]string
	reasoningOrder  []string

	historyMu sync.RWMutex
	history   []requestLogEntry

	// Circuit breaker state
	circuitMu           sync.Mutex
	consecutiveFailures map[string]int
	trippedUntil        map[string]time.Time

	// Auto-generated auth token when config.LocalAuthToken is empty
	// Protects the dashboard API from in-network access by default
	autoAuthToken string
	autoAuthOnce  sync.Once

	// Tracks in-flight streaming requests for graceful shutdown.
	// On shutdown, ListenAndServe waits for all tracked requests to complete.
	wg sync.WaitGroup
}

func (s *Server) SetConfigPath(path string) {
	s.configPath = path
}

type anthropicRequest struct {
	Model         string          `json:"model"`
	MaxTokens     int             `json:"max_tokens,omitempty"`
	Messages      []anthropicMsg  `json:"messages"`
	System        any             `json:"system,omitempty"`
	Stream        bool            `json:"stream,omitempty"`
	Temperature   *float64        `json:"temperature,omitempty"`
	TopP          *float64        `json:"top_p,omitempty"`
	StopSequences []string        `json:"stop_sequences,omitempty"`
	Thinking      any             `json:"thinking,omitempty"`
	Tools         []anthropicTool `json:"tools,omitempty"`
	ToolChoice    any             `json:"tool_choice,omitempty"`
}

type anthropicMsg struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type anthropicTool struct {
	Type        string         `json:"type,omitempty"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"input_schema"`
}

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Stream      bool            `json:"stream,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature *float64        `json:"temperature,omitempty"`
	TopP        *float64        `json:"top_p,omitempty"`
	Stop        []string        `json:"stop,omitempty"`
	Tools       []openAITool    `json:"tools,omitempty"`
	ToolChoice  any             `json:"tool_choice,omitempty"`
	Thinking    any             `json:"thinking,omitempty"`
	// DeepSeek-compatible thinking effort. Only set for models that advertise this parameter.
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
}

type openAIMessage struct {
	Role             string     `json:"role"`
	Content          any        `json:"content,omitempty"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	ToolCallID       string     `json:"tool_call_id,omitempty"`
	ToolCalls        []toolCall `json:"tool_calls,omitempty"`
	Name             string     `json:"name,omitempty"`
}

type openAITool struct {
	Type     string         `json:"type"`
	Function openAIFunction `json:"function"`
}

type openAIFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters"`
}

type openAIResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content          string     `json:"content"`
			ReasoningContent any        `json:"reasoning_content"`
			ThinkingContent  any        `json:"thinking_content"`  // compatibility
			Thinking         any        `json:"thinking"`          // compatibility
			Reasoning        any        `json:"reasoning"`         // compatibility
			ReasoningDetails any        `json:"reasoning_details"` // compatibility
			ToolCalls        []toolCall `json:"tool_calls"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage openAIUsage `json:"usage"`
}

type openAIChunk struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Delta struct {
			Content          string     `json:"content"`
			ReasoningContent any        `json:"reasoning_content"`
			ThinkingContent  any        `json:"thinking_content"`  // compatibility
			Thinking         any        `json:"thinking"`          // compatibility
			Reasoning        any        `json:"reasoning"`         // compatibility
			ReasoningDetails any        `json:"reasoning_details"` // compatibility
			ToolCalls        []toolCall `json:"tool_calls"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage openAIUsage `json:"usage"`
}

type toolCall struct {
	ID       string `json:"id,omitempty"`
	Type     string `json:"type,omitempty"`
	Function struct {
		Name      string `json:"name,omitempty"`
		Arguments string `json:"arguments,omitempty"`
	} `json:"function"`
	Index *int `json:"index,omitempty"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type anthropicError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type anthropicErrorResponse struct {
	Error anthropicError `json:"error"`
}

func New(cfg config.Config) (*Server, error) {
	transport := &http.Transport{
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       90 * time.Second,
		ForceAttemptHTTP2:     true,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 60 * time.Second,
	}
	return &Server{
		config:              cfg,
		upstream:            cfg.Upstream,
		client:              &http.Client{Timeout: cfg.RequestTimeout(), Transport: transport},
		reasoningByTool:     map[string]string{},
		consecutiveFailures: map[string]int{},
		trippedUntil:        map[string]time.Time{},
	}, nil
}

func (s *Server) Config() *config.Config {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return &s.config
}

func (s *Server) ApplyConfig(cfg config.Config) {
	s.configMu.Lock()
	defer s.configMu.Unlock()
	s.config = cfg
	s.upstream = cfg.Upstream
	if s.client != nil {
		s.client.Timeout = cfg.RequestTimeout()
	}
}

func (s *Server) ListenAddress() string {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return s.config.Listen
}

func (s *Server) recordModelSuccess(model string) {
	s.circuitMu.Lock()
	defer s.circuitMu.Unlock()
	s.consecutiveFailures[model] = 0
	delete(s.trippedUntil, model)
}

func (s *Server) recordModelFailure(model string) {
	s.circuitMu.Lock()
	defer s.circuitMu.Unlock()
	s.consecutiveFailures[model]++
	if s.consecutiveFailures[model] >= 3 {
		s.trippedUntil[model] = time.Now().Add(30 * time.Second)
		log.Printf("[CircuitBreaker] Model %q has failed consecutively %d times. Tripped for 30 seconds.", model, s.consecutiveFailures[model])
	}
}

func (s *Server) isModelCircuitTripped(model string) bool {
	s.circuitMu.Lock()
	defer s.circuitMu.Unlock()
	until, ok := s.trippedUntil[model]
	if !ok {
		return false
	}
	if time.Now().After(until) {
		// Cooldown period expired, reset failure counter and untrip
		s.consecutiveFailures[model] = 0
		delete(s.trippedUntil, model)
		return false
	}
	return true
}
