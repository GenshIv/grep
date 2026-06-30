package deanon

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"
)

// generateTestPatterns creates N random UUID-like patterns.
func generateTestPatterns(count int) [][]byte {
	patterns := make([][]byte, count)
	for i := 0; i < count; i++ {
		patterns[i] = []byte(fmt.Sprintf("SECRET_%d_KEY", rand.Intn(100000000)))
	}
	return patterns
}

// BenchmarkMatcher measures raw throughput of the 2-byte bucket algorithm in memory.
func BenchmarkMatcher(b *testing.B) {
	// Generate 50,000 patterns
	patterns := generateTestPatterns(50000)
	matcher := NewMatcher(patterns)

	// Generate 10MB of random memory payload with sparse matches
	const targetSize = 10 * 1024 * 1024
	payload := make([]byte, targetSize)
	for i := 0; i < targetSize; i++ {
		if i%100 == 0 {
			payload[i] = '\n'
		} else {
			payload[i] = byte(rand.Intn(26) + 97)
		}
	}
	
	// Inject 100 patterns
	for i := 0; i < 100; i++ {
		p := patterns[rand.Intn(len(patterns))]
		pos := rand.Intn(targetSize - len(p) - 1)
		copy(payload[pos:], p)
	}

	b.SetBytes(targetSize)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(payload)
		err := matcher.MatchReaderParallel(reader, 1, func(lineNum, bytePos int, pattern, lineContent []byte) {
			// Do nothing, just simulate callback overhead
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMatcherParallel measures throughput using 16 concurrent workers.
func BenchmarkMatcherParallel(b *testing.B) {
	patterns := generateTestPatterns(50000)
	matcher := NewMatcher(patterns)

	// Generate 100MB of random memory payload to give workers enough chunks
	const targetSize = 100 * 1024 * 1024
	payload := make([]byte, targetSize)
	for i := 0; i < targetSize; i++ {
		if i%100 == 0 {
			payload[i] = '\n'
		} else {
			payload[i] = byte(rand.Intn(26) + 97)
		}
	}
	
	// Inject 1000 patterns
	for i := 0; i < 1000; i++ {
		p := patterns[rand.Intn(len(patterns))]
		pos := rand.Intn(targetSize - len(p) - 1)
		copy(payload[pos:], p)
	}

	b.SetBytes(targetSize)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(payload)
		err := matcher.MatchReaderParallel(reader, 16, func(lineNum, bytePos int, pattern, lineContent []byte) {})
		if err != nil {
			b.Fatal(err)
		}
	}
}
