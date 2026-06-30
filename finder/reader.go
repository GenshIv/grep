package finder

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"strconv"
	"strings"
)

type Colour string

var (
	Reset   = Colour("\033[0m")
	Red     = Colour("\033[31m")
	Green   = Colour("\033[32m")
	Yellow  = Colour("\033[33m")
	Blue    = Colour("\033[34m")
	Magenta = Colour("\033[35m")
	Cyan    = Colour("\033[36m")
	Gray    = Colour("\033[37m")
	White   = Colour("\033[97m")
)

func ReadFromStdIn(needText []byte, c Colour) ([]string, error) {
	return ReadFromReaderLine(os.Stdin, needText, c)
}

func ReadFromFileLine(name string, needText []byte, c Colour) ([]string, error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Heuristic: check the first 512 bytes for null characters to detect binary files
	head := make([]byte, 512)
	n, err := file.Read(head)
	if err != nil && err != io.EOF {
		return nil, err
	}
	if bytes.IndexByte(head[:n], 0) != -1 {
		// Found a null byte (\x00), skipping this binary file
		return nil, nil
	}

	// Reset the read pointer to the beginning of the file
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	return ReadFromReaderLine(file, needText, c)
}

func ReadFromReaderLine(reader io.Reader, needText []byte, c Colour) ([]string, error) {
	out := make([]string, 0, 64)

	// Use Scanner with a custom large buffer to avoid memory allocations
	// per line (Scanner.Bytes() reuses the internal buffer)
	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 1024*1024)    // 1MB initial buffer
	scanner.Buffer(buf, 10*1024*1024) // 10MB maximum line size

	lineNum := 0
	selected := string(c) + string(needText) + string(Reset)
	var sb strings.Builder

	for scanner.Scan() {
		lineNum++
		lineBytes := scanner.Bytes() // Zero-allocation: points to the scanner's internal buffer

		if count := bytes.Count(lineBytes, needText); count > 0 {
			sb.Reset()
			sb.WriteString(strconv.Itoa(lineNum))
			sb.WriteByte(':')
			sb.WriteString(strconv.Itoa(count))
			sb.WriteByte('|')

			// Zero-allocation coloring: avoid bytes.Replace,
			// and write chunks directly to strings.Builder
			idx := 0
			for {
				i := bytes.Index(lineBytes[idx:], needText)
				if i == -1 {
					sb.Write(lineBytes[idx:])
					break
				}
				sb.Write(lineBytes[idx : idx+i])
				sb.WriteString(selected)
				idx += i + len(needText)
			}

			out = append(out, sb.String())
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return out, nil
}
