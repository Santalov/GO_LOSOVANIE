package evote

import (
	"encoding/binary"
	"time"
)

type BlocAndkHash struct {
	B    *Block
	Hash [HashSize]byte
}

type Block struct {
	PrevBlockHash [HashSize]byte
	MerkleTree    [HashSize]byte
	proposerPkey  [PkeySize]byte
	Timestamp     uint64
	TransSize     uint32
	Trans         []TransAndHash
}

func (b *Block) ToBytes() []byte {
	var data = make([]byte, MinBlockSize)
	copy(data[:HashSize], b.PrevBlockHash[:])
	copy(data[HashSize:2*HashSize], b.MerkleTree[:])
	copy(data[2*HashSize:2*HashSize+PkeySize], b.proposerPkey[:])
	binary.LittleEndian.PutUint64(data[2*HashSize:2*HashSize+Int32Size*2], b.Timestamp)
	binary.LittleEndian.PutUint32(data[2*HashSize+Int32Size*2:MinBlockSize], b.TransSize)
	for _, t := range b.Trans {
		data = append(data, t.Transaction.ToBytes()...)
	}
	return data
}

func (b *Block) FromBytes(data []byte) int {
	if len(data) < MinBlockSize {
		return ErrBlockSize
	}
	var offset = HashSize
	copy(b.PrevBlockHash[:], data[:offset])
	copy(b.MerkleTree[:], data[offset:offset+HashSize])
	offset += HashSize
	copy(b.proposerPkey[:], data[offset:offset+PkeySize])
	offset += PkeySize
	b.Timestamp = binary.LittleEndian.Uint64(data[offset : offset+Int32Size*2])
	offset += Int32Size * 2
	b.TransSize = binary.LittleEndian.Uint32(data[offset : offset+Int32Size])
	return OK
}

func (b *Block) BuildMerkleTree() [HashSize]byte {
	hashes := make([][]byte, 0)
	if len(b.Trans) == 0 {
		hashes = append(hashes, ZeroArrayHash[:])
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
	var hash [HashSize]byte
	copy(hash[:], hashes[0][:HashSize])
	return hash
}

func (b *Block) HashBlock(data []byte) []byte {
	return Hash(data)
}

func (b *Block) CreateBlock(
	t []TransAndHash,
	prevHash [HashSize]byte,
	timestamp time.Time,
	proposerPkey [PkeySize]byte,
) {
	b.Trans = append(b.Trans, t...)
	b.PrevBlockHash = prevHash
	b.MerkleTree = b.BuildMerkleTree()
	b.TransSize = uint32(len(b.Trans))
	b.Timestamp = uint64(timestamp.UnixNano())
	b.proposerPkey = proposerPkey
}
