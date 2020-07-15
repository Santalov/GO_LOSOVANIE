package main

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

}
