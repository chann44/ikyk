package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"
)

type LoggerConfig struct {
	Environment string
	LokiURL     string
	AppName     string
}

type Logger struct {
	*slog.Logger
	config LoggerConfig
	client *http.Client
}

type lokiStream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

type lokiPayload struct {
	Streams []lokiStream `json:"streams"`
}

type lokiHandler struct {
	handler slog.Handler
	config  LoggerConfig
	client  *http.Client
}

func (h *lokiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *lokiHandler) Handle(ctx context.Context, record slog.Record) error {
	if err := h.handler.Handle(ctx, record); err != nil {
		return err
	}

	if h.config.Environment == "production" && h.config.LokiURL != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] Sending log to Loki: %s - %s\n", record.Level, record.Message)
		go h.sendToLoki(record)
	} else {
		fmt.Fprintf(os.Stderr, "[DEBUG] NOT sending to Loki. Env=%s, LokiURL=%s\n", h.config.Environment, h.config.LokiURL)
	}

	return nil
}

func (h *lokiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &lokiHandler{
		handler: h.handler.WithAttrs(attrs),
		config:  h.config,
		client:  h.client,
	}
}

func (h *lokiHandler) WithGroup(name string) slog.Handler {
	return &lokiHandler{
		handler: h.handler.WithGroup(name),
		config:  h.config,
		client:  h.client,
	}
}

func (h *lokiHandler) sendToLoki(record slog.Record) {
	logEntry := map[string]interface{}{
		"level":   record.Level.String(),
		"message": record.Message,
		"time":    record.Time.Format(time.RFC3339),
	}

	// Extract additional attributes for better querying
	var method, status string
	record.Attrs(func(attr slog.Attr) bool {
		logEntry[attr.Key] = attr.Value.Any()
		// Extract specific fields for labels
		switch attr.Key {
		case "method":
			method = attr.Value.String()
		case "status":
			status = fmt.Sprintf("%v", attr.Value.Any())
		}
		return true
	})

	logJSON, err := json.Marshal(logEntry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal log entry: %v\n", err)
		return
	}

	// Build labels with more context
	labels := map[string]string{
		"app":         h.config.AppName,
		"level":       record.Level.String(),
		"environment": h.config.Environment,
	}

	// Add optional labels if available
	if method != "" {
		labels["method"] = method
	}
	if status != "" {
		labels["status"] = status
	}

	payload := lokiPayload{
		Streams: []lokiStream{
			{
				Stream: labels,
				Values: [][]string{
					{
						fmt.Sprintf("%d", record.Time.UnixNano()),
						string(logJSON),
					},
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal Loki payload: %v\n", err)
		return
	}

	req, err := http.NewRequest("POST", h.config.LokiURL+"/loki/api/v1/push", bytes.NewReader(body))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create Loki request: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to send to Loki: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Loki returned error %d: %s\n", resp.StatusCode, string(bodyBytes))
		return
	}
	io.Copy(io.Discard, resp.Body)
}

func NewLogger(config LoggerConfig) *Logger {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	if config.Environment == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	lokiH := &lokiHandler{
		handler: handler,
		config:  config,
		client:  client,
	}

	logger := &Logger{
		Logger: slog.New(lokiH),
		config: config,
		client: client,
	}

	return logger
}
