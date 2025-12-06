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
		go h.sendToLoki(record)
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

	record.Attrs(func(attr slog.Attr) bool {
		logEntry[attr.Key] = attr.Value.Any()
		return true
	})

	logJSON, err := json.Marshal(logEntry)
	if err != nil {
		return
	}

	payload := lokiPayload{
		Streams: []lokiStream{
			{
				Stream: map[string]string{
					"app":   h.config.AppName,
					"level": record.Level.String(),
				},
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
		return
	}

	req, err := http.NewRequest("POST", h.config.LokiURL+"/loki/api/v1/push", bytes.NewReader(body))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
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
