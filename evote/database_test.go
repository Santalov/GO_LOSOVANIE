package evote

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDatabase(t *testing.T) {
	//fmt.Printf("blocks: %+v\n%+v%+v\n", BLOCK0, BLOCK1, BLOCK2)
	var db Database
	err := db.Init(DBNAME, DBUSER, DBPASSWORD, DBHOST, 5432)
	assert.Nil(t, err)
	t.Run("insert_block0", func(t *testing.T) {
		err := db.SaveNextBlock(&B_AND_H0)
		assert.Nil(t, err)
		blocksReceived, err := db.GetBlocksByHashes([][HASH_SIZE]byte{
			B_AND_H0.Hash,
		})
		assert.Nil(t, err)
		assert.ElementsMatch(t, []*BlocAndkHash{
			&B_AND_H0,
		}, blocksReceived)
		assert.Equal(t, blocksReceived[0].Hash, blockHash(blocksReceived[0].B))
		assert.Equal(t, B_AND_H0.Hash, blockHash(blocksReceived[0].B))
	})
	t.Run("get_tx_by_id_0", func(t *testing.T) {
		txid := BLOCK0.Trans[0].Hash
		txsExpected := BLOCK0.Trans // there is only one tx, which we will receive
		txsReceived, err := db.GetTxsByHashes([][HASH_SIZE]byte{txid})
		assert.Nil(t, err)
		assert.Equal(t, txsExpected, txsReceived)
		assert.Equal(t, txsExpected[0].Hash, txHash(txsReceived[0].Transaction))
	})
	t.Run("get_tx_by_pkey_0", func(t *testing.T) {
		pkey := BLOCK0.Trans[0].Transaction.Outputs[0].PkeyTo
		txsExpected := BLOCK0.Trans
		txsReceived, err := db.GetTxsByPubKey(pkey)
		assert.Nil(t, err)
		assert.Equal(t, txsExpected, txsReceived)
		assert.Equal(t, txsExpected[0].Hash, txHash(txsReceived[0].Transaction))
	})
	t.Run("get_undefined_block_0", func(t *testing.T) {
		hashes := [][HASH_SIZE]byte{
			{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}, // несуществующий хеш блока
		}
		blocksExpected := make([]*BlocAndkHash, 0)
		blocksReceived, err := db.GetBlocksByHashes(hashes)
		assert.Nil(t, err)
		assert.Equal(t, blocksExpected, blocksReceived)
	})
	t.Run("get_undefined_tx_by_id_0", func(t *testing.T) {
		hashes := [][HASH_SIZE]byte{
			{1, 4, 8, 8, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}, // несуществующий хеш транзы
		}
		txsExpected := make([]TransAndHash, 0)
		txsReceived, err := db.GetTxsByHashes(hashes)
		assert.Nil(t, err)
		assert.Equal(t, txsExpected, txsReceived)
	})
	t.Run("get_undefined_tx_by_pkey", func(t *testing.T) {
		pkey := [PKEY_SIZE]byte{} // несуществующий публичный ключ
		txsExpected := make([]TransAndHash, 0)
		txsReceived, err := db.GetTxsByPubKey(pkey)
		assert.Nil(t, err)
		assert.Equal(t, txsExpected, txsReceived)
	})
	t.Run("double_block_insertion", func(t *testing.T) {
		err := db.SaveNextBlock(&B_AND_H0)
		assert.Error(t, err)
	})
	t.Run("expect_duplicate_was_not_saved", func(t *testing.T) {
		blocksReceived, err := db.GetBlocksByHashes([][HASH_SIZE]byte{
			B_AND_H0.Hash,
		})
		assert.Nil(t, err)
		assert.ElementsMatch(t, []*BlocAndkHash{
			&B_AND_H0,
		}, blocksReceived)
	})
	t.Run("get_utxos_by_pkey_0", func(t *testing.T) {
		pkey := BLOCK0.Trans[0].Transaction.Outputs[0].PkeyTo
		tx := BLOCK0.Trans[0]
		utxosExpected := []*UTXO{
			{
				TxId:   tx.Hash,
				Index:  0,
				Value:  tx.Transaction.Outputs[0].Value,
				PkeyTo: pkey,
			},
		}
		utxosReceived, err := db.GetUTXOSByPkey(pkey)
		assert.Nil(t, err)
		assert.Equal(t, utxosExpected, utxosReceived)
	})
	t.Run("get_utxos_by_txid_0", func(t *testing.T) {
		pkey := BLOCK0.Trans[0].Transaction.Outputs[0].PkeyTo
		tx := BLOCK0.Trans[0]
		utxosExpected := []*UTXO{
			{
				TxId:   tx.Hash,
				Index:  0,
				Value:  tx.Transaction.Outputs[0].Value,
				PkeyTo: pkey,
			},
		}
		utxosReceived, err := db.GetUTXOSByTxId(tx.Hash)
		assert.Nil(t, err)
		assert.Equal(t, utxosExpected, utxosReceived)
	})
	t.Run("get_undefined_utxos_by_pkey_0", func(t *testing.T) {
		pkey := [PKEY_SIZE]byte{} // несуществующий публичный ключ
		utxosExpected := make([]*UTXO, 0)
		utxosReceived, err := db.GetUTXOSByPkey(pkey)
		assert.Nil(t, err)
		assert.Equal(t, utxosExpected, utxosReceived)
	})
	t.Run("get_undefined_utxos_by_txid_0", func(t *testing.T) {
		txid := [HASH_SIZE]byte{} // несуществующий txid
		utxosExpected := make([]*UTXO, 0)
		utxosReceived, err := db.GetUTXOSByTxId(txid)
		assert.Nil(t, err)
		assert.Equal(t, utxosExpected, utxosReceived)
	})
	t.Run("get_first_block", func(t *testing.T) {
		hash := ZERO_ARRAY_HASH
		blockReceived, err := db.GetBlockAfter(hash)
		assert.Nil(t, err)
		assert.Equal(t, &B_AND_H0, blockReceived)
		assert.Equal(t, B_AND_H0.Hash, blockHash(blockReceived.B))
	})
	t.Run("get_undefined_first_block_0", func(t *testing.T) {
		hash := [HASH_SIZE]byte{1, 2, 3} //несуществующий хеш
		blockReceived, err := db.GetBlockAfter(hash)
		assert.Nil(t, err)
		assert.Nil(t, blockReceived)
	})
	t.Run("get_undefined_first_block_1", func(t *testing.T) {
		hash := B_AND_H0.Hash //хеш первого блока, а второго еще нет
		blockReceived, err := db.GetBlockAfter(hash)
		assert.Nil(t, err)
		assert.Nil(t, blockReceived)
	})
	t.Run("insert_block_1", func(t *testing.T) {
		err := db.SaveNextBlock(&B_AND_H1)
		assert.Nil(t, err)
		blocksReceived, err := db.GetBlocksByHashes([][HASH_SIZE]byte{
			B_AND_H1.Hash,
		})
		assert.Nil(t, err)
		assert.ElementsMatch(t, []*BlocAndkHash{
			&B_AND_H1,
		}, blocksReceived)
		assert.Equal(t, blocksReceived[0].Hash, blockHash(blocksReceived[0].B))
		assert.Equal(t, B_AND_H1.Hash, blockHash(blocksReceived[0].B))
	})
	t.Run("get_utxos_by_pkey_1", func(t *testing.T) {
		tx0 := BLOCK1.Trans[0]
		output0 := tx0.Transaction.Outputs[0]
		pkey := output0.PkeyTo
		tx1 := BLOCK1.Trans[1]
		output1 := tx1.Transaction.Outputs[1]
		utxosExpected := []*UTXO{
			{
				TxId:   tx0.Hash,
				Index:  0,
				Value:  output0.Value,
				PkeyTo: output0.PkeyTo,
			},
			{
				TxId:   tx1.Hash,
				Index:  1,
				Value:  output1.Value,
				PkeyTo: output1.PkeyTo,
			},
		}
		utxosReceived, err := db.GetUTXOSByPkey(pkey)
		assert.Nil(t, err)
		assert.ElementsMatch(t, utxosExpected, utxosReceived)
	})
	t.Run("get_utxos_by_pkey_2", func(t *testing.T) {
		tx := BLOCK1.Trans[1]
		output := tx.Transaction.Outputs[0]
		pkey := output.PkeyTo
		utxosExpected := []*UTXO{
			{
				TxId:   tx.Hash,
				Index:  0,
				Value:  output.Value,
				PkeyTo: pkey,
			},
		}
		utxosReceived, err := db.GetUTXOSByPkey(pkey)
		assert.Nil(t, err)
		assert.ElementsMatch(t, utxosExpected, utxosReceived)
	})
	t.Run("get_utxos_by_txid_1", func(t *testing.T) {
		tx0 := BLOCK1.Trans[0]
		output0 := tx0.Transaction.Outputs[0]
		utxosExpected := []*UTXO{
			{
				TxId:   tx0.Hash,
				Index:  0,
				Value:  output0.Value,
				PkeyTo: output0.PkeyTo,
			},
		}
		utxosReceived, err := db.GetUTXOSByTxId(tx0.Hash)
		assert.Nil(t, err)
		assert.Equal(t, utxosExpected, utxosReceived)
	})
	t.Run("get_utxos_by_txid_2", func(t *testing.T) {
		tx1 := BLOCK1.Trans[1]
		output0 := tx1.Transaction.Outputs[0]
		output1 := tx1.Transaction.Outputs[1]
		utxosExpected := []*UTXO{
			{
				TxId:   tx1.Hash,
				Index:  0,
				Value:  output0.Value,
				PkeyTo: output0.PkeyTo,
			},
			{
				TxId:   tx1.Hash,
				Index:  1,
				Value:  output1.Value,
				PkeyTo: output1.PkeyTo,
			},
		}
		utxosReceived, err := db.GetUTXOSByTxId(tx1.Hash)
		assert.Nil(t, err)
		assert.ElementsMatch(t, utxosExpected, utxosReceived)
	})
	t.Run("get_txs_by_pkey_1", func(t *testing.T) {
		pkey := BLOCK0.Trans[0].Transaction.Outputs[0].PkeyTo
		txsExpected := append(BLOCK0.Trans, BLOCK1.Trans...)
		txsReceived, err := db.GetTxsByPubKey(pkey)
		assert.Nil(t, err)
		assert.ElementsMatch(t, txsExpected, txsReceived)
	})
	t.Run("get_undefined_utxos_by_txid_1", func(t *testing.T) {
		tx := BLOCK0.Trans[0]
		utxosExpected := make([]*UTXO, 0)
		utxosReceived, err := db.GetUTXOSByTxId(tx.Hash)
		assert.Nil(t, err)
		assert.Equal(t, utxosExpected, utxosReceived)
	})
	t.Run("get_second_block", func(t *testing.T) {
		block, err := db.GetBlockAfter(B_AND_H0.Hash)
		assert.Nil(t, err)
		assert.Equal(t, &B_AND_H1, block)
	})
	t.Run("get_undefined_block_1", func(t *testing.T) {
		block, err := db.GetBlockAfter(B_AND_H1.Hash) // третьего блока еще нет
		assert.Nil(t, err)
		assert.Nil(t, block)
	})
	t.Run("insert_block2", func(t *testing.T) {
		err := db.SaveNextBlock(&B_AND_H2)
		assert.Nil(t, err)
		blocksReceived, err := db.GetBlocksByHashes([][HASH_SIZE]byte{
			B_AND_H2.Hash,
		})
		assert.Nil(t, err)
		assert.ElementsMatch(t, []*BlocAndkHash{
			&B_AND_H2,
		}, blocksReceived)
		assert.Equal(t, blocksReceived[0].Hash, blockHash(blocksReceived[0].B))
		assert.Equal(t, B_AND_H2.Hash, blockHash(blocksReceived[0].B))
	})
	t.Run("get_txs_by_id_1", func(t *testing.T) {
		hashes := [][HASH_SIZE]byte{
			BLOCK1.Trans[1].Hash,
			BLOCK2.Trans[0].Hash,
			BLOCK2.Trans[1].Hash,
		}
		txsExpected := []TransAndHash{
			BLOCK1.Trans[1],
			BLOCK2.Trans[0],
			BLOCK2.Trans[1],
		}
		txsReceived, err := db.GetTxsByHashes(hashes)
		assert.Nil(t, err)
		assert.ElementsMatch(t, txsExpected, txsReceived)
	})
	t.Run("get_blocks_by_hash", func(t *testing.T) {
		hashes := [][HASH_SIZE]byte{
			B_AND_H1.Hash,
			B_AND_H2.Hash,
		}
		blocksExpected := []*BlocAndkHash{
			&B_AND_H1,
			&B_AND_H2,
		}
		blocksReceived, err := db.GetBlocksByHashes(hashes)
		assert.Nil(t, err)
		assert.ElementsMatch(t, blocksExpected, blocksReceived)
	})
	t.Run("get_blocks_by_hash_but_not_all_valid", func(t *testing.T) {
		hashes := [][HASH_SIZE]byte{
			B_AND_H0.Hash,
			B_AND_H2.Hash,
			ZERO_ARRAY_HASH, // блока с таким хешем нет
			{1, 2, 32},      // с таким тоже
		}
		blocksExpected := []*BlocAndkHash{
			&B_AND_H0,
			&B_AND_H2,
		}
		blocksReceived, err := db.GetBlocksByHashes(hashes)
		assert.Nil(t, err)
		assert.ElementsMatch(t, blocksExpected, blocksReceived)
	})
	t.Run("get_txs_by_hash_but_not_all_valid", func(t *testing.T) {
		hashes := [][HASH_SIZE]byte{
			BLOCK0.Trans[0].Hash,
			BLOCK2.Trans[0].Hash,
			ZERO_ARRAY_HASH, // транзы с таким хешем нет
			{1, 3, 3, 7},    // с таким тоже
		}
		txsExpected := []TransAndHash{
			BLOCK0.Trans[0],
			BLOCK2.Trans[0],
		}
		txsReceived, err := db.GetTxsByHashes(hashes)
		assert.Nil(t, err)
		assert.ElementsMatch(t, txsExpected, txsReceived)
	})
	t.Run("get_undefined_utxos_by_pkey_1", func(t *testing.T) {
		pkey := BLOCK0.Trans[0].Transaction.Outputs[0].PkeyTo
		utxosReceived, err := db.GetUTXOSByPkey(pkey)
		assert.Nil(t, err)
		assert.Empty(t, utxosReceived)
	})
	t.Run("get_utxos_by_pkey_3", func(t *testing.T) {
		pkey := BLOCK2.Trans[0].Transaction.Outputs[0].PkeyTo
		tx0 := BLOCK1.Trans[1]
		output0 := tx0.Transaction.Outputs[0]
		tx1 := BLOCK2.Trans[0]
		output1 := tx1.Transaction.Outputs[0]
		utxosExpected := []*UTXO{
			{
				TxId:   tx0.Hash,
				Index:  0,
				Value:  output0.Value,
				PkeyTo: pkey,
			},
			{
				TxId:   tx1.Hash,
				Index:  0,
				Value:  output1.Value,
				PkeyTo: pkey,
			},
		}
		utxosReceived, err := db.GetUTXOSByPkey(pkey)
		assert.Nil(t, err)
		assert.ElementsMatch(t, utxosExpected, utxosReceived)
	})
	t.Run("get_txs_by_pkey_2", func(t *testing.T) {
		pkey := BLOCK0.Trans[0].Transaction.Outputs[0].PkeyTo
		txsExpected := append([]TransAndHash{}, BLOCK0.Trans...)
		txsExpected = append(txsExpected, BLOCK1.Trans...)
		txsExpected = append(txsExpected, BLOCK2.Trans[1:]...)
		txsReceived, err := db.GetTxsByPubKey(pkey)
		assert.Nil(t, err)
		assert.ElementsMatch(t, txsExpected, txsReceived)
	})
	t.Run("insert_block3_with_optional_fields", func(t *testing.T) {
		err := db.SaveNextBlock(&B_AND_H3)
		assert.Nil(t, err)
		blocksReceived, err := db.GetBlocksByHashes([][HASH_SIZE]byte{
			B_AND_H3.Hash,
		})
		assert.Nil(t, err)
		assert.ElementsMatch(t, []*BlocAndkHash{
			&B_AND_H3,
		}, blocksReceived)
		assert.Equal(t, blocksReceived[0].Hash, blockHash(blocksReceived[0].B))
		assert.Equal(t, B_AND_H3.Hash, blockHash(blocksReceived[0].B))
	})
	t.Run("insert_block_4_with_more_optional_fields", func(t *testing.T) {
		err := db.SaveNextBlock(&B_AND_H4)
		assert.Nil(t, err)
		blocksReceived, err := db.GetBlocksByHashes([][HASH_SIZE]byte{
			B_AND_H4.Hash,
		})
		assert.Nil(t, err)
		assert.ElementsMatch(t, []*BlocAndkHash{
			&B_AND_H4,
		}, blocksReceived)
		assert.Equal(t, blocksReceived[0].Hash, blockHash(blocksReceived[0].B))
		assert.Equal(t, B_AND_H4.Hash, blockHash(blocksReceived[0].B))
	})
	err = db.Close()
	assert.Nil(t, err)
}
