package main

import (
	"bufio"
	"fmt"
	"os"
	"time"
)

// This utility generates a large log file (~1.1 GB) for performance testing.
func main() {
	fileName := "huge_test_log.txt"
	// 10,000,000 lines will yield roughly 1.1 GB of text
	targetLines := 10_000_000

	file, err := os.Create(fileName)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	// Use a large buffer to speed up file writing
	writer := bufio.NewWriterSize(file, 256*1024)
	defer writer.Flush()

	start := time.Now()
	fmt.Printf("Generating %s (~1.1 GB)... This may take a moment.\n", fileName)

	for i := 1; i <= targetLines; i++ {
		var line string
		// Inject the target word every 10,000 lines for predictable search results
		if i%10000 == 0 {
			line = fmt.Sprintf("Line %d: Лог-запись, в которой нам очень хотелось найти совпадение.\n", i)
		} else {
			line = fmt.Sprintf("Line %d: Standard system event log message with some random performance metrics, status=200, uid=99281\n", i)
		}

		_, err := writer.WriteString(line)
		if err != nil {
			fmt.Println("Error writing to file:", err)
			return
		}
	}

	fmt.Printf("Done! Generated in: %v\n", time.Since(start))
}
