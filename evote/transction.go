package evote

import (
	"encoding/binary"
	"fmt"
)

//func for Search in Database
func SearchTrans(prevId [HASH_SIZE]byte) *Transaction {
	return nil
}

type TransAndHash struct {
	Hash        [HASH_SIZE]byte // 32 bytes
	Transaction *Transaction
}

type TransactionInput struct {
	PrevId   [HASH_SIZE]byte
	OutIndex uint32
}

type TransactionOutput struct {
	Value  uint32
	PkeyTo [PKEY_SIZE]byte
}

type Transaction struct {
	InputSize  uint32
	Inputs     []TransactionInput
	OutputSize uint32
	Outputs    []TransactionOutput
	TypeValue  [HASH_SIZE]byte // необязательное поле
	TypeVote   uint32          // необязательное поле
	Duration   uint32          // необязательное поле, в первой транзе в блоке (которая создает деньги) здесь номер блока
	HashLink   [HASH_SIZE]byte // необязательное поле
	Signature  [SIG_SIZE]byte
	// TypeValue равен хешу предыдущей транзы, если предыдущая транза была первой
	// транзой создания голосования, то есть у неё был typeVote != 0
	// TypeValue равен TypeValue предыдущей транзы, если предыдущая транза обычная (то есть её typeVote == 0)
	// При добавлении в голосование новых участников используется hashLink
	// hashLink = хешу предыдущей транзы добавления участников в голосование
	// (или хешу транзы создания голосования)
	// При добавлении участников голосования с помощью транзакции с ненулевым hashLink,
	// typeValue таких транз устанавливается равным самой первой транзакции создания голосования
	// typeVote таких транз равен нулю
}

type UTXO struct {
	TxId      [HASH_SIZE]byte // хеш транзы, из которой взят выход
	TypeValue [HASH_SIZE]byte
	Index     uint32 // номер выхода в массиве выходов
	Value     uint32
	PkeyTo    [PKEY_SIZE]byte
}

func (utxo *UTXO) FromBytes(data []byte) int {
	if len(data) != UTXO_SIZE {
		return ERR_UTXO_SIZE
	}
	var offset uint32 = HASH_SIZE
	copy(utxo.TxId[:], data[:offset])
	copy(utxo.TypeValue[:], data[offset:offset+HASH_SIZE])
	offset += HASH_SIZE
	utxo.Index = binary.LittleEndian.Uint32(data[offset : offset+INT_32_SIZE])
	offset += INT_32_SIZE
	utxo.Value = binary.LittleEndian.Uint32(data[offset : offset+INT_32_SIZE])
	offset += INT_32_SIZE
	copy(utxo.PkeyTo[:], data[offset:])
	return OK
}

func (utxo *UTXO) ToBytes() []byte {
	data := make([]byte, UTXO_SIZE)
	var offset uint32 = HASH_SIZE
	copy(data[:offset], utxo.TxId[:])
	copy(data[offset:offset+HASH_SIZE], utxo.TypeValue[:])
	offset += HASH_SIZE
	binary.LittleEndian.PutUint32(data[offset:offset+INT_32_SIZE], utxo.Index)
	offset += INT_32_SIZE
	binary.LittleEndian.PutUint32(data[offset:offset+INT_32_SIZE], utxo.Value)
	offset += INT_32_SIZE
	copy(data[offset:offset+PKEY_SIZE], utxo.PkeyTo[:])
	return data
}

func (t *TransactionInput) ToBytes() []byte {
	data := make([]byte, TRANS_INPUT_SIZE)
	copy(data[:HASH_SIZE], t.PrevId[:])
	binary.LittleEndian.PutUint32(data[HASH_SIZE:], t.OutIndex)
	return data
}

func (t *TransactionInput) FromBytes(data []byte) {
	copy(t.PrevId[:], data[:HASH_SIZE])
	t.OutIndex = binary.LittleEndian.Uint32(data[HASH_SIZE:])
}

func (t *TransactionOutput) ToBytes() []byte {
	data := make([]byte, TRANS_OUTPUT_SIZE)
	binary.LittleEndian.PutUint32(data[:INT_32_SIZE], t.Value)
	copy(data[INT_32_SIZE:], t.PkeyTo[:])
	return data
}

func (t *TransactionOutput) FromBytes(data []byte) {
	t.Value = binary.LittleEndian.Uint32(data[:INT_32_SIZE])
	copy(t.PkeyTo[:], data[INT_32_SIZE:])
}

func (t *Transaction) ToBytes() []byte {
	var size uint32 = MIN_TRANS_SIZE - TRANS_OUTPUT_SIZE
	size += TRANS_OUTPUT_SIZE * t.OutputSize
	size += TRANS_INPUT_SIZE * t.InputSize
	data := make([]byte, size)
	binary.LittleEndian.PutUint32(data[:INT_32_SIZE], t.InputSize)
	var i uint32 = 0
	var offset uint32 = INT_32_SIZE
	for i = 0; i < t.InputSize; i++ {
		copy(data[offset:offset+TRANS_INPUT_SIZE], t.Inputs[i].ToBytes())
		offset += TRANS_INPUT_SIZE
	}
	binary.LittleEndian.PutUint32(data[offset:offset+INT_32_SIZE], t.OutputSize)
	offset += INT_32_SIZE
	for i = 0; i < t.OutputSize; i++ {
		copy(data[offset:offset+TRANS_OUTPUT_SIZE], t.Outputs[i].ToBytes())
		offset += TRANS_OUTPUT_SIZE
	}
	copy(data[offset:offset+HASH_SIZE], t.TypeValue[:])
	offset += HASH_SIZE
	binary.LittleEndian.PutUint32(data[offset:offset+INT_32_SIZE], t.TypeVote)
	offset += INT_32_SIZE
	binary.LittleEndian.PutUint32(data[offset:offset+INT_32_SIZE], t.Duration)
	offset += INT_32_SIZE
	copy(data[offset:offset+HASH_SIZE], t.HashLink[:])
	offset += HASH_SIZE
	copy(data[offset:offset+SIG_SIZE], t.Signature[:])
	return data
}

func (t *Transaction) FromBytes(data []byte) int {
	var size = MIN_TRANS_SIZE
	if len(data) < size {
		return ERR_TRANS_SIZE
	}
	size -= TRANS_OUTPUT_SIZE
	t.InputSize = binary.LittleEndian.Uint32(data[:INT_32_SIZE])
	size += int(t.InputSize * TRANS_INPUT_SIZE)
	var offset uint32 = INT_32_SIZE
	var i uint32
	t.Inputs = make([]TransactionInput, t.InputSize)
	if len(data) < size {
		return ERR_TRANS_SIZE
	}
	for i = 0; i < t.InputSize; i++ {
		t.Inputs[i].FromBytes(data[offset : offset+TRANS_INPUT_SIZE])
		offset += TRANS_INPUT_SIZE
	}

	t.OutputSize = binary.LittleEndian.Uint32(data[offset : offset+INT_32_SIZE])
	size += int(t.OutputSize * TRANS_OUTPUT_SIZE)
	offset += INT_32_SIZE
	t.Outputs = make([]TransactionOutput, t.OutputSize)
	if len(data) < size {
		return ERR_TRANS_SIZE
	}
	for i = 0; i < t.OutputSize; i++ {
		t.Outputs[i].FromBytes(data[offset : offset+TRANS_OUTPUT_SIZE])
		offset += TRANS_OUTPUT_SIZE
	}

	copy(t.TypeValue[:], data[offset:offset+HASH_SIZE])
	offset += HASH_SIZE
	t.TypeVote = binary.LittleEndian.Uint32(data[offset : offset+INT_32_SIZE])
	offset += INT_32_SIZE
	t.Duration = binary.LittleEndian.Uint32(data[offset : offset+INT_32_SIZE])
	offset += INT_32_SIZE
	copy(t.HashLink[:], data[offset:offset+HASH_SIZE])
	offset += HASH_SIZE
	copy(t.Signature[:], data[offset:offset+SIG_SIZE])
	return size
}

func (t *Transaction) CreateTrans(inputs []*UTXO, outputs map[[PKEY_SIZE]byte]uint32,
	typeValue [HASH_SIZE]byte, keys *CryptoKeysData) int {
	if len(inputs) == 0 || len(outputs) == 0 {
		return ERR_CREATE_TRANS
	}

	var maxValInputs uint32 = 0
	var maxValOutputs uint32 = 0
	for pkey, val := range outputs {
		t.Outputs = append(t.Outputs,
			TransactionOutput{
				PkeyTo: pkey,
				Value:  val,
			})
		maxValOutputs += val
	}

	for _, in := range inputs {
		if in.TypeValue == typeValue && maxValInputs < maxValOutputs {
			t.Inputs = append(t.Inputs,
				TransactionInput{
					PrevId:   in.TxId,
					OutIndex: in.Index,
				})
			maxValInputs += in.Value
		}
	}
	transLen := MIN_TRANS_SIZE -TRANS_OUTPUT_SIZE + t.InputSize*TRANS_INPUT_SIZE
	transLen += TRANS_OUTPUT_SIZE*t.OutputSize
	if maxValInputs < maxValOutputs || transLen > MAX_BLOCK_SIZE-MIN_BLOCK_SIZE {
		t.Inputs = make([]TransactionInput, 0)
		t.Outputs = make([]TransactionOutput, 0)
		return ERR_CREATE_TRANS
	}

	if maxValInputs > maxValOutputs {
		t.Outputs = append(t.Outputs,
			TransactionOutput{
				PkeyTo: inputs[0].PkeyTo,
				Value:  maxValInputs - maxValOutputs,
			})
	}
	t.OutputSize = uint32(len(t.Outputs))
	t.InputSize = uint32(len(t.Inputs))
	t.TypeValue = typeValue
	t.TypeVote = 0 // заглушка, нужен фикс
	t.Duration = 0 // заглушка, нужен фикс
	t.HashLink = ZERO_ARRAY_HASH
	t.Signature = ZERO_ARRAY_SIG
	copy(t.Signature[:], keys.Sign(t.ToBytes()))

	return OK
}

func (t *Transaction) Verify(data []byte, db *Database) ([]byte, int) {
	var transSize = t.FromBytes(data)
	if transSize == ERR_TRANS_SIZE {
		fmt.Println("err: tx not parsed")
		return nil, ERR_TRANS_SIZE
	}
	if t.OutputSize == 0 || t.InputSize == 0 {
		fmt.Println("err: no inputs or outputs")
		return nil, ERR_TRANS_VERIFY
	}
	if t.TypeVote == 0 {
		// обработать как обычную транзу
		var inputsSum, outputsSum uint32
		var pkey [PKEY_SIZE]byte
		for i, input := range t.Inputs {
			var correspondingUtxo *UTXO
			utxos, err := db.GetUTXOSByTxId(input.PrevId)
			if err != nil {
				panic(err)
			}
			// проверка, что вход - непотраченный выход дургой транзы
			for _, utxo := range utxos {
				if utxo.Index == input.OutIndex {
					correspondingUtxo = utxo
					if i == 0 {
						pkey = correspondingUtxo.PkeyTo
					} else {
						if pkey != correspondingUtxo.PkeyTo {
							fmt.Println("err: input not owned by sender")
						}
					}
					break
				}
			}
			if correspondingUtxo == nil {
				fmt.Println("err: double spending in tx")
				return nil, ERR_TRANS_VERIFY
			}
			inputsSum += correspondingUtxo.Value
			// проверка, что в одной транзе не смешиваются разные typeValue
			if correspondingUtxo.TypeValue != t.TypeValue {
				fmt.Println("err: incorrect typeValue in input", input)
				return nil, ERR_TRANS_VERIFY
			}

		}
		for _, output := range t.Outputs {
			outputsSum += output.Value
		}
		if outputsSum != inputsSum {
			fmt.Printf("err: outputs sum %v is not matching than inputs sum %v\n", outputsSum, inputsSum)
			return nil, ERR_TRANS_VERIFY
		}
		if !VerifyData(data[:transSize-SIG_SIZE], t.Signature[:], pkey) {
			fmt.Println("err: signature doesnt match")
			return nil, ERR_TRANS_VERIFY
		}
		return Hash(data[:transSize]), transSize
	} else {
		// оработать как транзу создания голосования
		panic("not implemented")
	}
}
