package updates

import (
	"context"
	"sync"
	"time"
)

// Scheduler manages periodic and debounced Update Center refresh checks
type Scheduler struct {
	mu          sync.Mutex
	mgr         *Manager
	minInterval time.Duration
	lastRun     time.Time
	cancelFunc  context.CancelFunc
}

func NewScheduler(mgr *Manager, minInterval time.Duration) *Scheduler {
	if minInterval <= 0 {
		minInterval = 10 * time.Minute
	}
	return &Scheduler{
		mgr:         mgr,
		minInterval: minInterval,
	}
}

// CheckIfDue refreshes updates only if minInterval elapsed or force is true
func (s *Scheduler) CheckIfDue(ctx context.Context, force bool) UpdateSummary {
	s.mu.Lock()
	due := force || s.lastRun.IsZero() || time.Since(s.lastRun) >= s.minInterval
	if due {
		s.lastRun = time.Now()
	}
	s.mu.Unlock()

	if !due {
		if cached, ok := s.mgr.GetCachedSummary(); ok {
			return cached
		}
	}
	return s.mgr.CheckAllUpdates(ctx)
}

// Start launches background polling on interval and returns a stop function
func (s *Scheduler) Start(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Minute
	}
	subCtx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.cancelFunc = cancel
	s.mu.Unlock()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-subCtx.Done():
				return
			case <-ticker.C:
				s.CheckIfDue(subCtx, true)
			}
		}
	}()
}

// Stop stops background update polling
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if s.cancelFunc != nil {
		s.cancelFunc()
		s.cancelFunc = nil
	}
	s.mu.Unlock()
}

