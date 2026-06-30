package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strconv"
)

func main() {
	fmt.Println("Generating patterns.txt (50,000 patterns)...")
	os.MkdirAll(".", 0755)
	patternsFile, err := os.Create("patterns.txt")
	if err != nil {
		panic(err)
	}
	defer patternsFile.Close()

	patterns := make([][]byte, 50000)
	for i := 0; i < 50000; i++ {
		patterns[i] = []byte("SECRET_" + strconv.Itoa(rand.Intn(100000000)) + "_KEY")
		patternsFile.WriteString(string(patterns[i]) + "\n")
	}

	fmt.Println("Generating target.log (~1GB)...")
	targetFile, err := os.Create("target.log")
	if err != nil {
		panic(err)
	}
	defer targetFile.Close()

	writer := bufio.NewWriterSize(targetFile, 1024*1024)
	defer writer.Flush()

	targetSize := int64(1024 * 1024 * 1024) // 1 GB
	var currentSize int64 = 0

	for currentSize < targetSize {
		lineLength := 100 + rand.Intn(200)
		line := make([]byte, lineLength)
		for j := 0; j < lineLength-1; j++ {
			line[j] = byte(rand.Intn(26) + 97)
		}

		if rand.Intn(1000) == 0 {
			p := patterns[rand.Intn(len(patterns))]
			if len(p) < lineLength-1 {
				insertPos := rand.Intn(lineLength - 1 - len(p))
				copy(line[insertPos:], p)
			}
		}

		line[lineLength-1] = '\n'
		n, _ := writer.Write(line)
		currentSize += int64(n)
	}
	fmt.Println("Done generating 1GB file!")
}
