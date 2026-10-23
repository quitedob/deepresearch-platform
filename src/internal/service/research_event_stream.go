package service

import (
	"context"
	"sync"
	"time"

	"github.com/ai-research-platform/internal/types/constant"
)

const defaultResearchEventBufferSize = 200

type researchSubscription struct {
	events chan *ResearchEvent
	done   chan struct{}
}

// EventStream broadcasts progress to independent subscribers. Sending and closing
// share a lock so disconnects cannot race with a producer sending on the channel.
type EventStream struct {
	mutex      sync.Mutex
	streams    map[string]map[*researchSubscription]struct{}
	bufferSize int
}

func NewEventStream(bufferSize int) *EventStream {
	if bufferSize <= 0 {
		bufferSize = defaultResearchEventBufferSize
	}
	return &EventStream{bufferSize: bufferSize, streams: make(map[string]map[*researchSubscription]struct{})}
}

func (stream *EventStream) CreateStream(sessionID string) <-chan *ResearchEvent {
	events, _ := stream.subscribe(context.Background(), sessionID)
	return events
}

func (stream *EventStream) subscribe(ctx context.Context, sessionID string) (<-chan *ResearchEvent, func()) {
	subscription := &researchSubscription{events: make(chan *ResearchEvent, stream.bufferSize), done: make(chan struct{})}
	stream.mutex.Lock()
	if stream.streams[sessionID] == nil {
		stream.streams[sessionID] = make(map[*researchSubscription]struct{})
	}
	stream.streams[sessionID][subscription] = struct{}{}
	stream.mutex.Unlock()
	unsubscribe := func() {
		stream.mutex.Lock()
		defer stream.mutex.Unlock()
		stream.closeSubscriptionLocked(sessionID, subscription)
	}
	go func() {
		timer := time.NewTimer(constant.ResearchSSETimeout)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			unsubscribe()
		case <-timer.C:
			unsubscribe()
		case <-subscription.done:
		}
	}()
	return subscription.events, unsubscribe
}

func (stream *EventStream) closeSubscriptionLocked(sessionID string, subscription *researchSubscription) {
	if _, exists := stream.streams[sessionID][subscription]; !exists {
		return
	}
	delete(stream.streams[sessionID], subscription)
	close(subscription.events)
	close(subscription.done)
	if len(stream.streams[sessionID]) == 0 {
		delete(stream.streams, sessionID)
	}
}

func (stream *EventStream) Send(sessionID string, event *ResearchEvent) {
	stream.mutex.Lock()
	defer stream.mutex.Unlock()
	for subscription := range stream.streams[sessionID] {
		select {
		case subscription.events <- event:
		default:
			// A slow subscriber must not block research or another subscriber. The
			// service continues checking persisted terminal state after overflow.
			stream.closeSubscriptionLocked(sessionID, subscription)
		}
	}
}

func (stream *EventStream) CloseStream(sessionID string) {
	stream.mutex.Lock()
	defer stream.mutex.Unlock()
	for subscription := range stream.streams[sessionID] {
		stream.closeSubscriptionLocked(sessionID, subscription)
	}
}

func (stream *EventStream) GetStream(sessionID string) (<-chan *ResearchEvent, bool) {
	stream.mutex.Lock()
	defer stream.mutex.Unlock()
	for subscription := range stream.streams[sessionID] {
		return subscription.events, true
	}
	return nil, false
}
