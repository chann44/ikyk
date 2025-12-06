package types

import (
	"net/url"
	"sync"
	"time"
)

type Service struct {
	Name      string
	URL       *url.URL
	Healthy   bool
	Mu        sync.RWMutex
	LastCheck time.Time
}

type ServiceData struct {
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Healthy   bool      `json:"healthy"`
	LastCheck time.Time `json:"last_check"`
}
