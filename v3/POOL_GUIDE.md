# Progress Bar Pool Guide

## Overview

The `pb/v3` package provides a robust system for managing multiple progress bars simultaneously. The key components are:

1. **Pool** - Manages and displays multiple progress bars in sync
2. **ProgressBar** - Individual progress bar with customizable templates
3. **Progressable Interface** - Abstraction for progress sources
4. **PoolProgressFactory** - Convenience factory for registering progress sources with the pool

## Core Interfaces

### Progressable

```go
type Progressable interface {
	Total() int64
	Value() int64
	FinishedChan() <-chan struct{}
}
```

The basic interface for any source of progress data. Implement this to track progress of any operation:
- `Total()` - Returns the total work units
- `Value()` - Returns the current progress (0 to Total)
- `FinishedChan()` - Returns a channel that closes when progress is complete

### TitleProgressable

```go
type TitleProgressable interface {
	Progressable
	Title() string
}
```

Optional interface that extends `Progressable`. If your progressable implements this, the title will automatically be displayed in the progress bar. Titles are optional - progressables that don't implement this work fine without them.

## Pool Management

### Creating a Pool

```go
pool := pb.NewPool()
if err := pool.Start(); err != nil {
	log.Fatal(err)
}
defer pool.Stop()
```

### PoolProgressFactory

The factory simplifies registering progressables with a pool:

```go
wg := &sync.WaitGroup{}
factory := &pb.PoolProgressFactory{
	Pool: pool,
	Wg:   wg,  // WaitGroup to track completion
}

// Register a progressable
factory.Register(myProgressable)

// Wait for all to complete
wg.Wait()
```

The factory automatically:
- Adds the progressable to the pool
- Tracks completion with a WaitGroup
- Configures the progress bar appropriately
- Handles template setup (including titles if available)

## Working with Progressables

### Basic Example

Implement the `Progressable` interface:

```go
type MyTask struct {
	total      int64
	current    int64
	finishedCh chan struct{}
}

func (t *MyTask) Total() int64 {
	return t.total
}

func (t *MyTask) Value() int64 {
	return t.current
}

func (t *MyTask) FinishedChan() <-chan struct{} {
	return t.finishedCh
}
```

### With Optional Title

To add a title, also implement `TitleProgressable`:

```go
func (t *MyTask) Title() string {
	return "Processing: " + t.name
}
```

The pool will automatically detect this and include the title in the display.

## Template Customization

By default, titled progressables use this template:
```
{{string . "title"}} {{counters . }} {{bar . }} {{percent . }} {{speed . }}
```

You can customize individual progress bars after registration:

```go
bar := pb.RegisterProgressable(myProgressable, removeFunc)
bar.SetTemplateString(`[Custom] {{counters . }} {{bar . }} {{percent . }}`)
pool.Add(bar)
```

### Available Template Variables

- `{{counters .}}` - Shows "current / total"
- `{{bar .}}` - The visual progress bar
- `{{percent .}}` - Percentage completion
- `{{speed .}}` - Rate of progress (units/sec)
- `{{rtime . "format"}}` - Remaining time with custom format
- `{{string . "key"}}` - Custom string data set via `bar.Set("key", value)`

### Template Functions

- Colors: `{{red "text"}}`, `{{green "text"}}`, `{{blue "text"}}`, etc.
- All standard Go text/template functions

Example with color:
```go
bar.SetTemplateString(`{{green "✓"}} {{counters . }} {{bar . }}`)
```

## Complete Example

```go
package main

import (
	"log"
	"sync"
	"time"

	pb "github.com/cbehopkins/pb/v3"
)

type FileDownload struct {
	filename   string
	total      int64
	current    int64
	finishedCh chan struct{}
}

func NewFileDownload(name string, size int64) *FileDownload {
	f := &FileDownload{
		filename:   name,
		total:      size,
		finishedCh: make(chan struct{}),
	}
	go f.simulateDownload()
	return f
}

func (f *FileDownload) Total() int64 {
	return f.total
}

func (f *FileDownload) Value() int64 {
	return f.current
}

func (f *FileDownload) FinishedChan() <-chan struct{} {
	return f.finishedCh
}

func (f *FileDownload) Title() string {
	return f.filename
}

func (f *FileDownload) simulateDownload() {
	defer close(f.finishedCh)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for range ticker.C {
		f.current += int64(1000 + time.Now().UnixNano()%2000)
		if f.current >= f.total {
			f.current = f.total
			return
		}
	}
}

func main() {
	pool := pb.NewPool()
	if err := pool.Start(); err != nil {
		log.Fatal(err)
	}
	defer pool.Stop()

	wg := &sync.WaitGroup{}
	factory := &pb.PoolProgressFactory{
		Pool: pool,
		Wg:   wg,
	}

	// Start multiple downloads
	factory.Register(NewFileDownload("document.pdf", 50000000))
	time.Sleep(500 * time.Millisecond)
	factory.Register(NewFileDownload("image.zip", 100000000))
	time.Sleep(500 * time.Millisecond)
	factory.Register(NewFileDownload("video.mp4", 500000000))

	// Wait for all to complete
	wg.Wait()
	log.Println("All downloads complete!")
}
```

## Key Features

### Automatic Title Support
- If your progressable implements `TitleProgressable`, the title is automatically included
- Titles are optional - leave `Title()` returning empty string to hide it
- Titles are displayed at the start of the progress bar line

### Synchronized Display
- All progress bars are displayed together
- Updates are synchronized with configurable refresh rate
- Proper terminal handling for clean display

### Flexible Progress Tracking
- Works with any operation that can report progress
- Goroutine-safe
- Efficient with minimal overhead

### WaitGroup Integration
- Factory integrates with `sync.WaitGroup` for easy synchronization
- Call `wg.Wait()` to block until all progress bars complete
- Automatic tracking - no manual management needed

## Best Practices

1. **Use TitleProgressable for Clarity** - Always implement the title interface to label your operations
2. **Close finishedCh Properly** - The channel should close when progress is complete, not when value reaches total
3. **Thread Safety** - Ensure your Progressable implementation is safe for concurrent reads
4. **Defer pool.Stop()** - Always clean up the pool with defer to restore terminal state
5. **Let the Pool Handle Display** - Don't call progress bar methods directly; let the pool manage updates
