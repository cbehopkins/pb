package main

import (
	"log"
	"time"

	pb "github.com/cbehopkins/pb/v3"
)

type MockProgressable struct {
	total      int64
	value      int64
	title      string
	finishedCh chan struct{}
}

func NewMockProgressable(total int64) *MockProgressable {
	m := &MockProgressable{
		total:      total,
		value:      0,
		title:      "",
		finishedCh: make(chan struct{}),
	}
	go m.worker()
	return m
}
func (m *MockProgressable) worker() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	defer close(m.finishedCh)
	for range ticker.C {
		if m.value >= m.total {
			return
		}
		m.value += int64(10 + (time.Now().UnixNano() % 20))
		if m.value > m.total {
			m.value = m.total
		}
	}
}
func (m *MockProgressable) Total() int64 {
	return m.total
}
func (m *MockProgressable) Value() int64 {
	return m.value
}
func (m *MockProgressable) FinishedChan() <-chan struct{} {
	return m.finishedCh
}
func (m *MockProgressable) Title() string {
	return m.title
}
func (m *MockProgressable) SetTitle(title string) {
	m.title = title
}
func main() {
	// Create a pool for managing progress bars
	pool := pb.NewPool()
	if err := pool.Start(); err != nil {
		log.Fatalf("Failed to start progress bar pool: %v", err)
	}

	// Create a factory for managing progress bars
	factory := pb.NewPoolProgressFactory(pool)

	// Create and register multiple mock progressables
	// Mock item 1: Fast counter (100 total)
	mock1 := NewMockProgressable(100)
	if err := factory.Register(mock1); err != nil {
		log.Fatalf("Failed to register mock1: %v", err)
	}

	// Wait a second before adding the next one
	time.Sleep(1 * time.Second)

	// Mock item 2: Slower counter (200 total)
	mock2 := NewMockProgressable(200)
	mock2.SetTitle("Downloading File A:")
	if err := factory.Register(mock2); err != nil {
		log.Fatalf("Failed to register mock2: %v", err)
	}

	// Wait another second before adding the third one
	time.Sleep(1 * time.Second)

	// Mock item 3: Medium counter (150 total)
	mock3 := NewMockProgressable(150)
	if err := factory.Register(mock3); err != nil {
		log.Fatalf("Failed to register mock3: %v", err)
	}

	// Wait for all progress bars to complete
	factory.Wg.Wait()

	// Stop the pool to clean up the progress bar display
	if err := pool.Stop(); err != nil {
		log.Printf("Failed to stop pool: %v", err)
	}
}
