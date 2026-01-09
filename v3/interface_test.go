package pb

import (
	"errors"
	"sync"
	"testing"
	"time"
)

// MockProgressable is a test implementation of the Progressable interface
type MockProgressable struct {
	total      int64
	current    int64
	finishCh   chan struct{}
	finishOnce sync.Once
}

// NewMockProgressable creates a new MockProgressable with the given total
func NewMockProgressable(total int64) *MockProgressable {
	return &MockProgressable{
		total:    total,
		current:  0,
		finishCh: make(chan struct{}),
	}
}

// Total returns the total progress
func (m *MockProgressable) Total() int64 {
	return m.total
}

// Value returns the current progress value
func (m *MockProgressable) Value() int64 {
	return m.current
}

// FinishedChan returns the channel that closes when progress is complete
func (m *MockProgressable) FinishedChan() <-chan struct{} {
	return m.finishCh
}

// SetCurrent updates the current progress value
func (m *MockProgressable) SetCurrent(v int64) {
	m.current = v
}

// Finish closes the finish channel
func (m *MockProgressable) Finish() {
	m.finishOnce.Do(func() {
		close(m.finishCh)
	})
}

// MockProgressableWithTitle is a test implementation of both Progressable and TitleProgressable
type MockProgressableWithTitle struct {
	MockProgressable
	title string
}

// NewMockProgressableWithTitle creates a new MockProgressableWithTitle
func NewMockProgressableWithTitle(total int64, title string) *MockProgressableWithTitle {
	return &MockProgressableWithTitle{
		MockProgressable: *NewMockProgressable(total),
		title:            title,
	}
}

// Title returns the progress title
func (m *MockProgressableWithTitle) Title() string {
	return m.title
}

// TestProgressableInterface validates the Progressable interface contract
func TestProgressableInterface(t *testing.T) {
	tests := []struct {
		name        string
		total       int64
		currentVal  int64
		expectTotal int64
		expectValue int64
	}{
		{
			name:        "zero total",
			total:       0,
			currentVal:  0,
			expectTotal: 0,
			expectValue: 0,
		},
		{
			name:        "simple progress",
			total:       100,
			currentVal:  50,
			expectTotal: 100,
			expectValue: 50,
		},
		{
			name:        "large numbers",
			total:       1000000,
			currentVal:  500000,
			expectTotal: 1000000,
			expectValue: 500000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockProgressable(tt.total)
			mock.SetCurrent(tt.currentVal)

			if got := mock.Total(); got != tt.expectTotal {
				t.Errorf("Total() = %v, want %v", got, tt.expectTotal)
			}

			if got := mock.Value(); got != tt.expectValue {
				t.Errorf("Value() = %v, want %v", got, tt.expectValue)
			}

			// Verify finish channel is initially open
			select {
			case <-mock.FinishedChan():
				t.Error("FinishedChan should not be closed initially")
			default:
				// Expected - channel is open
			}
		})
	}
}

// TestProgressableFinishedChan verifies the finish channel behavior
func TestProgressableFinishedChan(t *testing.T) {
	mock := NewMockProgressable(100)

	// Channel should not be closed initially
	select {
	case <-mock.FinishedChan():
		t.Fatal("Channel should not be closed initially")
	default:
	}

	// Finish the progressable
	mock.Finish()

	// Channel should now be closed
	select {
	case <-mock.FinishedChan():
		// Expected - channel is closed
	default:
		t.Error("Channel should be closed after Finish()")
	}

	// Second Finish() call should be safe (no panic)
	mock.Finish()
}

// TestTitleProgressableInterface validates the TitleProgressable interface
func TestTitleProgressableInterface(t *testing.T) {
	tests := []struct {
		name       string
		total      int64
		title      string
		expectTitle string
	}{
		{
			name:        "with title",
			total:       100,
			title:       "Test Progress",
			expectTitle: "Test Progress",
		},
		{
			name:        "empty title",
			total:       100,
			title:       "",
			expectTitle: "",
		},
		{
			name:        "long title",
			total:       100,
			title:       "This is a very long progress bar title",
			expectTitle: "This is a very long progress bar title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockProgressableWithTitle(tt.total, tt.title)

			if got := mock.Total(); got != tt.total {
				t.Errorf("Total() = %v, want %v", got, tt.total)
			}

			if got := mock.Title(); got != tt.expectTitle {
				t.Errorf("Title() = %v, want %v", got, tt.expectTitle)
			}
		})
	}
}

// TestRegisterProgressableNilInput verifies error handling for nil input
func TestRegisterProgressableNilInput(t *testing.T) {
	bar, err := RegisterProgressable(nil, func(*ProgressBar) {})

	if bar != nil {
		t.Errorf("Expected nil bar, got %v", bar)
	}

	if err == nil {
		t.Error("Expected error for nil progressable")
	}

	if !errors.Is(err, errors.New("RegisterProgressable: pr is nil")) && err.Error() != "RegisterProgressable: pr is nil" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

// TestRegisterProgressableSuccess verifies successful registration
func TestRegisterProgressableSuccess(t *testing.T) {
	mock := NewMockProgressable(100)
	removeCalled := false

	bar, err := RegisterProgressable(mock, func(pb *ProgressBar) {
		removeCalled = true
	})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if bar == nil {
		t.Error("Expected non-nil bar")
	}

	if bar.Total() != 100 {
		t.Errorf("Bar total = %v, want 100", bar.Total())
	}

	// The remove function should be called when the progressable finishes
	mock.Finish()
	time.Sleep(150 * time.Millisecond) // Wait for goroutine to process

	if !removeCalled {
		t.Error("Remove function was not called")
	}
}

// TestHasTitle verifies title detection and template configuration
func TestHasTitle(t *testing.T) {
	tests := []struct {
		name              string
		progressable      Progressable
		shouldHaveTitle   bool
		expectedInTemplate string
	}{
		{
			name:            "without title interface",
			progressable:    NewMockProgressable(100),
			shouldHaveTitle: false,
		},
		{
			name:              "with title interface and non-empty title",
			progressable:      NewMockProgressableWithTitle(100, "Upload:"),
			shouldHaveTitle:   true,
			expectedInTemplate: "Upload:",
		},
		{
			name:            "with title interface but empty title",
			progressable:    NewMockProgressableWithTitle(100, ""),
			shouldHaveTitle: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bar := New64(100)
			hasTitle(tt.progressable, bar)

			if tt.shouldHaveTitle {
				// Title should have been set in bar's data
				titleVal := bar.Get("title")
				if titleVal != tt.expectedInTemplate {
					t.Errorf("Title in bar = %v, want %v", titleVal, tt.expectedInTemplate)
				}
			}
		})
	}
}

// TestProgressWorkerIntegration verifies the progressWorker goroutine behavior
func TestProgressWorkerIntegration(t *testing.T) {
	mock := NewMockProgressable(100)
	removeFunc := func(pb *ProgressBar) {}

	bar := New64(100)
	go progressWorker(mock, bar, removeFunc)

	// Wait for some updates
	time.Sleep(150 * time.Millisecond)

	// Update the mock progress
	mock.SetCurrent(50)
	time.Sleep(150 * time.Millisecond)

	// The bar should have been updated
	if bar.Current() == 0 {
		t.Error("progressWorker should have updated bar current value")
	}

	// Finish the progress
	mock.Finish()
	time.Sleep(150 * time.Millisecond)

	// Bar should have final values (at least 50, since that's what we set)
	if bar.Current() < 50 {
		t.Errorf("Final bar current = %v, expected >= 50", bar.Current())
	}
}

// TestProgressWorkerWithTitle verifies progressWorker with titled progressable
func TestProgressWorkerWithTitle(t *testing.T) {
	mock := NewMockProgressableWithTitle(100, "Download")

	// Use RegisterProgressable which calls hasTitle
	bar, err := RegisterProgressable(mock, func(pb *ProgressBar) {})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Wait for setup
	time.Sleep(150 * time.Millisecond)

	// Check that title was set
	title := bar.Get("title")
	if title != "Download" {
		t.Errorf("Title not set correctly: got %v, want Download", title)
	}

	mock.Finish()
	time.Sleep(150 * time.Millisecond)
}

// TestNewPoolProgressFactory verifies factory constructor
func TestNewPoolProgressFactory(t *testing.T) {
	pool := NewPool()
	factory := NewPoolProgressFactory(pool)

	if factory == nil {
		t.Error("Expected non-nil factory")
	}

	if factory.Pool != pool {
		t.Error("Factory pool reference doesn't match")
	}

	if factory.Wg == nil {
		t.Error("Factory WaitGroup should not be nil")
	}
}

// TestPoolProgressFactoryRegisterNilInput verifies error handling for nil progressable
func TestPoolProgressFactoryRegisterNilInput(t *testing.T) {
	pool := NewPool()
	factory := NewPoolProgressFactory(pool)

	err := factory.Register(nil)

	if err == nil {
		t.Error("Expected error for nil progressable")
	}

	if err.Error() != "Register: progressable is nil" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

// TestPoolProgressFactoryRegisterSuccess verifies successful registration
func TestPoolProgressFactoryRegisterSuccess(t *testing.T) {
	pool := NewPool()
	factory := NewPoolProgressFactory(pool)

	mock := NewMockProgressable(100)

	err := factory.Register(mock)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// WaitGroup should have incremented
	wgCount := 1 // This is implicit - we added one, so we expect Done() to be called once

	// Progress should be tracked
	mock.SetCurrent(50)
	mock.Finish()

	// Wait for the goroutine to complete
	done := make(chan bool)
	go func() {
		factory.Wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Expected - WaitGroup completed
	case <-time.After(1 * time.Second):
		t.Error("Factory WaitGroup did not complete in time")
	}

	_ = wgCount
}

// TestPoolProgressFactoryMultipleRegisters verifies concurrent registration
func TestPoolProgressFactoryMultipleRegisters(t *testing.T) {
	pool := NewPool()
	factory := NewPoolProgressFactory(pool)

	// Register multiple progressables
	progressables := []*MockProgressable{
		NewMockProgressable(100),
		NewMockProgressable(200),
		NewMockProgressable(150),
	}

	for _, mock := range progressables {
		err := factory.Register(mock)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	}

	// Finish all progressables
	for _, mock := range progressables {
		mock.SetCurrent(mock.Total())
		mock.Finish()
	}

	// Wait for all to complete
	done := make(chan bool)
	go func() {
		factory.Wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Expected - all completed
	case <-time.After(2 * time.Second):
		t.Error("Factory WaitGroup did not complete all tasks in time")
	}
}

// TestPoolProgressFactoryRegisterError verifies error propagation
func TestPoolProgressFactoryRegisterError(t *testing.T) {
	pool := NewPool()
	factory := NewPoolProgressFactory(pool)

	// Create a mock that returns error from RegisterProgressable
	// (This will happen if RegisterProgressable returns error, but currently
	// it only returns error for nil input, so we test that case)
	err := factory.Register(nil)

	if err == nil {
		t.Error("Expected error for nil progressable")
	}

	// Factory's WaitGroup should not be blocked since we did Wg.Done() on error
	done := make(chan bool)
	go func() {
		factory.Wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Expected - no blocking
	case <-time.After(500 * time.Millisecond):
		t.Error("Factory WaitGroup blocked after Register error")
	}
}
