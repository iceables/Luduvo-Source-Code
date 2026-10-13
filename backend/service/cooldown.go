package service

import (
	"fmt"
	"math"
	"sync"
	"time"
)

type CooldownService struct {
	mu              sync.RWMutex
	lastAction      map[string]time.Time
	defaultDuration time.Duration
}

func NewCooldownService(defaultDuration time.Duration) *CooldownService {
	if defaultDuration <= 0 {
		defaultDuration = 5 * time.Second
	}
	s := &CooldownService{
		lastAction:      make(map[string]time.Time),
		defaultDuration: defaultDuration,
	}
	go s.cleanupLoop()
	return s
}

func (s *CooldownService) Allow(key string, customDuration ...time.Duration) (bool, time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	duration := s.defaultDuration
	if len(customDuration) > 0 && customDuration[0] > 0 {
		duration = customDuration[0]
	}

	now := time.Now()
	if last, exists := s.lastAction[key]; exists {
		elapsed := now.Sub(last)
		if elapsed < duration {
			remaining := duration - elapsed
			return false, remaining
		}
	}

	s.lastAction[key] = now
	return true, 0
}

func (s *CooldownService) FormatRemaining(remaining time.Duration) string {
	secs := int(math.Ceil(remaining.Seconds()))
	if secs <= 1 {
		return "1 second"
	}
	return fmt.Sprintf("%d seconds", secs)
}

func (s *CooldownService) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for key, last := range s.lastAction {
			if now.Sub(last) > 1*time.Hour {
				delete(s.lastAction, key)
			}
		}
		s.mu.Unlock()
	}
}