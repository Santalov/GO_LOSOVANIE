package main

import (
	"C"
	"fmt"
)

type TransactionOutput struct {
	aSize uint32
	a []byte
}

func writeBytes(t *TransactionOutput) {
	t.a = make([]byte, t.aSize)
	var i uint32
	for i = 0; i < t.aSize; i++  {
		t.a[i] = byte(i)
	}
}

func main() {
	var hashes [][]byte
	var a = make([]byte, 4)
	a[0] = 0
	a[1] = 0
	a[2] = 0
	a[3] = 5
	var b = make([]byte, 3)
	b[0] = 1
	b[1] = 1
	b[2] = 1
	hashes = append(hashes, a)
	b = a
	hashes = append(hashes, b)
	fmt.Println(hashes)
	fmt.Println(append(hashes[0], hashes[1]...))
	fmt.Println(len(hashes))
}
