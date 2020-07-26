package evote

import (
	"encoding/binary"
	"time"
)

type BlocAndkHash struct {
	B    *Block
	Hash [HASH_SIZE]byte
}

type Block struct {
	PrevBlockHash [HASH_SIZE]byte
	MerkleTree    [HASH_SIZE]byte
	Timestamp     uint64
	TransSize     uint32
	Trans         []TransAndHash
}

func (b *Block) ToBytes() []byte {
	var data = make([]byte, MIN_BLOCK_SIZE)
	copy(data[:HASH_SIZE], b.PrevBlockHash[:])
	copy(data[HASH_SIZE:2*HASH_SIZE], b.MerkleTree[:])
	binary.LittleEndian.PutUint64(data[2*HASH_SIZE:2*HASH_SIZE+INT_32_SIZE*2], b.Timestamp)
	binary.LittleEndian.PutUint32(data[2*HASH_SIZE+INT_32_SIZE*2:MIN_BLOCK_SIZE], b.TransSize)
	for _, t := range b.Trans {
		data = append(data, t.Transaction.ToBytes()...)
	}
	return data
}

func (b *Block) FromBytes(data []byte) int {
	if len(data) < MIN_BLOCK_SIZE {
		return ERR_BLOCK_SIZE
	}
	var offset = HASH_SIZE
	copy(b.PrevBlockHash[:], data[:offset])
	copy(b.MerkleTree[:], data[offset:offset+HASH_SIZE])
	offset += HASH_SIZE
	b.Timestamp = binary.LittleEndian.Uint64(data[offset : offset+INT_32_SIZE*2])
	offset += INT_32_SIZE * 2
	b.TransSize = binary.LittleEndian.Uint32(data[offset : offset+INT_32_SIZE])
	return OK
}

func (b *Block) CheckMiningReward(data []byte, creator [PKEY_SIZE]byte) ([]byte, *Transaction, int) {
	var t Transaction
	var transLen = t.FromBytes(data)
	if transLen < 0 {
		return nil, nil, transLen
	}

	if t.InputSize != 0 && t.OutputSize != 1 {
		return nil, nil, ERR_TRANS_VERIFY
	}

	var pkey = t.Outputs[0].PkeyTo
	if creator[0] != 0x00 && pkey != creator {
		return nil, nil, ERR_BLOCK_CREATOR
	}
	if !VerifyData(data[:transLen-SIG_SIZE], t.Signature[:], pkey) {
		return nil, nil, ERR_TRANS_VERIFY
	}
	return Hash(data[:transLen]), &t, transLen

}

func (b *Block) CreateMiningReward(keys *CryptoKeysData, curChainSize uint64) TransAndHash {
	var t Transaction
	t.InputSize = 0
	t.OutputSize = 1
	var tOut TransactionOutput
	tOut.Value = REWARD
	tOut.PkeyTo = keys.PubkeyByte
	t.Outputs = append(t.Outputs, tOut)
	t.TypeValue = ZERO_ARRAY_HASH
	t.TypeVote = 0
	t.Duration = uint32(curChainSize)
	t.HashLink = ZERO_ARRAY_HASH
	t.Signature = ZERO_ARRAY_SIG
	copy(t.Signature[:], keys.Sign(t.ToBytes()))

	var minigReward TransAndHash
	minigReward.Transaction = &t
	copy(minigReward.Hash[:], Hash(t.ToBytes()))
	return minigReward
}

func (b *Block) BuildMerkleTree() [HASH_SIZE]byte {
	var hashes [][]byte
	for _, t := range b.Trans {
		hashes = append(hashes, t.Hash[:])
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

func (b *Block) CreateBlock(t []TransAndHash, prevHash [HASH_SIZE]byte,
	key *CryptoKeysData, curChainSize uint64) {
	b.Trans = append(b.Trans, b.CreateMiningReward(key, curChainSize))
	b.Trans = append(b.Trans, t...)
	b.PrevBlockHash = prevHash
	b.MerkleTree = b.BuildMerkleTree()
	b.TransSize = uint32(len(b.Trans))
	b.Timestamp = uint64(time.Now().UnixNano())
}

func (b *Block) Verify(data []byte, prevHash [HASH_SIZE]byte,
	creator [PKEY_SIZE]byte, db *Database) ([]byte, int) {
	if len(data) > MAX_BLOCK_SIZE {
		return nil, ERR_BLOCK_VERIFY
	}
	if b.FromBytes(data) != OK || prevHash != b.PrevBlockHash ||
		b.TransSize == 0 {
		return nil, ERR_BLOCK_VERIFY
	}
	var blockLen = MIN_BLOCK_SIZE
	var transData = data[MIN_BLOCK_SIZE:]
	var hash, trans, transLen = b.CheckMiningReward(transData, creator)
	if transLen < 0 {
		return nil, transLen
	}
	var transHash TransAndHash
	copy(transHash.Hash[:], hash)
	transHash.Transaction = trans
	b.Trans = append(b.Trans, transHash)
	blockLen += transLen
	transData = data[blockLen:]

	var i uint32
	for i = 1; i < b.TransSize; i++ {
		var t Transaction
		hash, transLen = t.Verify(transData, db)
		if transLen < 0 {
			return nil, ERR_BLOCK_VERIFY
		}
		copy(transHash.Hash[:], hash)
		transHash.Transaction = &t
		b.Trans = append(b.Trans, transHash)
		blockLen += transLen
		transData = transData[transLen:]
	}

	if b.BuildMerkleTree() != b.MerkleTree || len(data) != blockLen {
		return nil, ERR_BLOCK_VERIFY
	}

	return Hash(data), blockLen
}
