package evote

import (
	"GO_LOSOVANIE/evote/golosovaniepb"
	"bytes"
	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/testing/protocmp"
	"sort"
	"testing"
)

func blockHash(block *golosovaniepb.Block) []byte {
	headerBytes, err := proto.Marshal(block.BlockHeader)
	if err != nil {
		panic(err)
	}
	return Hash(headerBytes)
}

type SortTx []*golosovaniepb.Transaction

func (a SortTx) Len() int           { return len(a) }
func (a SortTx) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SortTx) Less(i, j int) bool { return bytes.Compare(a[i].Hash, a[j].Hash) == -1 }

type SortUtxo []*golosovaniepb.Utxo

func (a SortUtxo) Len() int      { return len(a) }
func (a SortUtxo) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a SortUtxo) Less(i, j int) bool {
	compare := bytes.Compare(a[i].TxHash, a[j].TxHash)
	if compare == 0 {
		return a[i].Index < a[j].Index
	} else {
		return compare == -1
	}
}

type SortBlock []*golosovaniepb.Block

func (a SortBlock) Len() int           { return len(a) }
func (a SortBlock) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SortBlock) Less(i, j int) bool { return bytes.Compare(a[i].Hash, a[j].Hash) == -1 }

func txsMatch(t *testing.T, expected []*golosovaniepb.Transaction, received []*golosovaniepb.Transaction) bool {
	sort.Sort(SortTx(expected))
	sort.Sort(SortTx(received))
	return assert.Zero(t, cmp.Diff(
		expected,
		received,
		protocmp.Transform(),
	))
}

func utxosMatch(t *testing.T, expected []*golosovaniepb.Utxo, received []*golosovaniepb.Utxo) bool {
	sort.Sort(SortUtxo(expected))
	sort.Sort(SortUtxo(received))
	return assert.Zero(t, cmp.Diff(
		expected,
		received,
		protocmp.Transform(),
	))
}

func blocksMatch(t *testing.T, expected []*golosovaniepb.Block, received []*golosovaniepb.Block) bool {
	sort.Sort(SortBlock(expected))
	sort.Sort(SortBlock(received))
	return assert.Zero(t, cmp.Diff(
		expected,
		received,
		protocmp.Transform(),
	))
}

func TestDatabase(t *testing.T) {
	//fmt.Printf("blocks: %+v\n%+v%+v\n", BLOCK0, BLOCK1, BLOCK2)
	var db Database
	err := db.Init(DbName, DbUser, DbPassword, DbHost, 5432)
	assert.Nil(t, err)
	t.Run("insert_blockz", func(t *testing.T) {
		err := db.SaveNextBlock(BlockZ)
		assert.Nil(t, err)
		blocksReceived, err := db.GetBlocksByHashes([][]byte{
			BlockZ.Hash,
		})
		assert.Nil(t, err)
		assert.Len(t, blocksReceived, 1)
		blocksMatch(
			t,
			[]*golosovaniepb.Block{
				BlockZ,
			},
			blocksReceived,
		)
		assert.Equal(t, blocksReceived[0].Hash, blockHash(blocksReceived[0]))
		assert.Equal(t, BlockZ.Hash, blockHash(blocksReceived[0]))
	})
	t.Run("insert_block0", func(t *testing.T) {
		err := db.SaveNextBlock(Block0)
		assert.Nil(t, err)
		blocksReceived, err := db.GetBlocksByHashes([][]byte{
			Block0.Hash,
		})
		assert.Nil(t, err)
		blocksMatch(
			t,
			[]*golosovaniepb.Block{
				Block0,
			},
			blocksReceived,
		)
		assert.Equal(t, blocksReceived[0].Hash, blockHash(blocksReceived[0]))
		assert.Equal(t, Block0.Hash, blockHash(blocksReceived[0]))
	})
	t.Run("get_tx_by_id_0", func(t *testing.T) {
		txHash := Block0.Transactions[0].Hash
		txsExpected := Block0.Transactions // there is only one tx, which we will receive
		txsReceived, err := db.GetTxsByHashes([][]byte{txHash})
		assert.Nil(t, err)
		txsMatch(t, txsExpected, txsReceived)
		assert.Equal(t, txsExpected[0].Hash, Hash(txsReceived[0].TxBody))
	})
	t.Run("get_tx_by_pkey_0", func(t *testing.T) {
		var body golosovaniepb.TxBody
		err := proto.Unmarshal(Block0.Transactions[0].TxBody, &body)
		assert.Nil(t, err)
		pkey := body.Outputs[0].ReceiverSpendPkey
		txsExpected := Block0.Transactions
		txsReceived, err := db.GetTxsByPubKey(pkey)
		assert.Nil(t, err)
		txsMatch(t, txsExpected, txsReceived)
		assert.Equal(t, txsExpected[0].Hash, Hash(txsReceived[0].TxBody))
	})
	t.Run("get_undefined_block_0", func(t *testing.T) {
		hashes := [][]byte{
			{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}, // несуществующий хеш блока
		}
		blocksExpected := make([]*golosovaniepb.Block, 0)
		blocksReceived, err := db.GetBlocksByHashes(hashes)
		assert.Nil(t, err)
		blocksMatch(t, blocksExpected, blocksReceived)
	})
	t.Run("get_undefined_tx_by_id_0", func(t *testing.T) {
		hashes := [][]byte{
			{1, 4, 8, 8, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}, // несуществующий хеш транзы
		}
		txsExpected := make([]*golosovaniepb.Transaction, 0)
		txsReceived, err := db.GetTxsByHashes(hashes)
		assert.Nil(t, err)
		txsMatch(t, txsExpected, txsReceived)
	})
	t.Run("get_undefined_tx_by_pkey", func(t *testing.T) {
		pkey := [PkeySize]byte{} // несуществующий публичный ключ
		txsExpected := make([]*golosovaniepb.Transaction, 0)
		txsReceived, err := db.GetTxsByPubKey(pkey[:])
		assert.Nil(t, err)
		txsMatch(t, txsExpected, txsReceived)
	})
	t.Run("double_block_insertion", func(t *testing.T) {
		err := db.SaveNextBlock(Block0)
		assert.Error(t, err)
	})
	t.Run("expect_duplicate_was_not_saved", func(t *testing.T) {
		blocksReceived, err := db.GetBlocksByHashes([][]byte{
			Block0.Hash,
		})
		assert.Nil(t, err)
		blocksMatch(
			t,
			[]*golosovaniepb.Block{
				Block0,
			},
			blocksReceived,
		)
	})
	t.Run("get_utxos_by_pkey_0", func(t *testing.T) {
		var body golosovaniepb.TxBody
		err := proto.Unmarshal(Block0.Transactions[0].TxBody, &body)
		assert.Nil(t, err)
		pkey := body.Outputs[0].ReceiverSpendPkey
		tx := Block0.Transactions[0]
		utxosExpected := []*golosovaniepb.Utxo{
			{
				TxHash:            tx.Hash,
				Index:             0,
				Value:             body.Outputs[0].Value,
				ReceiverSpendPkey: pkey,
				Timestamp:         Block0.BlockHeader.Timestamp,
			},
		}
		utxosReceived, err := db.GetUTXOSByPkey(pkey)
		assert.Nil(t, err)
		utxosMatch(t, utxosExpected, utxosReceived)
	})
	t.Run("get_utxos_by_txid_0", func(t *testing.T) {
		var body golosovaniepb.TxBody
		err := proto.Unmarshal(Block0.Transactions[0].TxBody, &body)
		assert.Nil(t, err)
		pkey := body.Outputs[0].ReceiverSpendPkey
		tx := Block0.Transactions[0]
		utxosExpected := []*golosovaniepb.Utxo{
			{
				TxHash:            tx.Hash,
				Index:             0,
				Value:             body.Outputs[0].Value,
				ReceiverSpendPkey: pkey,
				Timestamp:         Block0.BlockHeader.Timestamp,
			},
		}
		utxosReceived, err := db.GetUtxosByTxHash(tx.Hash)
		assert.Nil(t, err)
		utxosMatch(t, utxosExpected, utxosReceived)
	})
	t.Run("get_undefined_utxos_by_pkey_0", func(t *testing.T) {
		pkey := [PkeySize]byte{} // несуществующий публичный ключ
		utxosExpected := make([]*golosovaniepb.Utxo, 0)
		utxosReceived, err := db.GetUTXOSByPkey(pkey[:])
		assert.Nil(t, err)
		utxosMatch(t, utxosExpected, utxosReceived)
	})
	t.Run("get_undefined_utxos_by_txid_0", func(t *testing.T) {
		txid := [HashSize]byte{} // несуществующий txid
		utxosExpected := make([]*golosovaniepb.Utxo, 0)
		utxosReceived, err := db.GetUtxosByTxHash(txid[:])
		assert.Nil(t, err)
		utxosMatch(t, utxosExpected, utxosReceived)
	})
	t.Run("get_first_block", func(t *testing.T) {
		blockReceived, err := db.GetBlockAfter(nil)
		assert.Nil(t, err)
		assert.Zero(t, cmp.Diff(
			BlockZ,
			blockReceived,
			protocmp.Transform(),
		))
		assert.Equal(t, BlockZ.Hash, blockHash(blockReceived))
	})
	t.Run("get_undefined_first_block_0", func(t *testing.T) {
		hash := []byte{1, 2, 3} //несуществующий хеш
		blockReceived, err := db.GetBlockAfter(hash)
		assert.Nil(t, err)
		assert.Nil(t, blockReceived)
	})
	t.Run("get_undefined_first_block_1", func(t *testing.T) {
		hash := Block0.Hash //хеш первого блока, а второго еще нет
		blockReceived, err := db.GetBlockAfter(hash)
		assert.Nil(t, err)
		assert.Nil(t, blockReceived)
	})
	t.Run("insert_block_1", func(t *testing.T) {
		err := db.SaveNextBlock(Block1)
		assert.Nil(t, err)
		blocksReceived, err := db.GetBlocksByHashes([][]byte{
			Block1.Hash,
		})
		assert.Nil(t, err)
		blocksMatch(
			t,
			[]*golosovaniepb.Block{
				Block1,
			},
			blocksReceived,
		)
		assert.Equal(t, blocksReceived[0].Hash, blockHash(blocksReceived[0]))
		assert.Equal(t, Block1.Hash, blockHash(blocksReceived[0]))
	})
	t.Run("get_utxos_by_pkey_1", func(t *testing.T) {
		tx0 := Block1.Transactions[0]
		var body0 golosovaniepb.TxBody
		err := proto.Unmarshal(tx0.TxBody, &body0)
		assert.Nil(t, err)
		output0 := body0.Outputs[0]
		pkey := output0.ReceiverSpendPkey
		tx1 := Block1.Transactions[1]
		var body1 golosovaniepb.TxBody
		err = proto.Unmarshal(tx1.TxBody, &body1)
		assert.Nil(t, err)
		output1 := body1.Outputs[1]
		utxosExpected := []*golosovaniepb.Utxo{
			{
				TxHash:            tx0.Hash,
				Index:             0,
				Value:             output0.Value,
				ReceiverSpendPkey: output0.ReceiverSpendPkey,
				Timestamp:         Block1.BlockHeader.Timestamp,
			},
			{
				TxHash:            tx1.Hash,
				Index:             1,
				Value:             output1.Value,
				ReceiverSpendPkey: output1.ReceiverSpendPkey,
				Timestamp:         Block1.BlockHeader.Timestamp,
			},
		}
		utxosReceived, err := db.GetUTXOSByPkey(pkey)
		assert.Nil(t, err)
		utxosMatch(t, utxosExpected, utxosReceived)
	})
	t.Run("get_utxos_by_pkey_2", func(t *testing.T) {
		tx := Block1.Transactions[1]
		var body golosovaniepb.TxBody
		err := proto.Unmarshal(tx.TxBody, &body)
		output := body.Outputs[0]
		pkey := output.ReceiverSpendPkey
		utxosExpected := []*golosovaniepb.Utxo{
			{
				TxHash:            tx.Hash,
				Index:             0,
				Value:             output.Value,
				ReceiverSpendPkey: pkey,
				Timestamp:         Block1.BlockHeader.Timestamp,
			},
		}
		utxosReceived, err := db.GetUTXOSByPkey(pkey)
		assert.Nil(t, err)
		utxosMatch(t, utxosExpected, utxosReceived)
	})
	t.Run("get_utxos_by_txid_1", func(t *testing.T) {
		tx0 := Block1.Transactions[0]
		var body0 golosovaniepb.TxBody
		err := proto.Unmarshal(tx0.TxBody, &body0)
		assert.Nil(t, err)
		output0 := body0.Outputs[0]
		utxosExpected := []*golosovaniepb.Utxo{
			{
				TxHash:            tx0.Hash,
				Index:             0,
				Value:             output0.Value,
				ReceiverSpendPkey: output0.ReceiverSpendPkey,
				Timestamp:         Block1.BlockHeader.Timestamp,
			},
		}
		utxosReceived, err := db.GetUtxosByTxHash(tx0.Hash)
		assert.Nil(t, err)
		utxosMatch(t, utxosExpected, utxosReceived)
	})
	t.Run("get_utxos_by_txid_2", func(t *testing.T) {
		tx1 := Block1.Transactions[1]
		var body1 golosovaniepb.TxBody
		err := proto.Unmarshal(tx1.TxBody, &body1)
		assert.Nil(t, err)
		output0 := body1.Outputs[0]
		output1 := body1.Outputs[1]
		utxosExpected := []*golosovaniepb.Utxo{
			{
				TxHash:            tx1.Hash,
				Index:             0,
				Value:             output0.Value,
				ReceiverSpendPkey: output0.ReceiverSpendPkey,
				Timestamp:         Block1.BlockHeader.Timestamp,
			},
			{
				TxHash:            tx1.Hash,
				Index:             1,
				Value:             output1.Value,
				ReceiverSpendPkey: output1.ReceiverSpendPkey,
				Timestamp:         Block1.BlockHeader.Timestamp,
			},
		}
		utxosReceived, err := db.GetUtxosByTxHash(tx1.Hash)
		assert.Nil(t, err)
		utxosMatch(t, utxosExpected, utxosReceived)
	})
	t.Run("get_txs_by_pkey_1", func(t *testing.T) {
		var body golosovaniepb.TxBody
		err := proto.Unmarshal(Block0.Transactions[0].TxBody, &body)
		pkey := body.Outputs[0].ReceiverSpendPkey
		txsExpected := append(Block0.Transactions, Block1.Transactions...)
		txsReceived, err := db.GetTxsByPubKey(pkey)
		assert.Nil(t, err)
		txsMatch(t, txsExpected, txsReceived)
	})
	t.Run("get_undefined_utxos_by_txid_1", func(t *testing.T) {
		tx := Block0.Transactions[0]
		utxosExpected := make([]*golosovaniepb.Utxo, 0)
		utxosReceived, err := db.GetUtxosByTxHash(tx.Hash)
		assert.Nil(t, err)
		utxosMatch(t, utxosExpected, utxosReceived)
	})
	t.Run("get_second_block", func(t *testing.T) {
		block, err := db.GetBlockAfter(Block0.Hash)
		assert.Nil(t, err)
		assert.Zero(t, cmp.Diff(
			Block1,
			block,
			protocmp.Transform(),
		))
	})
	t.Run("get_undefined_block_1", func(t *testing.T) {
		block, err := db.GetBlockAfter(Block1.Hash) // третьего блока еще нет
		assert.Nil(t, err)
		assert.Nil(t, block)
	})
	t.Run("insert_block2", func(t *testing.T) {
		err := db.SaveNextBlock(Block2)
		assert.Nil(t, err)
		blocksReceived, err := db.GetBlocksByHashes([][]byte{
			Block2.Hash,
		})
		assert.Nil(t, err)
		blocksMatch(
			t,
			[]*golosovaniepb.Block{
				Block2,
			},
			blocksReceived,
		)
		assert.Equal(t, blocksReceived[0].Hash, blockHash(blocksReceived[0]))
		assert.Equal(t, Block2.Hash, blockHash(blocksReceived[0]))
	})
	t.Run("get_txs_by_id_1", func(t *testing.T) {
		hashes := [][]byte{
			Block1.Transactions[1].Hash,
			Block2.Transactions[0].Hash,
			Block2.Transactions[1].Hash,
		}
		txsExpected := []*golosovaniepb.Transaction{
			Block1.Transactions[1],
			Block2.Transactions[0],
			Block2.Transactions[1],
		}
		txsReceived, err := db.GetTxsByHashes(hashes)
		assert.Nil(t, err)
		txsMatch(t, txsExpected, txsReceived)
	})
	t.Run("get_blocks_by_hash", func(t *testing.T) {
		hashes := [][]byte{
			Block1.Hash,
			Block2.Hash,
		}
		blocksExpected := []*golosovaniepb.Block{
			Block1,
			Block2,
		}
		blocksReceived, err := db.GetBlocksByHashes(hashes)
		assert.Nil(t, err)
		blocksMatch(t, blocksExpected, blocksReceived)
	})
	t.Run("get_blocks_by_hash_but_not_all_valid", func(t *testing.T) {
		hashes := [][]byte{
			Block0.Hash,
			Block2.Hash,
			ZeroArrayHash[:], // блока с таким хешем нет
			{1, 2, 32},       // с таким тоже
		}
		blocksExpected := []*golosovaniepb.Block{
			Block0,
			Block2,
		}
		blocksReceived, err := db.GetBlocksByHashes(hashes)
		assert.Nil(t, err)
		blocksMatch(t, blocksExpected, blocksReceived)
	})
	t.Run("get_txs_by_hash_but_not_all_valid", func(t *testing.T) {
		hashes := [][]byte{
			Block0.Transactions[0].Hash,
			Block2.Transactions[0].Hash,
			ZeroArrayHash[:], // транзы с таким хешем нет
			{1, 3, 3, 7},     // с таким тоже
		}
		txsExpected := []*golosovaniepb.Transaction{
			Block0.Transactions[0],
			Block2.Transactions[0],
		}
		txsReceived, err := db.GetTxsByHashes(hashes)
		assert.Nil(t, err)
		txsMatch(t, txsExpected, txsReceived)
	})
	t.Run("get_undefined_utxos_by_pkey_1", func(t *testing.T) {
		var body golosovaniepb.TxBody
		err := proto.Unmarshal(Block0.Transactions[0].TxBody, &body)
		assert.Nil(t, err)
		pkey := body.Outputs[0].ReceiverSpendPkey
		utxosReceived, err := db.GetUTXOSByPkey(pkey)
		assert.Nil(t, err)
		assert.Empty(t, utxosReceived)
	})
	t.Run("get_utxos_by_pkey_3", func(t *testing.T) {
		tx0 := Block1.Transactions[1]
		var body0 golosovaniepb.TxBody
		err := proto.Unmarshal(tx0.TxBody, &body0)
		assert.Nil(t, err)
		output0 := body0.Outputs[0]
		tx1 := Block2.Transactions[0]
		var body1 golosovaniepb.TxBody
		err = proto.Unmarshal(tx1.TxBody, &body1)
		assert.Nil(t, err)
		output1 := body1.Outputs[0]
		pkey := output1.ReceiverSpendPkey
		utxosExpected := []*golosovaniepb.Utxo{
			{
				TxHash:            tx0.Hash,
				Index:             0,
				Value:             output0.Value,
				ReceiverSpendPkey: pkey,
				Timestamp:         Block1.BlockHeader.Timestamp,
			},
			{
				TxHash:            tx1.Hash,
				Index:             0,
				Value:             output1.Value,
				ReceiverSpendPkey: pkey,
				Timestamp:         Block2.BlockHeader.Timestamp,
			},
		}
		utxosReceived, err := db.GetUTXOSByPkey(pkey)
		assert.Nil(t, err)
		utxosMatch(t, utxosExpected, utxosReceived)
	})
	t.Run("get_txs_by_pkey_2", func(t *testing.T) {
		var body golosovaniepb.TxBody
		err := proto.Unmarshal(Block0.Transactions[0].TxBody, &body)
		assert.Nil(t, err)
		pkey := body.Outputs[0].ReceiverSpendPkey
		txsExpected := append([]*golosovaniepb.Transaction{}, Block0.Transactions...)
		txsExpected = append(txsExpected, Block1.Transactions...)
		txsExpected = append(txsExpected, Block2.Transactions[1:]...)
		txsReceived, err := db.GetTxsByPubKey(pkey)
		assert.Nil(t, err)
		txsMatch(t, txsExpected, txsReceived)
	})
	t.Run("insert_block3_with_optional_fields", func(t *testing.T) {
		err := db.SaveNextBlock(Block3)
		assert.Nil(t, err)
		blocksReceived, err := db.GetBlocksByHashes([][]byte{
			Block3.Hash,
		})
		assert.Nil(t, err)
		blocksMatch(
			t,
			[]*golosovaniepb.Block{
				Block3,
			},
			blocksReceived,
		)
		assert.Equal(t, blocksReceived[0].Hash, blockHash(blocksReceived[0]))
		assert.Equal(t, Block3.Hash, blockHash(blocksReceived[0]))
	})
	t.Run("get_utxo_by_pkey_with_typeValue_0", func(t *testing.T) {
		tx := Block3.Transactions[1]
		var body golosovaniepb.TxBody
		err := proto.Unmarshal(tx.TxBody, &body)
		assert.Nil(t, err)
		pkey := body.Outputs[0].ReceiverSpendPkey
		utxosExpected := []*golosovaniepb.Utxo{
			{
				TxHash:            tx.Hash,
				ValueType:         tx.Hash,
				Index:             0,
				Value:             2000,
				ReceiverSpendPkey: pkey,
				Timestamp:         Block3.BlockHeader.Timestamp,
			},
		}
		utxosReceived, err := db.GetUTXOSByPkey(pkey)
		assert.Nil(t, err)
		utxosMatch(t, utxosExpected, utxosReceived)
	})
	t.Run("get_utxo_by_txid_with_typeValue_0", func(t *testing.T) {
		tx := Block3.Transactions[1]
		var body golosovaniepb.TxBody
		err := proto.Unmarshal(tx.TxBody, &body)
		assert.Nil(t, err)
		utxosExpected := []*golosovaniepb.Utxo{
			{
				TxHash:            tx.Hash,
				ValueType:         tx.Hash,
				Index:             0,
				Value:             2000,
				ReceiverSpendPkey: body.Outputs[0].ReceiverSpendPkey,
				Timestamp:         Block3.BlockHeader.Timestamp,
			},
			{
				TxHash:            tx.Hash,
				ValueType:         tx.Hash,
				Index:             1,
				Value:             3000,
				ReceiverSpendPkey: body.Outputs[1].ReceiverSpendPkey,
				Timestamp:         Block3.BlockHeader.Timestamp,
			},
			{
				TxHash:            tx.Hash,
				ValueType:         tx.Hash,
				Index:             2,
				Value:             4000,
				ReceiverSpendPkey: body.Outputs[2].ReceiverSpendPkey,
				Timestamp:         Block3.BlockHeader.Timestamp,
			},
		}
		utxosReceived, err := db.GetUtxosByTxHash(tx.Hash)
		assert.Nil(t, err)
		utxosMatch(t, utxosExpected, utxosReceived)
	})
	t.Run("insert_block_4_with_more_optional_fields", func(t *testing.T) {
		err := db.SaveNextBlock(Block4)
		assert.Nil(t, err)
		blocksReceived, err := db.GetBlocksByHashes([][]byte{
			Block4.Hash,
		})
		assert.Nil(t, err)
		blocksMatch(
			t,
			[]*golosovaniepb.Block{
				Block4,
			},
			blocksReceived,
		)
		assert.Equal(t, blocksReceived[0].Hash, blockHash(blocksReceived[0]))
		assert.Equal(t, Block4.Hash, blockHash(blocksReceived[0]))
	})
	t.Run("get_utxo_by_pkey_with_typeValue_1", func(t *testing.T) {
		tx := Block4.Transactions[1]
		var body golosovaniepb.TxBody
		err := proto.Unmarshal(tx.TxBody, &body)
		assert.Nil(t, err)
		pkey := body.Outputs[0].ReceiverSpendPkey
		tx1 := Block4.Transactions[0]
		var body1 golosovaniepb.TxBody
		err = proto.Unmarshal(tx1.TxBody, &body1)
		assert.Nil(t, err)
		utxosExpected := []*golosovaniepb.Utxo{
			{
				TxHash:            tx1.Hash,
				Index:             0,
				Value:             3000,
				ReceiverSpendPkey: pkey,
				Timestamp:         Block4.BlockHeader.Timestamp,
			},
			{
				TxHash:            tx.Hash,
				ValueType:         Block3.Transactions[1].Hash,
				Index:             0,
				Value:             2000,
				ReceiverSpendPkey: pkey,
				Timestamp:         Block4.BlockHeader.Timestamp,
			},
		}
		utxosReceived, err := db.GetUTXOSByPkey(pkey)
		assert.Nil(t, err)
		utxosMatch(t, utxosExpected, utxosReceived)
	})
	t.Run("get_undefined_utxo_with_typeValue", func(t *testing.T) {
		tx := Block3.Transactions[1]
		var body golosovaniepb.TxBody
		err := proto.Unmarshal(tx.TxBody, &body)
		assert.Nil(t, err)
		pkey := body.Outputs[0].ReceiverSpendPkey
		utxosReceived, err := db.GetUTXOSByPkey(pkey)
		assert.Nil(t, err)
		assert.Empty(t, utxosReceived)
	})
	t.Run("get_utxo_by_txid_with_typeValue_1", func(t *testing.T) {
		tx := Block4.Transactions[1]
		var body golosovaniepb.TxBody
		err := proto.Unmarshal(tx.TxBody, &body)
		assert.Nil(t, err)
		utxosExpected := []*golosovaniepb.Utxo{
			{
				TxHash:            tx.Hash,
				ValueType:         Block3.Transactions[1].Hash,
				Index:             0,
				Value:             2000,
				ReceiverSpendPkey: body.Outputs[0].ReceiverSpendPkey,
				Timestamp:         Block4.BlockHeader.Timestamp,
			},
		}
		utxosReceived, err := db.GetUtxosByTxHash(tx.Hash)
		assert.Nil(t, err)
		utxosMatch(t, utxosExpected, utxosReceived)
	})
	t.Run("get_utxo_by_txid_with_typeValue_2", func(t *testing.T) {
		tx := Block3.Transactions[1]
		var body golosovaniepb.TxBody
		err := proto.Unmarshal(tx.TxBody, &body)
		assert.Nil(t, err)
		utxosExpected := []*golosovaniepb.Utxo{
			{
				TxHash:            tx.Hash,
				ValueType:         tx.Hash,
				Index:             1,
				Value:             3000,
				ReceiverSpendPkey: body.Outputs[1].ReceiverSpendPkey,
				Timestamp:         Block3.BlockHeader.Timestamp,
			},
			{
				TxHash:            tx.Hash,
				ValueType:         tx.Hash,
				Index:             2,
				Value:             4000,
				ReceiverSpendPkey: body.Outputs[2].ReceiverSpendPkey,
				Timestamp:         Block3.BlockHeader.Timestamp,
			},
		}
		utxosReceived, err := db.GetUtxosByTxHash(tx.Hash)
		assert.Nil(t, err)
		utxosMatch(t, utxosExpected, utxosReceived)
	})
	err = db.Close()
	assert.Nil(t, err)
}
