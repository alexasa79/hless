package main

import (
	"fmt"
	"os"

	"golang.org/x/exp/mmap"
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

	m, err := mmap.Open(os.Args[1])
	if err != nil {
		panic(err)
	}

	sum := byte(0)
	for i := 0; i < int(fileSize); i++ {
		sum += m.At(i)
	}

	fmt.Println("Sum of all bytes is", sum)
	fmt.Println("Scanned", fileSize, "bytes")
}
