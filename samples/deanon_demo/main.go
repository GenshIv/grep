package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"runtime"
	"sync/atomic"
	"time"

	"grep/deanon"
)

func main() {
	workers := flag.Int("w", runtime.NumCPU(), "number of worker routines")
	flag.Parse()

	args := flag.Args()
	if len(args) < 2 {
		fmt.Println("Usage: deanon_demo [-w workers] <patterns_file> <target_file>")
		return
	}
	patternsFile := args[0]
	targetFile := args[1]

	timeStart := time.Now()

	// 1. Load patterns
	pf, err := os.Open(patternsFile)
	if err != nil {
		panic(err)
	}
	defer pf.Close()

	var patterns [][]byte
	scanner := bufio.NewScanner(pf)
	for scanner.Scan() {
		text := scanner.Bytes()
		if len(text) >= 2 {
			p := make([]byte, len(text))
			copy(p, text)
			patterns = append(patterns, p)
		}
	}
	fmt.Printf("Loaded %d patterns in %v\n", len(patterns), time.Since(timeStart))

	// 2. Build Matcher
	buildStart := time.Now()
	matcher := deanon.NewMatcher(patterns)
	fmt.Printf("Built 2-byte bucket index in %v\n", time.Since(buildStart))

	// 3. Scan Target File
	tf, err := os.Open(targetFile)
	if err != nil {
		panic(err)
	}
	defer tf.Close()

	scanStart := time.Now()
	var matchesCount int64

	fmt.Printf("Scanning with %d workers...\n", *workers)
	err = matcher.MatchReaderParallel(tf, *workers, func(lineNum, bytePos int, pattern, lineContent []byte) {
		atomic.AddInt64(&matchesCount, 1)
		// fmt.Printf("Match at line %d, byte %d: %s\n  Line: %s\n", lineNum, bytePos, string(pattern), string(lineContent))
	})
	
	if err != nil {
		fmt.Println("Error reading target file:", err)
	}

	fmt.Printf("Found %d matches in %v\n", matchesCount, time.Since(scanStart))
	fmt.Printf("Total execution time: %v\n", time.Since(timeStart))
}
