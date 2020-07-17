package evote

import (
	"encoding/binary"
	"fmt"
	"time"
)

type BlocAndkHash struct {
	b    *Block
	hash [HASH_SIZE]byte
}

type Block struct {
	prevBlockHash [HASH_SIZE]byte
	merkleTree    [HASH_SIZE]byte
	timestamp     uint64
	transSize     uint32
	trans         []TransAndHash
}

func (b *Block) ToBytes() []byte {
	var data = make([]byte, MIN_BLOCK_SIZE)
	copy(data[:HASH_SIZE], b.prevBlockHash[:])
	copy(data[HASH_SIZE:2*HASH_SIZE], b.merkleTree[:])
	binary.LittleEndian.PutUint64(data[2*HASH_SIZE:2*HASH_SIZE+INT_32_SIZE*2], b.timestamp)
	binary.LittleEndian.PutUint32(data[2*HASH_SIZE+INT_32_SIZE*2:MIN_BLOCK_SIZE], b.transSize)
	for _, t := range b.trans {
		data = append(data, t.transaction.ToBytes()...)
	}
	return data
}

func (b *Block) FromBytes(data []byte) int {
	if len(data) < MIN_BLOCK_SIZE {
		return ERR_BLOCK_SIZE
	}
	var offset = HASH_SIZE
	copy(b.prevBlockHash[:], data[:offset])
	copy(b.merkleTree[:], data[offset:offset+HASH_SIZE])
	offset += HASH_SIZE
	b.timestamp = binary.LittleEndian.Uint64(data[offset : offset+INT_32_SIZE*2])
	offset += INT_32_SIZE * 2
	b.transSize = binary.LittleEndian.Uint32(data[offset : offset+INT_32_SIZE])
	return OK
}

func (b *Block) CheckMiningReward(data []byte, creator [PKEY_SIZE]byte) ([]byte, *Transaction, int) {
	var t Transaction
	var transLen = t.FromBytes(data)
	if transLen < 0 {
		return nil, nil, transLen
	}

	if t.inputSize != 0 && t.outputSize != 1 {
		return nil, nil, ERR_TRANS_VERIFY
	}

	var pkey = t.outputs[0].pkeyTo
	fmt.Println("pkey and creator", pkey, creator)
	if pkey != creator {
		return nil, nil, ERR_BLOCK_CREATOR
	}
	if !VerifyData(data[:transLen-SIG_SIZE], t.signature[:], pkey) {
		return nil, nil, ERR_TRANS_VERIFY
	}

	fmt.Println("checkMiningReward: creator ", creator)
	return Hash(data[:transLen]), &t, transLen

}

func (b *Block) CreateMiningReward(keys *CryptoKeysData) TransAndHash {
	var t Transaction
	t.inputSize = 0
	t.outputSize = 1
	var tOut TransactionOutput
	tOut.value = REWARD
	tOut.pkeyTo = keys.pubKeyByte
	t.outputs = append(t.outputs, tOut)
	t.typeValue = ZERO_ARRAY_HASH
	t.typeVote = 0
	t.duration = 0
	t.hashLink = ZERO_ARRAY_HASH
	t.signature = ZERO_ARRAY_SIG
	copy(t.signature[:], keys.Sign(t.ToBytes()))
	var minigReward TransAndHash
	minigReward.transaction = &t
	copy(minigReward.hash[:], Hash(t.ToBytes()))
	return minigReward
}

func (b *Block) BuildMerkleTree() [HASH_SIZE]byte {
	var hashes [][]byte
	for _, t := range b.trans {
		hashes = append(hashes, t.hash[:])
	}
	for {
		var nextHashes [][]byte
		var lenHashes = len(hashes)
		if lenHashes == 1 {
			break
		}
		if lenHashes%2 != 0 {
			hashes = append(hashes, hashes[lenHashes-1])
		}
		lenHashes = len(hashes)
		for i := 0; i < lenHashes; i += 2 {
			nextHashes = append(nextHashes, Hash(append(hashes[i], hashes[i+1]...)))
		}
		hashes = nextHashes
	}
	var hash [HASH_SIZE]byte
	copy(hash[:], hashes[0][:HASH_SIZE])
	return hash
}

func (b *Block) HashBlock(data []byte) []byte {
	return Hash(data)
}

func (b *Block) CreateBlock(t []TransAndHash, prevHash [HASH_SIZE]byte, key *CryptoKeysData) {
	b.trans = append(b.trans, b.CreateMiningReward(key))
	b.trans = append(b.trans, t...)
	b.prevBlockHash = prevHash
	b.merkleTree = b.BuildMerkleTree()
	b.transSize = uint32(len(b.trans))
	b.timestamp = uint64(time.Now().UnixNano())
}

func (b *Block) Verify(data []byte, prevHash [HASH_SIZE]byte,
	creator [PKEY_SIZE]byte) ([]byte, int) {
	if len(data) > MAX_BLOCK_SIZE {
		return nil, ERR_BLOCK_VERIFY
	}
	if b.FromBytes(data) != OK || prevHash != b.prevBlockHash ||
		b.transSize == 0 {
		return nil, ERR_BLOCK_VERIFY
	}
	var blockLen = MIN_BLOCK_SIZE
	var transData = data[MIN_BLOCK_SIZE:]
	var hash, trans, transLen = b.CheckMiningReward(transData, creator)
	if transLen < 0 {
		return nil, transLen
	}
	var transHash TransAndHash
	copy(transHash.hash[:], hash)
	transHash.transaction = trans
	b.trans = append(b.trans, transHash)
	blockLen += transLen
	transData = data[blockLen:]

	var i uint32
	for i = 1; i < b.transSize; i++ {
		var t Transaction
		hash, transLen = t.Verify(transData)
		if transLen < 0 {
			return nil, ERR_BLOCK_VERIFY
		}
		copy(transHash.hash[:], hash)
		transHash.transaction = &t
		b.trans = append(b.trans, transHash)
		blockLen += transLen
		transData = transData[transLen:]
	}

	if b.BuildMerkleTree() != b.merkleTree || len(data) != blockLen {
		return nil, ERR_BLOCK_VERIFY
	}

	return Hash(data), blockLen
}
