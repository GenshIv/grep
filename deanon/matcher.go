package deanon

import (
	"bytes"
	"io"
	"sync"
	"unsafe"
)

// Matcher represents a multi-pattern search engine utilizing a 2-byte prefix index.
type Matcher struct {
	hasPattern [65536]bool
	buckets    [65536][][]byte
}

// NewMatcher creates a new Matcher from a slice of byte patterns.
// Patterns shorter than 2 bytes are currently ignored for optimization purposes.
func NewMatcher(patterns [][]byte) *Matcher {
	m := &Matcher{}
	for _, p := range patterns {
		if len(p) >= 2 {
			idx := uint16(p[0]) | (uint16(p[1]) << 8)
			m.hasPattern[idx] = true
			m.buckets[idx] = append(m.buckets[idx], p)
		}
	}
	return m
}

// MatchCallback is the signature for the callback function triggered on matches.
type MatchCallback func(lineNum int, bytePos int, pattern []byte, lineContent []byte)

type scanTask struct {
	data      []byte
	startLine int
	bufPtr    *[]byte
}

// MatchReaderParallel scans the provided io.Reader concurrently using a worker pool.
// It utilizes unsafe.Pointer for memory reads, achieving multi-GB/s throughput.
func (m *Matcher) MatchReaderParallel(reader io.Reader, workers int, onMatch MatchCallback) error {
	if workers < 1 {
		workers = 1
	}

	taskChan := make(chan scanTask, workers*2)
	var wg sync.WaitGroup
	
	// Buffer pool to achieve zero-allocation across chunks
	pool := sync.Pool{
		New: func() interface{} {
			b := make([]byte, 2*1024*1024) // 2MB chunk
			return &b
		},
	}

	// Start workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskChan {
				m.scanChunk(task.data, task.startLine, onMatch)
				pool.Put(task.bufPtr) // Return buffer to pool
			}
		}()
	}

	var leftover []byte
	currentLine := 0

	for {
		bufPtr := pool.Get().(*[]byte)
		buf := *bufPtr

		copy(buf, leftover)
		readOffset := len(leftover)

		n, err := io.ReadFull(reader, buf[readOffset:])
		if n == 0 && err == io.EOF {
			pool.Put(bufPtr)
			break
		}
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			pool.Put(bufPtr)
			return err
		}

		totalLen := readOffset + n
		data := buf[:totalLen]

		// Smart Dispatching: find last \n to never break strings
		lastNewline := bytes.LastIndexByte(data, '\n')
		if lastNewline == -1 {
			lastNewline = len(data) - 1 // Fallback if line is larger than 2MB
		}

		taskData := data[:lastNewline+1]
		
		// Save leftover for the next read
		leftoverLen := len(data) - (lastNewline + 1)
		leftover = make([]byte, leftoverLen)
		copy(leftover, data[lastNewline+1:])

		newlinesInTask := bytes.Count(taskData, []byte{'\n'})

		taskChan <- scanTask{
			data:      taskData,
			startLine: currentLine,
			bufPtr:    bufPtr,
		}

		currentLine += newlinesInTask

		if err == io.EOF || err == io.ErrUnexpectedEOF {
			if len(leftover) > 0 {
				lastBufPtr := pool.Get().(*[]byte)
				lastBuf := *lastBufPtr
				copy(lastBuf, leftover)
				taskChan <- scanTask{
					data:      lastBuf[:len(leftover)],
					startLine: currentLine,
					bufPtr:    lastBufPtr,
				}
			}
			break
		}
	}
	
	close(taskChan)
	wg.Wait()
	return nil
}

// scanChunk uses unsafe.Pointer for maximum L1 Cache throughput
func (m *Matcher) scanChunk(data []byte, startLine int, onMatch MatchCallback) {
	if len(data) < 2 {
		return
	}

	ptr := unsafe.Pointer(&data[0])
	max := uintptr(len(data) - 1)
	
	currentLine := startLine
	lastCountPos := 0

	for i := uintptr(0); i < max; i++ {
		// Read 2 bytes directly from memory, bypassing Go bounds checking
		idx := *(*uint16)(unsafe.Pointer(uintptr(ptr) + i))
		
		if !m.hasPattern[idx] {
			continue
		}

		bucket := m.buckets[idx]
		for _, p := range bucket {
			if len(data)-int(i) >= len(p) {
				if bytes.Equal(data[i:i+uintptr(len(p))], p) {
					// Lazy-calculate line numbers only when match occurs
					currentLine += bytes.Count(data[lastCountPos:i], []byte{'\n'})
					lastCountPos = int(i)

					lineStart := bytes.LastIndexByte(data[:i], '\n') + 1
					lineEnd := bytes.IndexByte(data[i:], '\n')
					if lineEnd == -1 {
						lineEnd = len(data)
					} else {
						lineEnd += int(i)
					}
					
					if onMatch != nil {
						// Pass a copy because data buffer will be recycled
						lineContent := make([]byte, lineEnd-lineStart)
						copy(lineContent, data[lineStart:lineEnd])
						onMatch(currentLine+1, int(i)-lineStart, p, lineContent)
					}

					i += uintptr(len(p) - 1)
					break
				}
			}
		}
	}
}
