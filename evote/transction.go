package evote

import (
	"encoding/binary"
)

//func for Search in Database
func SearchTrans(prevId [HASH_SIZE]byte) *Transaction {
	return nil
}

type TransactionInput struct {
	prevId [HASH_SIZE]byte
	outIndex uint32
}

type TransactionOutput struct {
	value uint32
	pkeyTo [PKEY_SIZE]byte
}

type Transaction struct {
	inputSize uint32
	inputs []TransactionInput
	outputSize uint32
	outputs []TransactionOutput
	typeValue [HASH_SIZE]byte
	typeVote uint32
	duration uint32
	hashLink [HASH_SIZE]byte
	signature [SIG_SIZE]byte
}

func (t *TransactionInput) ToBytes() []byte {
	data := make([]byte, TRANS_INPUT_SIZE)
	copy(data[:HASH_SIZE], t.prevId[:])
	binary.LittleEndian.PutUint32(data[HASH_SIZE:], t.outIndex)
	return data
}

func (t *TransactionInput) FromBytes(data []byte) {
	copy(t.prevId[:], data[:HASH_SIZE])
	t.outIndex = binary.LittleEndian.Uint32(data[HASH_SIZE:])
}

func (t *TransactionOutput) ToBytes() []byte {
	data := make([]byte, TRANS_OUTPUT_SIZE)
	binary.LittleEndian.PutUint32(data[:INT_32_SIZE], t.value)
	copy(data[INT_32_SIZE:], t.pkeyTo[:])
	return data
}

func (t *TransactionOutput) FromBytes(data []byte) {
	t.value = binary.LittleEndian.Uint32(data[:INT_32_SIZE])
	copy(t.pkeyTo[:], data[INT_32_SIZE:])
}

func (t *Transaction) ToBytes() []byte {
	var size uint32 = MIN_TRANS_SIZE - TRANS_OUTPUT_SIZE
	size += TRANS_OUTPUT_SIZE * t.outputSize
	size += TRANS_INPUT_SIZE * t.inputSize
	data := make([]byte, size)
	binary.LittleEndian.PutUint32(data[:INT_32_SIZE], t.inputSize)
	var i uint32 = 0
	var offset uint32 = INT_32_SIZE
	for i = 0; i < t.inputSize; i++ {
		copy(data[offset:offset+TRANS_INPUT_SIZE],t.inputs[i].ToBytes())
		offset += TRANS_INPUT_SIZE
	}
	binary.LittleEndian.PutUint32(data[offset:offset+INT_32_SIZE], t.outputSize)
	offset += INT_32_SIZE
	for i = 0; i < t.inputSize; i++ {
		copy(data[offset:offset+TRANS_OUTPUT_SIZE], t.outputs[i].ToBytes())
		offset += TRANS_OUTPUT_SIZE
	}
	copy(data[offset:offset+HASH_SIZE], t.typeValue[:])
	offset += HASH_SIZE
	binary.LittleEndian.PutUint32(data[offset:offset+INT_32_SIZE], t.typeVote)
	offset += INT_32_SIZE
	binary.LittleEndian.PutUint32(data[offset:offset+INT_32_SIZE], t.duration)
	offset += INT_32_SIZE
	copy(data[offset:offset+HASH_SIZE], t.hashLink[:])
	offset += HASH_SIZE
	copy(data[offset:offset+SIG_SIZE], t.signature[:])
	return data
}

func (t *Transaction) FromBytes(data []byte) int {
	var size = MIN_TRANS_SIZE
	if len(data) < size {
		return ERR_TRANS_SIZE
	}
	size -= TRANS_OUTPUT_SIZE
	t.inputSize = binary.LittleEndian.Uint32(data[:INT_32_SIZE])
	size += int(t.inputSize * TRANS_INPUT_SIZE)
	var offset uint32 = INT_32_SIZE
	var i uint32
	t.inputs = make([]TransactionInput, t.inputSize)
	if len(data) < size {
		return ERR_TRANS_SIZE
	}
	for i = 0; i < t.inputSize; i++ {
		t.inputs[i].FromBytes(data[offset:offset+TRANS_INPUT_SIZE])
		offset += TRANS_INPUT_SIZE
	}

	t.outputSize = binary.LittleEndian.Uint32(data[offset:offset+INT_32_SIZE])
	size += int(t.outputSize * TRANS_OUTPUT_SIZE)
	offset += INT_32_SIZE
	t.outputs = make([]TransactionOutput, t.outputSize)
	if len(data) < size {
		return ERR_TRANS_SIZE
	}
	for i = 0; i < t.outputSize; i++ {
		t.outputs[i].FromBytes(data[offset:offset+TRANS_OUTPUT_SIZE])
		offset += TRANS_OUTPUT_SIZE
	}

	copy(t.typeValue[:], data[offset:offset+HASH_SIZE])
	offset += HASH_SIZE
	t.typeVote = binary.LittleEndian.Uint32(data[offset:offset+INT_32_SIZE])
	offset += INT_32_SIZE
	t.duration = binary.LittleEndian.Uint32(data[offset:offset+INT_32_SIZE])
	offset += INT_32_SIZE
	copy(t.hashLink[:], data[offset:offset+HASH_SIZE])
	offset += HASH_SIZE
	copy(t.signature[:], data[offset:offset+SIG_SIZE])
	return size
}

func (t *Transaction) Verify(data []byte) ([]byte, int) {
	var transSize = t.FromBytes(data)
	if transSize == ERR_TRANS_SIZE {
		return nil, ERR_TRANS_SIZE
	}
	if t.outputSize == 0 || t.inputSize == 0 {
		return nil, ERR_TRANS_VERIFY
	}
	var special = t.outputs[0].pkeyTo == SPECIAL_PKEY
	var inputTrans = t.inputs[0]
	var oldTrans = SearchTrans(inputTrans.prevId)
	if oldTrans == nil {
		return nil, ERR_TRANS_VERIFY
	}
	var outIndex = inputTrans.outIndex
	var pkey = oldTrans.outputs[outIndex].pkeyTo
	var oldValSum uint32 = 0
	var thisValSum uint32 = 0
	for _, inputTrans := range t.inputs {
		oldTrans = SearchTrans(inputTrans.prevId)
		outIndex = inputTrans.outIndex
		if oldTrans == nil || oldTrans.outputs[outIndex].pkeyTo != pkey ||
					t.typeVote != oldTrans.typeVote ||
					t.duration != oldTrans.duration {
			return nil, ERR_TRANS_VERIFY
		}
		if !special && t.typeValue != oldTrans.typeValue {
			return nil, ERR_TRANS_VERIFY
		}
		oldValSum += oldTrans.outputs[outIndex].value
	}

	for _, outputTrans := range t.outputs {
		thisValSum += outputTrans.value
	}

	if oldValSum != thisValSum {
		return nil, ERR_TRANS_VERIFY
	}

	if !VerifyTransaction(data[:transSize - SIG_SIZE], t.signature[:], pkey) {
		return nil, ERR_TRANS_VERIFY
	}
	return Hash(data[:transSize]), transSize
}

