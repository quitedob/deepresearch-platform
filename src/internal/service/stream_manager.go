package service

import (
	"sync"
	"time"
)

// StreamChunk represents a chunk of streaming data
type StreamChunk struct {
	Type     string                 `json:"type"` // chunk, error, done
	Content  string                 `json:"content,omitempty"`
	Error    string                 `json:"error,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// StreamManager manages streaming connections
type StreamManager struct {
	connections sync.Map // sessionID -> *StreamConnection
	bufferSize  int
}

// StreamConnection represents a single streaming connection
type StreamConnection struct {
	sessionID string
	channel   chan StreamChunk
	done      chan struct{}
	mu        sync.Mutex
	closed    bool
}

// NewStreamManager creates a new stream manager
func NewStreamManager(bufferSize int) *StreamManager {
	if bufferSize <= 0 {
		bufferSize = 100
	}

	return &StreamManager{
		bufferSize: bufferSize,
	}
}

// CreateStream creates a new streaming connection
func (sm *StreamManager) CreateStream(sessionID string) (*StreamConnection, error) {
	conn := &StreamConnection{
		sessionID: sessionID,
		channel:   make(chan StreamChunk, sm.bufferSize),
		done:      make(chan struct{}),
		closed:    false,
	}

	sm.connections.Store(sessionID, conn)

	return conn, nil
}

// GetStream retrieves an existing stream connection
func (sm *StreamManager) GetStream(sessionID string) (*StreamConnection, bool) {
	if conn, ok := sm.connections.Load(sessionID); ok {
		return conn.(*StreamConnection), true
	}
	return nil, false
}

// SendChunk sends a chunk to a stream
func (sm *StreamManager) SendChunk(sessionID string, chunk StreamChunk) error {
	conn, ok := sm.GetStream(sessionID)
	if !ok {
		return nil // Stream doesn't exist, silently ignore
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	if conn.closed {
		return nil // Stream is closed, silently ignore
	}

	select {
	case conn.channel <- chunk:
		return nil
	case <-time.After(5 * time.Second):
		// Timeout sending, close the stream
		sm.CloseStream(sessionID)
		return nil
	}
}

// CloseStream closes a streaming connection
// 修复：添加安全检查，防止重复关闭panic
func (sm *StreamManager) CloseStream(sessionID string) error {
	conn, ok := sm.GetStream(sessionID)
	if !ok {
		return nil
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	if !conn.closed {
		conn.closed = true
		// 使用recover防止重复关闭panic
		func() {
			defer func() { recover() }()
			close(conn.channel)
		}()
		func() {
			defer func() { recover() }()
			close(conn.done)
		}()
		sm.connections.Delete(sessionID)
	}

	return nil
}

// Channel returns the channel for receiving chunks
func (sc *StreamConnection) Channel() <-chan StreamChunk {
	return sc.channel
}

// Done returns the done channel
func (sc *StreamConnection) Done() <-chan struct{} {
	return sc.done
}

// ResearchEvent represents a research progress event
type ResearchEvent struct {
	Type        string                 `json:"type"` // progress, error, completed, cancelled
	Stage       string                 `json:"stage,omitempty"`
	Progress    float32                `json:"progress,omitempty"`
	Message     string                 `json:"message"`
	TaskName    string                 `json:"task_name,omitempty"`
	TaskStatus  string                 `json:"task_status,omitempty"`
	PartialData map[string]interface{} `json:"partial_data,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

// EventStream manages research event streams
type EventStream struct {
	streams    sync.Map // sessionID -> chan *ResearchEvent
	bufferSize int
}

// NewEventStream creates a new event stream manager
func NewEventStream(bufferSize int) *EventStream {
	if bufferSize <= 0 {
		bufferSize = 100
	}

	return &EventStream{
		bufferSize: bufferSize,
	}
}

// CreateStream creates a new event stream for a session
func (es *EventStream) CreateStream(sessionID string) <-chan *ResearchEvent {
	ch := make(chan *ResearchEvent, es.bufferSize)
	es.streams.Store(sessionID, ch)
	return ch
}

// Send sends an event to a stream
func (es *EventStream) Send(sessionID string, event *ResearchEvent) {
	if ch, ok := es.streams.Load(sessionID); ok {
		eventChan := ch.(chan *ResearchEvent)
		select {
		case eventChan <- event:
		case <-time.After(5 * time.Second):
			// Timeout, close the stream
			es.CloseStream(sessionID)
		}
	}
}

// CloseStream closes an event stream
// 修复：添加安全检查，防止重复关闭
func (es *EventStream) CloseStream(sessionID string) {
	if ch, ok := es.streams.LoadAndDelete(sessionID); ok {
		eventChan := ch.(chan *ResearchEvent)
		// 使用recover防止重复关闭panic
		defer func() {
			recover()
		}()
		close(eventChan)
	}
}

// GetStream retrieves an existing event stream
func (es *EventStream) GetStream(sessionID string) (<-chan *ResearchEvent, bool) {
	if ch, ok := es.streams.Load(sessionID); ok {
		return ch.(chan *ResearchEvent), true
	}
	return nil, false
}
