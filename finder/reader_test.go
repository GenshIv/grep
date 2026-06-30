package finder

import (
	"bytes"
	"strings"
	"testing"
)

// Generate test data
func generateTestData(linesCount int, lineLength int, matchRate int) []byte {
	var sb strings.Builder
	sb.Grow(linesCount * (lineLength + 1))
	for i := 0; i < linesCount; i++ {
		if i%matchRate == 0 {
			sb.WriteString(strings.Repeat("a", lineLength-5))
			sb.WriteString("match")
		} else {
			sb.WriteString(strings.Repeat("a", lineLength))
		}
		sb.WriteByte('\n')
	}
	return []byte(sb.String())
}

// Benchmark: Target word appears VERY frequently (on every line)
// This stresses the memory allocation logic for the results slice
func BenchmarkReadDense(b *testing.B) {
	data := generateTestData(10000, 100, 1) // 10k lines, match on every line
	needText := []byte("match")
	b.SetBytes(int64(len(data))) // For calculating MB/s

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(data)
		_, err := ReadFromReaderLine(reader, needText, Blue)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark: Target word appears rarely (1 time per 100 lines)
// This emulates real log searching, where matches are rare
func BenchmarkReadSparse(b *testing.B) {
	data := generateTestData(100000, 100, 100) // 100k lines, 1% match rate
	needText := []byte("match")
	b.SetBytes(int64(len(data)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(data)
		_, err := ReadFromReaderLine(reader, needText, Blue)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark: Target word NEVER appears
// This shows the maximum pure scanning speed of the buffer (Zero-allocation path)
func BenchmarkReadNoMatch(b *testing.B) {
	data := generateTestData(100000, 100, 9999999) // 100k lines, no matches
	needText := []byte("match")
	b.SetBytes(int64(len(data)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(data)
		_, err := ReadFromReaderLine(reader, needText, Blue)
		if err != nil {
			b.Fatal(err)
		}
	}
}
