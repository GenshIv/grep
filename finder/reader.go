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

	return ReadFromReaderLine(file, needText, c)
}

func ReadFromReaderLine(reader io.Reader, needText []byte, c Colour) ([]string, error) {
	out := make([]string, 0, 64)

	// Используем Reader вместо Scanner — у него нет лимита в 64KB
	bufReader := bufio.NewReaderSize(reader, 64*1024)
	lineNum := 0
	selected := string(c) + string(needText) + string(Reset)
	var sb strings.Builder

	for {
		lineBytes, err := bufReader.ReadBytes('\n')
		if len(lineBytes) > 0 {
			lineNum++

			// Отрезаем символы конца строки для чистого вывода
			cleanedBytes := bytes.TrimSuffix(lineBytes, []byte("\n"))
			cleanedBytes = bytes.TrimSuffix(cleanedBytes, []byte("\r"))

			if count := bytes.Count(cleanedBytes, needText); count > 0 {
				sb.Reset()
				sb.WriteString(strconv.Itoa(lineNum))
				sb.WriteByte(':')
				sb.WriteString(strconv.Itoa(count))
				sb.WriteByte('|')

				resSelected := bytes.Replace(cleanedBytes, needText, []byte(selected), -1)
				sb.Write(resSelected)

				out = append(out, sb.String())
			}
		}

		if err != nil {
			if err == io.EOF {
				break // Файл успешно прочитан до конца
			}
			return nil, err // Возвращаем реальную системную ошибку чтения
		}
	}

	return out, nil
}
