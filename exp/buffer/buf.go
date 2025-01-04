package main

import (
	"fmt"
	"os"

	"github.com/alexasa79/hless/internal/lines"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: mmap <FILENAME>")
		os.Exit(-1)
	}

	stat, err := os.Stat(os.Args[1])
	if err != nil {
		panic(err)
	}
	fileSize := stat.Size()

	b := lines.NewBufferFromFile(os.Args[1])

	sum := byte(0)
	for i := 0; i < int(fileSize); i++ {
		sum += b.At(int64(i))
	}

	fmt.Println("Scanned", fileSize, "bytes")
	fmt.Println("The sum of all bytes is", sum)
}
