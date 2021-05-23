package evote

import (
	"encoding/binary"
)

type TransAndHash struct {
	Hash        [HashSize]byte // 32 bytes
	Transaction *Transaction
}

type TransactionInput struct {
	PrevId   [HashSize]byte
	OutIndex uint32
}

type TransactionOutput struct {
	Value  uint32
	PkeyTo [PkeySize]byte
}

type Transaction struct {
	InputSize  uint32
	Inputs     []TransactionInput
	OutputSize uint32
	Outputs    []TransactionOutput
	TypeValue  [HashSize]byte // необязательное поле
	TypeVote   uint32         // необязательное поле
	Duration   uint32         // необязательное поле, в первой транзе в блоке (которая создает деньги) здесь номер блока
	HashLink   [HashSize]byte // необязательное поле
	Signature  [SigSize]byte
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
	TxId      [HashSize]byte // хеш транзы, из которой взят выход
	TypeValue [HashSize]byte
	Index     uint32 // номер выхода в массиве выходов
	Value     uint32
	PkeyTo    [PkeySize]byte
	Timestamp uint64
}

func (utxo *UTXO) FromBytes(data []byte) int {
	if len(data) != UtxoSize {
		return ErrUtxoSize
	}
	var offset uint32 = HashSize
	copy(utxo.TxId[:], data[:offset])
	copy(utxo.TypeValue[:], data[offset:offset+HashSize])
	offset += HashSize
	utxo.Index = binary.LittleEndian.Uint32(data[offset : offset+Int32Size])
	offset += Int32Size
	utxo.Value = binary.LittleEndian.Uint32(data[offset : offset+Int32Size])
	offset += Int32Size
	copy(utxo.PkeyTo[:], data[offset:offset+PkeySize])
	offset += PkeySize
	utxo.Timestamp = binary.LittleEndian.Uint64(data[offset : offset+2*Int32Size])
	return OK
}

func (utxo *UTXO) ToBytes() []byte {
	data := make([]byte, UtxoSize)
	var offset uint32 = HashSize
	copy(data[:offset], utxo.TxId[:])
	copy(data[offset:offset+HashSize], utxo.TypeValue[:])
	offset += HashSize
	binary.LittleEndian.PutUint32(data[offset:offset+Int32Size], utxo.Index)
	offset += Int32Size
	binary.LittleEndian.PutUint32(data[offset:offset+Int32Size], utxo.Value)
	offset += Int32Size
	copy(data[offset:offset+PkeySize], utxo.PkeyTo[:])
	offset += PkeySize
	binary.LittleEndian.PutUint64(data[offset:offset+2*Int32Size], utxo.Timestamp)
	return data
}

func (t *TransactionInput) ToBytes() []byte {
	data := make([]byte, TransInputSize)
	copy(data[:HashSize], t.PrevId[:])
	binary.LittleEndian.PutUint32(data[HashSize:], t.OutIndex)
	return data
}

func (t *TransactionInput) FromBytes(data []byte) {
	copy(t.PrevId[:], data[:HashSize])
	t.OutIndex = binary.LittleEndian.Uint32(data[HashSize:])
}

func (t *TransactionOutput) ToBytes() []byte {
	data := make([]byte, TransOutputSize)
	binary.LittleEndian.PutUint32(data[:Int32Size], t.Value)
	copy(data[Int32Size:], t.PkeyTo[:])
	return data
}

func (t *TransactionOutput) FromBytes(data []byte) {
	t.Value = binary.LittleEndian.Uint32(data[:Int32Size])
	copy(t.PkeyTo[:], data[Int32Size:])
}

func (t *Transaction) ToBytes() []byte {
	var size uint32 = MinTransSize - TransOutputSize
	size += TransOutputSize * t.OutputSize
	size += TransInputSize * t.InputSize
	data := make([]byte, size)
	binary.LittleEndian.PutUint32(data[:Int32Size], t.InputSize)
	var i uint32 = 0
	var offset uint32 = Int32Size
	for i = 0; i < t.InputSize; i++ {
		copy(data[offset:offset+TransInputSize], t.Inputs[i].ToBytes())
		offset += TransInputSize
	}
	binary.LittleEndian.PutUint32(data[offset:offset+Int32Size], t.OutputSize)
	offset += Int32Size
	for i = 0; i < t.OutputSize; i++ {
		copy(data[offset:offset+TransOutputSize], t.Outputs[i].ToBytes())
		offset += TransOutputSize
	}
	copy(data[offset:offset+HashSize], t.TypeValue[:])
	offset += HashSize
	binary.LittleEndian.PutUint32(data[offset:offset+Int32Size], t.TypeVote)
	offset += Int32Size
	binary.LittleEndian.PutUint32(data[offset:offset+Int32Size], t.Duration)
	offset += Int32Size
	copy(data[offset:offset+HashSize], t.HashLink[:])
	offset += HashSize
	copy(data[offset:offset+SigSize], t.Signature[:])
	return data
}

func (t *Transaction) FromBytes(data []byte) int {
	var size = MinTransSize
	if len(data) < size {
		return ErrTransSize
	}
	size -= TransOutputSize
	t.InputSize = binary.LittleEndian.Uint32(data[:Int32Size])
	size += int(t.InputSize * TransInputSize)
	var offset uint32 = Int32Size
	var i uint32
	t.Inputs = make([]TransactionInput, t.InputSize)
	if len(data) < size {
		return ErrTransSize
	}
	for i = 0; i < t.InputSize; i++ {
		t.Inputs[i].FromBytes(data[offset : offset+TransInputSize])
		offset += TransInputSize
	}

	t.OutputSize = binary.LittleEndian.Uint32(data[offset : offset+Int32Size])
	size += int(t.OutputSize * TransOutputSize)
	offset += Int32Size
	t.Outputs = make([]TransactionOutput, t.OutputSize)
	if len(data) < size {
		return ErrTransSize
	}
	for i = 0; i < t.OutputSize; i++ {
		t.Outputs[i].FromBytes(data[offset : offset+TransOutputSize])
		offset += TransOutputSize
	}

	copy(t.TypeValue[:], data[offset:offset+HashSize])
	offset += HashSize
	t.TypeVote = binary.LittleEndian.Uint32(data[offset : offset+Int32Size])
	offset += Int32Size
	t.Duration = binary.LittleEndian.Uint32(data[offset : offset+Int32Size])
	offset += Int32Size
	copy(t.HashLink[:], data[offset:offset+HashSize])
	offset += HashSize
	copy(t.Signature[:], data[offset:offset+SigSize])
	return size
}

func (t *Transaction) CreateTrans(inputs []*UTXO, outputs map[[PkeySize]byte]uint32,
	typeValue [HashSize]byte, keys *CryptoKeysData, typeVote uint32, duration uint32, ignoreTypeValue bool) int {
	if len(inputs) == 0 || len(outputs) == 0 {
		return ErrCreateTrans
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
		if (ignoreTypeValue || in.TypeValue == typeValue) && maxValInputs < maxValOutputs {
			t.Inputs = append(t.Inputs,
				TransactionInput{
					PrevId:   in.TxId,
					OutIndex: in.Index,
				})
			maxValInputs += in.Value
		}
	}
	transLen := MinTransSize - TransOutputSize + t.InputSize*TransInputSize
	transLen += TransOutputSize * t.OutputSize
	if maxValInputs < maxValOutputs || transLen > MaxBlockSize-MinBlockSize {
		t.Inputs = make([]TransactionInput, 0)
		t.Outputs = make([]TransactionOutput, 0)
		return ErrCreateTrans
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
	t.TypeVote = typeVote
	t.Duration = duration
	t.HashLink = ZeroArrayHash
	t.Signature = ZeroArraySig
	copy(t.Signature[:], keys.Sign(t.ToBytes()))

	return OK
}

func (t *Transaction) CreateMiningReward(keys *CryptoKeysData, rewardForBlock [HashSize]byte) {
	// reward for block is created after that block
	t.InputSize = 0
	t.OutputSize = 1
	var tOut TransactionOutput
	tOut.Value = RewardCoins
	tOut.PkeyTo = keys.PkeyByte
	t.Outputs = append(t.Outputs, tOut)
	t.TypeValue = ZeroArrayHash
	t.TypeVote = 0
	t.Duration = 0
	t.HashLink = rewardForBlock
	t.Signature = ZeroArraySig
	copy(t.Signature[:], keys.Sign(t.ToBytes()))
}
