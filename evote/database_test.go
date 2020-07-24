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
			B_AND_H0.hash,
		})
		assert.Nil(t, err)
		assert.ElementsMatch(t, []*BlocAndkHash{
			&B_AND_H0,
		}, blocksReceived)
		assert.Equal(t, blocksReceived[0].hash, blockHash(blocksReceived[0].b))
		assert.Equal(t, B_AND_H0.hash, blockHash(blocksReceived[0].b))
	})
	t.Run("get_tx_by_id_0", func(t *testing.T) {
		txid := BLOCK0.trans[0].hash
		txsExpected := BLOCK0.trans // there is only one tx, which we will receive
		txsReceived, err := db.GetTxsByHashes([][HASH_SIZE]byte{txid})
		assert.Nil(t, err)
		assert.Equal(t, txsExpected, txsReceived)
		assert.Equal(t, txsExpected[0].hash, txHash(txsReceived[0].transaction))
	})
	t.Run("get_tx_by_pkey_0", func(t *testing.T) {
		pkey := BLOCK0.trans[0].transaction.outputs[0].pkeyTo
		txsExpected := BLOCK0.trans
		txsReceived, err := db.GetTxsByPubKey(pkey)
		assert.Nil(t, err)
		assert.Equal(t, txsExpected, txsReceived)
		assert.Equal(t, txsExpected[0].hash, txHash(txsReceived[0].transaction))
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
			B_AND_H0.hash,
		})
		assert.Nil(t, err)
		assert.ElementsMatch(t, []*BlocAndkHash{
			&B_AND_H0,
		}, blocksReceived)
	})
	t.Run("get_utxos_by_pkey_0", func(t *testing.T) {
		pkey := BLOCK0.trans[0].transaction.outputs[0].pkeyTo
		tx := BLOCK0.trans[0]
		utxosExpected := []*UTXO{
			{
				txId:   tx.hash,
				index:  0,
				value:  tx.transaction.outputs[0].value,
				pkeyTo: pkey,
			},
		}
		utxosReceived, err := db.GetUTXOSByPkey(pkey)
		assert.Nil(t, err)
		assert.Equal(t, utxosExpected, utxosReceived)
	})
	t.Run("get_utxos_by_txid_0", func(t *testing.T) {
		pkey := BLOCK0.trans[0].transaction.outputs[0].pkeyTo
		tx := BLOCK0.trans[0]
		utxosExpected := []*UTXO{
			{
				txId:   tx.hash,
				index:  0,
				value:  tx.transaction.outputs[0].value,
				pkeyTo: pkey,
			},
		}
		utxosReceived, err := db.GetUTXOSByTxId(tx.hash)
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
		assert.Equal(t, B_AND_H0.hash, blockHash(blockReceived.b))
	})
	t.Run("get_undefined_first_block_0", func(t *testing.T) {
		hash := [HASH_SIZE]byte{1, 2, 3} //несуществующий хеш
		blockReceived, err := db.GetBlockAfter(hash)
		assert.Nil(t, err)
		assert.Nil(t, blockReceived)
	})
	t.Run("get_undefined_first_block_1", func(t *testing.T) {
		hash := B_AND_H0.hash //хеш первого блока, а второго еще нет
		blockReceived, err := db.GetBlockAfter(hash)
		assert.Nil(t, err)
		assert.Nil(t, blockReceived)
	})
	t.Run("insert_block_1", func(t *testing.T) {
		err := db.SaveNextBlock(&B_AND_H1)
		assert.Nil(t, err)
		blocksReceived, err := db.GetBlocksByHashes([][HASH_SIZE]byte{
			B_AND_H1.hash,
		})
		assert.Nil(t, err)
		assert.ElementsMatch(t, []*BlocAndkHash{
			&B_AND_H1,
		}, blocksReceived)
		assert.Equal(t, blocksReceived[0].hash, blockHash(blocksReceived[0].b))
		assert.Equal(t, B_AND_H1.hash, blockHash(blocksReceived[0].b))
	})
	t.Run("get_utxos_by_pkey_1", func(t *testing.T) {
		tx0 := BLOCK1.trans[0]
		output0 := tx0.transaction.outputs[0]
		pkey := output0.pkeyTo
		tx1 := BLOCK1.trans[1]
		output1 := tx1.transaction.outputs[1]
		utxosExpected := []*UTXO{
			{
				txId:   tx0.hash,
				index:  0,
				value:  output0.value,
				pkeyTo: output0.pkeyTo,
			},
			{
				txId:   tx1.hash,
				index:  1,
				value:  output1.value,
				pkeyTo: output1.pkeyTo,
			},
		}
		utxosReceived, err := db.GetUTXOSByPkey(pkey)
		assert.Nil(t, err)
		assert.ElementsMatch(t, utxosExpected, utxosReceived)
	})
	t.Run("get_utxos_by_pkey_2", func(t *testing.T) {
		tx := BLOCK1.trans[1]
		output := tx.transaction.outputs[0]
		pkey := output.pkeyTo
		utxosExpected := []*UTXO{
			{
				txId:   tx.hash,
				index:  0,
				value:  output.value,
				pkeyTo: pkey,
			},
		}
		utxosReceived, err := db.GetUTXOSByPkey(pkey)
		assert.Nil(t, err)
		assert.ElementsMatch(t, utxosExpected, utxosReceived)
	})
	t.Run("get_utxos_by_txid_1", func(t *testing.T) {
		tx0 := BLOCK1.trans[0]
		output0 := tx0.transaction.outputs[0]
		utxosExpected := []*UTXO{
			{
				txId:   tx0.hash,
				index:  0,
				value:  output0.value,
				pkeyTo: output0.pkeyTo,
			},
		}
		utxosReceived, err := db.GetUTXOSByTxId(tx0.hash)
		assert.Nil(t, err)
		assert.Equal(t, utxosExpected, utxosReceived)
	})
	t.Run("get_utxos_by_txid_2", func(t *testing.T) {
		tx1 := BLOCK1.trans[1]
		output0 := tx1.transaction.outputs[0]
		output1 := tx1.transaction.outputs[1]
		utxosExpected := []*UTXO{
			{
				txId:   tx1.hash,
				index:  0,
				value:  output0.value,
				pkeyTo: output0.pkeyTo,
			},
			{
				txId:   tx1.hash,
				index:  1,
				value:  output1.value,
				pkeyTo: output1.pkeyTo,
			},
		}
		utxosReceived, err := db.GetUTXOSByTxId(tx1.hash)
		assert.Nil(t, err)
		assert.ElementsMatch(t, utxosExpected, utxosReceived)
	})
	t.Run("get_txs_by_pkey_1", func(t *testing.T) {
		pkey := BLOCK0.trans[0].transaction.outputs[0].pkeyTo
		txsExpected := append(BLOCK0.trans, BLOCK1.trans...)
		txsReceived, err := db.GetTxsByPubKey(pkey)
		assert.Nil(t, err)
		assert.ElementsMatch(t, txsExpected, txsReceived)
	})
	t.Run("get_undefined_utxos_by_txid_1", func(t *testing.T) {
		tx := BLOCK0.trans[0]
		utxosExpected := make([]*UTXO, 0)
		utxosReceived, err := db.GetUTXOSByTxId(tx.hash)
		assert.Nil(t, err)
		assert.Equal(t, utxosExpected, utxosReceived)
	})
	t.Run("get_second_block", func(t *testing.T) {
		block, err := db.GetBlockAfter(B_AND_H0.hash)
		assert.Nil(t, err)
		assert.Equal(t, &B_AND_H1, block)
	})
	t.Run("get_undefined_block_1", func(t *testing.T) {
		block, err := db.GetBlockAfter(B_AND_H1.hash) // третьего блока еще нет
		assert.Nil(t, err)
		assert.Nil(t, block)
	})
	t.Run("insert_block2", func(t *testing.T) {
		err := db.SaveNextBlock(&B_AND_H2)
		assert.Nil(t, err)
		blocksReceived, err := db.GetBlocksByHashes([][HASH_SIZE]byte{
			B_AND_H2.hash,
		})
		assert.Nil(t, err)
		assert.ElementsMatch(t, []*BlocAndkHash{
			&B_AND_H2,
		}, blocksReceived)
		assert.Equal(t, blocksReceived[0].hash, blockHash(blocksReceived[0].b))
		assert.Equal(t, B_AND_H2.hash, blockHash(blocksReceived[0].b))
	})
	t.Run("get_txs_by_id_1", func(t *testing.T) {
		hashes := [][HASH_SIZE]byte{
			BLOCK1.trans[1].hash,
			BLOCK2.trans[0].hash,
			BLOCK2.trans[1].hash,
		}
		txsExpected := []TransAndHash{
			BLOCK1.trans[1],
			BLOCK2.trans[0],
			BLOCK2.trans[1],
		}
		txsReceived, err := db.GetTxsByHashes(hashes)
		assert.Nil(t, err)
		assert.ElementsMatch(t, txsExpected, txsReceived)
	})
	t.Run("get_blocks_by_hash", func(t *testing.T) {
		hashes := [][HASH_SIZE]byte{
			B_AND_H1.hash,
			B_AND_H2.hash,
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
			B_AND_H0.hash,
			B_AND_H2.hash,
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
			BLOCK0.trans[0].hash,
			BLOCK2.trans[0].hash,
			ZERO_ARRAY_HASH, // транзы с таким хешем нет
			{1, 3, 3, 7},    // с таким тоже
		}
		txsExpected := []TransAndHash{
			BLOCK0.trans[0],
			BLOCK2.trans[0],
		}
		txsReceived, err := db.GetTxsByHashes(hashes)
		assert.Nil(t, err)
		assert.ElementsMatch(t, txsExpected, txsReceived)
	})
	t.Run("get_undefined_utxos_by_pkey_1", func(t *testing.T) {
		pkey := BLOCK0.trans[0].transaction.outputs[0].pkeyTo
		utxosReceived, err := db.GetUTXOSByPkey(pkey)
		assert.Nil(t, err)
		assert.Empty(t, utxosReceived)
	})
	t.Run("get_utxos_by_pkey_3", func(t *testing.T) {
		pkey := BLOCK2.trans[0].transaction.outputs[0].pkeyTo
		tx0 := BLOCK1.trans[1]
		output0 := tx0.transaction.outputs[0]
		tx1 := BLOCK2.trans[0]
		output1 := tx1.transaction.outputs[0]
		utxosExpected := []*UTXO{
			{
				txId:   tx0.hash,
				index:  0,
				value:  output0.value,
				pkeyTo: pkey,
			},
			{
				txId:   tx1.hash,
				index:  0,
				value:  output1.value,
				pkeyTo: pkey,
			},
		}
		utxosReceived, err := db.GetUTXOSByPkey(pkey)
		assert.Nil(t, err)
		assert.ElementsMatch(t, utxosExpected, utxosReceived)
	})
	t.Run("get_txs_by_pkey_2", func(t *testing.T) {
		pkey := BLOCK0.trans[0].transaction.outputs[0].pkeyTo
		txsExpected := append([]TransAndHash{}, BLOCK0.trans...)
		txsExpected = append(txsExpected, BLOCK1.trans...)
		txsExpected = append(txsExpected, BLOCK2.trans[1:]...)
		txsReceived, err := db.GetTxsByPubKey(pkey)
		assert.Nil(t, err)
		assert.ElementsMatch(t, txsExpected, txsReceived)
	})
	t.Run("insert_block3_with_optional_fields", func(t *testing.T) {
		err := db.SaveNextBlock(&B_AND_H3)
		assert.Nil(t, err)
		blocksReceived, err := db.GetBlocksByHashes([][HASH_SIZE]byte{
			B_AND_H3.hash,
		})
		assert.Nil(t, err)
		assert.ElementsMatch(t, []*BlocAndkHash{
			&B_AND_H3,
		}, blocksReceived)
		assert.Equal(t, blocksReceived[0].hash, blockHash(blocksReceived[0].b))
		assert.Equal(t, B_AND_H3.hash, blockHash(blocksReceived[0].b))
	})
	t.Run("insert_block_4_with_more_optional_fields", func(t *testing.T) {
		err := db.SaveNextBlock(&B_AND_H4)
		assert.Nil(t, err)
		blocksReceived, err := db.GetBlocksByHashes([][HASH_SIZE]byte{
			B_AND_H4.hash,
		})
		assert.Nil(t, err)
		assert.ElementsMatch(t, []*BlocAndkHash{
			&B_AND_H4,
		}, blocksReceived)
		assert.Equal(t, blocksReceived[0].hash, blockHash(blocksReceived[0].b))
		assert.Equal(t, B_AND_H4.hash, blockHash(blocksReceived[0].b))
	})
	err = db.Close()
	assert.Nil(t, err)
}
