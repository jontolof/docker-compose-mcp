package session

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Manager struct {
	sessions map[string]*Session
	mutex    sync.RWMutex
}

type Session struct {
	ID        string
	Type      string
	StartTime time.Time
	Context   context.Context
	Cancel    context.CancelFunc
	Output    chan string
	Done      chan struct{}
}

func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]*Session),
	}
}

func (m *Manager) CreateSession(sessionType string) *Session {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	id := fmt.Sprintf("%s_%d", sessionType, time.Now().UnixNano())
	ctx, cancel := context.WithCancel(context.Background())

	session := &Session{
		ID:        id,
		Type:      sessionType,
		StartTime: time.Now(),
		Context:   ctx,
		Cancel:    cancel,
		Output:    make(chan string, 100),
		Done:      make(chan struct{}),
	}

	m.sessions[id] = session
	return session
}

func (m *Manager) GetSession(id string) (*Session, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	session, exists := m.sessions[id]
	return session, exists
}

func (m *Manager) StopSession(id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	session, exists := m.sessions[id]
	if !exists {
		return fmt.Errorf("session not found: %s", id)
	}

	session.Cancel()
	close(session.Done)
	delete(m.sessions, id)

	return nil
}

func (m *Manager) ListSessions() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var ids []string
	for id := range m.sessions {
		ids = append(ids, id)
	}
	return ids
}