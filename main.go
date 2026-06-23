package main

import (
	"fmt"
	"grep/finder"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: grep <text> [file or directory]")
		os.Exit(1)
	}

	needText := []byte(os.Args[1])
	pathArg := "."
	if len(os.Args) > 2 {
		pathArg = os.Args[2]
	}

	timeStart := time.Now()
	defer func() {
		fmt.Fprintln(os.Stderr, "\n⚡ Total execution time:", time.Since(timeStart))
	}()

	// If no path is provided, default to reading from stdin
	if len(os.Args) < 3 {
		if err := ReadFromStdIn(needText); err != nil {
			fmt.Fprintln(os.Stderr, "Error reading from stdin:", err)
		}
		return
	}

	info, err := os.Stat(pathArg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	if info.IsDir() {
		// Launch high-performance parallel directory traversal
		if err := ReadFromDirParallel(pathArg, needText); err != nil {
			fmt.Fprintln(os.Stderr, "Error reading directory:", err)
		}
	} else {
		if err := ReadFromFile(pathArg, needText); err != nil {
			fmt.Fprintln(os.Stderr, "Error reading file:", err)
		}
	}
}

func ReadFromFile(filename string, needText []byte) error {
	lines, err := finder.ReadFromFileLine(filename, needText, finder.Blue)
	if err != nil {
		return fmt.Errorf("reading file %s: %w", filename, err)
	}
	for _, line := range lines {
		fmt.Println(line)
	}
	return nil
}

func ReadFromStdIn(needText []byte) error {
	lines, err := finder.ReadFromStdIn(needText, finder.Blue)
	if err != nil {
		return err
	}
	for _, line := range lines {
		fmt.Println(line)
	}
	return nil
}

// ReadFromDirParallel utilizes all available CPU cores to search files concurrently
func ReadFromDirParallel(dirname string, needText []byte) error {
	numWorkers := runtime.NumCPU()
	pathsChan := make(chan string, 1000)

	var wg sync.WaitGroup
	var outputMutex sync.Mutex // Prevents console output interleaving from multiple goroutines

	// 1. Initialize the worker pool matching the CPU thread count
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range pathsChan {
				// Each worker processes its assigned file independently to maximize L3 cache usage
				lines, err := finder.ReadFromFileLine(path, needText, finder.Blue)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error processing file %s: %v\n", path, err)
					continue
				}

				// If matches are found, lock stdout briefly to print the chunk sequentially
				if len(lines) > 0 {
					outputMutex.Lock()
					fmt.Println(string(finder.Green) + path + string(finder.Reset))
					for _, line := range lines {
						fmt.Println(line)
					}
					outputMutex.Unlock()
				}
			}
		}()
	}

	// 2. The main thread quickly walks the directory tree and feeds file paths into the channel
	err := filepath.WalkDir(dirname, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			pathsChan <- path
		}
		return nil
	})

	// 3. Close the task channel and wait for all workers to finish remaining processing
	close(pathsChan)
	wg.Wait()

	return err
}
