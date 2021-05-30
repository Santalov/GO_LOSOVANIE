package evote

import (
	"GO_LOSOVANIE/evote/golosovaniepb"
	"github.com/golang/protobuf/proto"
	"time"
)

func BuildMerkleTree(txHashes [][HashSize]byte) [HashSize]byte {
	hashes := make([][]byte, len(txHashes))
	if len(txHashes) == 0 {
		hashes = append(hashes, ZeroArrayHash[:])
	}
	for i, h := range txHashes {
		hashes[i] = h[:]
	}
	return buildMerkleTreeMutable(hashes)
}

func BuildMerkleTreeTxs(txs []*golosovaniepb.Transaction) [HashSize]byte {
	hashes := make([][]byte, len(txs))
	if len(txs) == 0 {
		hashes = append(hashes, ZeroArrayHash[:])
	}
	for i, t := range txs {
		hashes[i] = t.Hash
	}
	return buildMerkleTreeMutable(hashes)
}

func buildMerkleTreeMutable(hashes [][]byte) [HashSize]byte {
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

func CreateBlock(
	transactions []*golosovaniepb.Transaction,
	prevHash []byte,
	timestamp time.Time,
	proposerPkey [PkeySize]byte,
) (*golosovaniepb.Block, error) {
	merkleTree := BuildMerkleTreeTxs(transactions)
	header := golosovaniepb.BlockHeader{
		PrevBlockHash: prevHash[:],
		MerkleTree:    merkleTree[:],
		ProposerPkey:  proposerPkey[:],
		Timestamp:     uint64(timestamp.UnixNano()),
	}
	headerBytes, err := proto.Marshal(&header)
	if err != nil {
		return nil, err
	}
	return &golosovaniepb.Block{
		BlockHeader:  &header,
		Transactions: transactions,
		Hash:         Hash(headerBytes),
	}, nil
}
