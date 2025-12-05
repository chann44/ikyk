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
