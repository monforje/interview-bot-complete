package metrics

import (
	"sync"
	"time"
)

type Metrics struct {
	mu                  sync.RWMutex
	InterviewsStarted   int64
	InterviewsCompleted int64
	QuestionsAsked      int64
	ProfilesGenerated   int64
	APICallsTotal       int64
	APICallsSuccessful  int64
	LastUpdateTime      time.Time
}

func NewMetrics() *Metrics {
	return &Metrics{
		LastUpdateTime: time.Now(),
	}
}

func (m *Metrics) IncrementInterviewsStarted() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.InterviewsStarted++
	m.LastUpdateTime = time.Now()
}

func (m *Metrics) IncrementInterviewsCompleted() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.InterviewsCompleted++
	m.LastUpdateTime = time.Now()
}

func (m *Metrics) IncrementQuestionsAsked() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.QuestionsAsked++
	m.LastUpdateTime = time.Now()
}

func (m *Metrics) IncrementProfilesGenerated() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ProfilesGenerated++
	m.LastUpdateTime = time.Now()
}

func (m *Metrics) IncrementAPICall(success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.APICallsTotal++
	if success {
		m.APICallsSuccessful++
	}
	m.LastUpdateTime = time.Now()
}

func (m *Metrics) GetSnapshot() Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return *m
}
