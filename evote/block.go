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
	proposerPkey  [PKEY_SIZE]byte
	Timestamp     uint64
	TransSize     uint32
	Trans         []TransAndHash
}

func (b *Block) ToBytes() []byte {
	var data = make([]byte, MIN_BLOCK_SIZE)
	copy(data[:HASH_SIZE], b.PrevBlockHash[:])
	copy(data[HASH_SIZE:2*HASH_SIZE], b.MerkleTree[:])
	copy(data[2*HASH_SIZE:2*HASH_SIZE+PKEY_SIZE], b.proposerPkey[:])
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
	copy(b.proposerPkey[:], data[offset:offset+PKEY_SIZE])
	offset += PKEY_SIZE
	b.Timestamp = binary.LittleEndian.Uint64(data[offset : offset+INT_32_SIZE*2])
	offset += INT_32_SIZE * 2
	b.TransSize = binary.LittleEndian.Uint32(data[offset : offset+INT_32_SIZE])
	return OK
}

func (b *Block) BuildMerkleTree() [HASH_SIZE]byte {
	hashes := make([][]byte, 0)
	if len(b.Trans) == 0 {
		hashes = append(hashes, ZERO_ARRAY_HASH[:])
	}
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

func (b *Block) CreateBlock(
	t []TransAndHash,
	prevHash [HASH_SIZE]byte,
	timestamp time.Time,
	proposerPkey [PKEY_SIZE]byte,
) {
	b.Trans = append(b.Trans, t...)
	b.PrevBlockHash = prevHash
	b.MerkleTree = b.BuildMerkleTree()
	b.TransSize = uint32(len(b.Trans))
	b.Timestamp = uint64(timestamp.UnixNano())
	b.proposerPkey = proposerPkey
}
