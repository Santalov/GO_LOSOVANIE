package evote

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"strconv"
	"strings"
)

// функции ниже нужны, чтоб обрабатывать необязательные поля.
// В байтовом представлении считается, что отсутствующие необязательные поля принимают нулевое значение и это норм
// А в бд такое соглашение приводит к ошибкам
// при поддержании ссылочной целостности и запросам с условием на отсутствие полей
//
// Поэтому необязтельные поля с нулевыми значениями преобразуются в nil, который бд преобразует в null
// Еще одна фича, у массиовов нет значения nil, поэтому они сканируются как срезы

// Transaction.TypeVote и Transaction.TypeValue не преобразуются к nil, так как на них не завязана ссылочная целостность
// и преобразование к nil добавляет дополнительную сложность

func hashToSlice(v [HASH_SIZE]byte) interface{} {
	if v == [HASH_SIZE]byte{} {
		return nil
	} else {
		return v[:]
	}
}

func sliceToHash(v []byte) [HASH_SIZE]byte {
	if v == nil {
		return [HASH_SIZE]byte{}
	} else {
		var hash [HASH_SIZE]byte
		copy(hash[:], v)
		return hash
	}
}

func buildInLookup(from int, to int) string {
	inLookupBuilder := strings.Builder{}
	for i := from; i < to; i++ {
		inLookupBuilder.WriteString("$" + strconv.Itoa(i))
		if i != to-1 {
			inLookupBuilder.WriteString(", ")
		}
	}
	return inLookupBuilder.String()
}

// не откатывает транзу при ошибке
func getTxInputsAndOutputs(dbTx *sql.Tx, txHash [HASH_SIZE]byte) ([]TransactionInput, []TransactionOutput, error) {
	var inputs []TransactionInput
	var outputs []TransactionOutput
	inputRows, err := dbTx.Query(
		`SELECT prevtxid, outputindex 
		FROM input WHERE txid = $1 ORDER BY Index`,
		txHash[:],
	)
	if err != nil {
		return nil, nil, err
	}
	for inputRows.Next() {
		var input TransactionInput
		var prevId []byte
		err := inputRows.Scan(&prevId, &input.OutIndex)
		if err != nil {
			return nil, nil, err
		}
		copy(input.PrevId[:], prevId)
		inputs = append(inputs, input)
	}
	err = inputRows.Close()
	if err != nil {
		return nil, nil, err
	}
	outputRows, err := dbTx.Query(
		`SELECT Value, publickeyto 
		FROM output WHERE txid = $1 ORDER BY Index`,
		txHash[:],
	)
	if err != nil {
		return nil, nil, err
	}
	for outputRows.Next() {
		var output TransactionOutput
		var pkey []byte
		err := outputRows.Scan(&output.Value, &pkey)
		if err != nil {
			return nil, nil, err
		}
		copy(output.PkeyTo[:], pkey)
		outputs = append(outputs, output)
	}
	err = outputRows.Close()
	if err != nil {
		return nil, nil, err
	}
	return inputs, outputs, nil
}

// rows must be like: txid, typevalue, typevote, Duration, hashlink, Signature
// функция не делает RollBack при ошибке
func scanTxs(txRows *sql.Rows, dbTx *sql.Tx) ([]TransAndHash, error) {
	txs := make([]TransAndHash, 0)
	for txRows.Next() {
		var typeValue, hashLink, hash, signature []byte
		var txAndHash TransAndHash
		txAndHash.Transaction = new(Transaction)
		err := txRows.Scan(
			&hash,
			&typeValue,
			&txAndHash.Transaction.TypeVote,
			&txAndHash.Transaction.Duration,
			&hashLink,
			&signature,
		)
		if err != nil {
			return nil, err
		}
		txAndHash.Transaction.TypeValue = sliceToHash(typeValue)
		txAndHash.Transaction.HashLink = sliceToHash(hashLink)
		copy(txAndHash.Hash[:], hash)
		copy(txAndHash.Transaction.Signature[:], signature)
		txs = append(txs, txAndHash)
	}
	err := txRows.Close()
	if err != nil {
		return nil, err
	}
	for _, txAndHash := range txs {
		tx := txAndHash.Transaction
		tx.Inputs, tx.Outputs, err = getTxInputsAndOutputs(dbTx, txAndHash.Hash)
		if err != nil {
			return nil, err
		}
		tx.InputSize = uint32(len(tx.Inputs))
		tx.OutputSize = uint32(len(tx.Outputs))
	}
	return txs, nil
}

type Database struct {
	db *sql.DB
}

func (d *Database) Init(dbname, user, password, host string, port int) error {
	connStr := fmt.Sprintf("dbname=%s user=%s password=%s host=%s port=%v sslmode=disable", dbname, user, password, host, port)
	var err error
	d.db, err = sql.Open("postgres", connStr)
	return err
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) SaveNextBlock(block *BlocAndkHash) error {
	dbTx, err := d.db.Begin()
	if err != nil {
		return err
	}
	_, err = dbTx.Exec(
		`INSERT INTO block (blockHash, PrevBlockHash, MerkleTree, Timestamp) VALUES ($1, $2, $3, $4)`,
		block.Hash[:],
		hashToSlice(block.B.PrevBlockHash),
		block.B.MerkleTree[:],
		block.B.Timestamp,
	)
	if err != nil {
		_ = dbTx.Rollback()
		return err
	}
	for i, txAndHash := range block.B.Trans {
		txId := txAndHash.Hash
		tx := txAndHash.Transaction
		_, err = dbTx.Exec(
			`INSERT INTO 
			Transaction(txid, Index, typevalue, typevote, Duration, hashlink, Signature, blockhash) 
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			txId[:],
			i,
			hashToSlice(tx.TypeValue),
			tx.TypeVote,
			tx.Duration,
			hashToSlice(tx.HashLink),
			tx.Signature[:],
			block.Hash[:],
		)
		if err != nil {
			_ = dbTx.Rollback()
			return err
		}
		for inputIndex, input := range tx.Inputs {
			_, err = dbTx.Exec(
				`INSERT INTO input(txid, Index, prevtxid, outputindex) VALUES ($1, $2, $3, $4)`,
				txId[:],
				inputIndex,
				input.PrevId[:],
				input.OutIndex,
			)
			if err != nil {
				_ = dbTx.Rollback()
				return err
			}
		}
		for outputIndex, output := range tx.Outputs {
			_, err = dbTx.Exec(
				`INSERT INTO output(txid, Index, Value, publickeyto) VALUES ($1, $2, $3, $4)`,
				txId[:],
				outputIndex,
				output.Value,
				output.PkeyTo[:],
			)
			if err != nil {
				_ = dbTx.Rollback()
				return err
			}
		}
	}
	err = dbTx.Commit()
	if err != nil {
		_ = dbTx.Rollback()
		return err
	}
	return err
}

// функция может не найти некоторые блоки (если их нет), но ошибки не будет
// так же эти блоке не появятся в возращаемом срезе
func (d *Database) GetBlocksByHashes(blockHashes [][HASH_SIZE]byte) ([]*BlocAndkHash, error) {
	dbTx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}
	blocksQuery := "SELECT block.blockhash, block.PrevBlockHash, block.merkletree, block.Timestamp " +
		"FROM block WHERE block.blockhash in (" + buildInLookup(1, len(blockHashes)+1) + ")"
	blockQueryArgs := make([]interface{}, len(blockHashes))
	for i := range blockHashes {
		blockQueryArgs[i] = blockHashes[i][:]
	}
	blockRows, err := dbTx.Query(blocksQuery, blockQueryArgs...)
	if err != nil {
		_ = dbTx.Rollback()
		return nil, err
	}
	blocks := make([]*BlocAndkHash, 0)
	for blockRows.Next() {
		blockAndHash := new(BlocAndkHash)
		b := new(Block)
		blockAndHash.B = b
		// создание срезов постгри не поддерживает массивы
		var hash, prevBlockHash, merkleTree []byte
		err := blockRows.Scan(&hash, &prevBlockHash, &merkleTree, &b.Timestamp)
		if err != nil {
			_ = dbTx.Rollback()
			return nil, err
		}
		copy(blockAndHash.Hash[:], hash)
		b.PrevBlockHash = sliceToHash(prevBlockHash)
		copy(b.MerkleTree[:], merkleTree)
		blocks = append(blocks, blockAndHash)
	}
	err = blockRows.Close()
	if err != nil {
		_ = dbTx.Rollback()
		return nil, err
	}

	for _, blockAndHash := range blocks {
		b := blockAndHash.B
		txRows, err := dbTx.Query(
			`SELECT txid, typevalue, typevote, Duration, hashlink, Signature 
			FROM Transaction WHERE Transaction.blockhash = $1 ORDER BY Transaction.Index`,
			blockAndHash.Hash[:],
		)
		if err != nil {
			_ = dbTx.Rollback()
			return nil, err
		}
		txs, err := scanTxs(txRows, dbTx)
		if err != nil {
			_ = dbTx.Rollback()
			return nil, err
		}
		b.Trans = txs
		b.TransSize = uint32(len(txs))
	}

	err = dbTx.Commit()
	if err != nil {
		_ = dbTx.Rollback()
		return nil, err
	}
	return blocks, nil
}

func (d *Database) GetTxByHash(hash [HASH_SIZE]byte) (*Transaction, error) {
	transAndHash, err := d.GetTxsByHashes([][HASH_SIZE]byte{hash})
	if err != nil {
		return nil, err
	}
	if len(transAndHash) < 1 {
		return nil, nil
	} else if len(transAndHash) == 1 {
		return transAndHash[0].Transaction, nil
	} else {
		panic("got too much transactions")
	}
}

// не все транзы из перечисленных в txHashes могут быть в ответе
// (если таких транз нет в бд)
func (d *Database) GetTxsByHashes(txHashes [][HASH_SIZE]byte) ([]TransAndHash, error) {
	dbTx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}
	txQuery := "SELECT txid, typevalue, typevote, Duration, HashLink, Signature " +
		"FROM Transaction WHERE Transaction.txid in (" + buildInLookup(1, len(txHashes)+1) + ")"
	txQueryArgs := make([]interface{}, len(txHashes))
	for i := range txHashes {
		txQueryArgs[i] = txHashes[i][:]
	}
	txRows, err := dbTx.Query(txQuery, txQueryArgs...)
	if err != nil {
		_ = dbTx.Rollback()
		return nil, err
	}
	txs, err := scanTxs(txRows, dbTx)
	if err != nil {
		_ = dbTx.Rollback()
		return nil, err
	}
	err = dbTx.Commit()
	if err != nil {
		_ = dbTx.Rollback()
		return nil, err
	}
	return txs, nil
}

func (d *Database) GetTxAndTimeByHash(hash [HASH_SIZE]byte) (*TransAndHash, uint64, error) {
	dbTx, err := d.db.Begin()
	if err != nil {
		return nil, 0, err
	}
	txRow, err := dbTx.Query(
		`
		SELECT block.timestamp, transaction.txid, transaction.typevalue, transaction.typevote, 
		transaction.Duration, transaction.HashLink, transaction.Signature 
		FROM block, transaction WHERE Transaction.txid = $1 and block.blockHash = transaction.blockHash`,
		hash[:],
	)

	if err != nil {
		_ = dbTx.Rollback()
		return nil, 0, err
	}

	if txRow.Next() {
		var typeValue, hashLink, hash, signature []byte
		var txAndHash TransAndHash
		var timestamp uint64
		txAndHash.Transaction = new(Transaction)
		err := txRow.Scan(
			&timestamp,
			&hash,
			&typeValue,
			&txAndHash.Transaction.TypeVote,
			&txAndHash.Transaction.Duration,
			&hashLink,
			&signature,
		)
		if err != nil {
			_ = dbTx.Rollback()
			return nil, 0, err
		}
		txAndHash.Transaction.TypeValue = sliceToHash(typeValue)
		txAndHash.Transaction.HashLink = sliceToHash(hashLink)
		copy(txAndHash.Hash[:], hash)
		copy(txAndHash.Transaction.Signature[:], signature)
		err = txRow.Close()
		if err != nil {
			_ = dbTx.Rollback()
			return nil, 0, err
		}
		tx := txAndHash.Transaction
		tx.Inputs, tx.Outputs, err = getTxInputsAndOutputs(dbTx, txAndHash.Hash)
		if err != nil {
			_ = dbTx.Rollback()
			return nil, 0, err
		}
		tx.InputSize = uint32(len(tx.Inputs))
		tx.OutputSize = uint32(len(tx.Outputs))
		err = dbTx.Commit()
		if err != nil {
			_ = dbTx.Rollback()
			return nil, 0, err
		}
		return &txAndHash, timestamp, nil
	} else {
		_ = dbTx.Rollback()
		return nil, 0, err
	}
}

func (d *Database) GetTxByHashLink(hashLink [HASH_SIZE]byte) (*TransAndHash, error) {
	dbTx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}
	txRow, err := dbTx.Query(
		`
		SELECT transaction.txid, transaction.typevalue, transaction.typevote, 
		transaction.Duration, transaction.HashLink, transaction.Signature 
		FROM transaction WHERE Transaction.HashLink = $1`,
		hashLink[:],
	)

	if err != nil {
		_ = dbTx.Rollback()
		return nil, err
	}

	if txRow.Next() {
		var typeValue, hashLink, hash, signature []byte
		var txAndHash TransAndHash
		var timestamp uint64
		txAndHash.Transaction = new(Transaction)
		err := txRow.Scan(
			&timestamp,
			&hash,
			&typeValue,
			&txAndHash.Transaction.TypeVote,
			&txAndHash.Transaction.Duration,
			&hashLink,
			&signature,
		)
		if err != nil {
			_ = dbTx.Rollback()
			return nil, err
		}
		txAndHash.Transaction.TypeValue = sliceToHash(typeValue)
		txAndHash.Transaction.HashLink = sliceToHash(hashLink)
		copy(txAndHash.Hash[:], hash)
		copy(txAndHash.Transaction.Signature[:], signature)
		err = txRow.Close()
		if err != nil {
			_ = dbTx.Rollback()
			return nil, err
		}
		tx := txAndHash.Transaction
		tx.Inputs, tx.Outputs, err = getTxInputsAndOutputs(dbTx, txAndHash.Hash)
		if err != nil {
			_ = dbTx.Rollback()
			return nil, err
		}
		tx.InputSize = uint32(len(tx.Inputs))
		tx.OutputSize = uint32(len(tx.Outputs))
		err = dbTx.Commit()
		if err != nil {
			_ = dbTx.Rollback()
			return nil, err
		}
		return &txAndHash, nil
	} else {
		_ = dbTx.Rollback()
		return nil, err
	}
}

func (d *Database) GetTxsByPubKey(pkey [PKEY_SIZE]byte) ([]TransAndHash, error) {
	dbTx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}
	txRows, err := dbTx.Query(
		`SELECT txid, typevalue, typevote, Duration, HashLink, Signature 
		FROM Transaction 
		WHERE EXISTS(
	   		SELECT * FROM output 
	   		WHERE output.txid = Transaction.txid and output.publickeyto = $1
		) 
		union 
		SELECT txid, typevalue, typevote, Duration, HashLink, Signature 
		FROM Transaction 
		WHERE Transaction.txid IN (
	   		SELECT isspentbytx from output 
	   		WHERE output.publickeyto = $1 
		)`,
		pkey[:],
	)
	if err != nil {
		_ = dbTx.Rollback()
		return nil, err
	}
	txs, err := scanTxs(txRows, dbTx)
	if err != nil {
		_ = dbTx.Rollback()
		return nil, err
	}
	err = dbTx.Commit()
	if err != nil {
		_ = dbTx.Rollback()
		return nil, err
	}
	return txs, nil
}

func (d *Database) getUTXOS(sqlQuery string, params []interface{}) ([]*UTXO, error) {
	dbTx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}
	utxosRows, err := d.db.Query(sqlQuery, params...)
	if err != nil {
		_ = dbTx.Rollback()
		return nil, err
	}
	utxos := make([]*UTXO, 0)
	for utxosRows.Next() {
		var typeValue, txid, pkeyTo []byte
		var timestamp uint64
		utxo := new(UTXO)
		err := utxosRows.Scan(&timestamp, &typeValue, &txid, &utxo.Index, &utxo.Value, &pkeyTo)
		if err != nil {
			_ = dbTx.Rollback()
			return nil, err
		}
		utxo.TypeValue = sliceToHash(typeValue)
		utxo.Timestamp = timestamp
		copy(utxo.TxId[:], txid)
		copy(utxo.PkeyTo[:], pkeyTo)
		utxos = append(utxos, utxo)
	}
	err = utxosRows.Close()
	if err != nil {
		_ = dbTx.Rollback()
		return nil, err
	}
	err = dbTx.Commit()
	if err != nil {
		_ = dbTx.Rollback()
		return nil, err
	}
	return utxos, nil
}

func (d *Database) GetUTXOSByPkey(pkey [PKEY_SIZE]byte) ([]*UTXO, error) {
	// условие transaction.typeVote = 0 нужно, чтобы не выбрать typeValue транзы создания голосования, который всегда нулевой
	// typeValue выходов транзы создания голосования - её хеш, для этого нужен второй селект после union
	return d.getUTXOS(
		`SELECT block.timestamp, transaction.typeValue, output.txid, output.Index, output.Value, output.publicKeyTo 
			from block,transaction, output  
			WHERE transaction.typeVote = 0 
				and transaction.txid = output.txid and block.blockHash = transaction.blockHash 
				and output.publickeyto = $1 and output.isspentbytx is null
			UNION
			SELECT block.timestamp, transaction.txid, output.txid, output.Index, output.Value, output.publicKeyTo
			from block, transaction, output
			WHERE transaction.typeVote != 0 
				and transaction.txid = output.txid and block.blockHash = transaction.blockHash
				and output.publickeyto = $1 and output.isspentbytx is null`,
		[]interface{}{pkey[:]},
	)
}

func (d *Database) GetUTXOSByTxId(txid [HASH_SIZE]byte) ([]*UTXO, error) {
	return d.getUTXOS(
		`SELECT block.timestamp, transaction.typeValue, output.txid, output.Index, output.Value, output.publicKeyTo 
			from block, transaction, output  
			WHERE transaction.typeVote = 0 
				and transaction.txid = output.txid and block.blockHash = transaction.blockHash 
				and output.txid = $1 and output.isspentbytx is null
			UNION
			SELECT block.timestamp, transaction.txid, output.txid, output.Index, output.Value, output.publicKeyTo
			from block, transaction, output
			WHERE transaction.typeVote != 0 
				and transaction.txid = output.txid and block.blockHash = transaction.blockHash
				and output.txid = $1 and output.isspentbytx is null`,
		[]interface{}{txid[:]},
	)
}

// если следующего блока нет, ошибки не будет, вернется nil, nil
func (d *Database) GetBlockAfter(blockHash [HASH_SIZE]byte) (*BlocAndkHash, error) {
	dbTx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}
	var blockRow *sql.Rows
	if blockHash != ZERO_ARRAY_HASH {
		blockRow, err = dbTx.Query(
			`SELECT block.blockhash, block.PrevBlockHash, block.merkletree, block.Timestamp 
			FROM block WHERE block.PrevBlockHash = $1`,
			blockHash[:],
		)
	} else {
		// получение первого блока
		blockRow, err = dbTx.Query(
			`SELECT block.blockhash, block.PrevBlockHash, block.merkletree, block.Timestamp 
			FROM block WHERE block.PrevBlockHash IS NULL`,
		)
	}
	if err != nil {
		_ = dbTx.Rollback()
		return nil, err
	}
	blockAndHash := new(BlocAndkHash)
	b := new(Block)
	blockAndHash.B = b
	if blockRow.Next() {
		var hash, prevBlockHash, merkleTree []byte
		err := blockRow.Scan(&hash, &prevBlockHash, &merkleTree, &b.Timestamp)
		if err != nil {
			_ = dbTx.Rollback()
			return nil, err
		}
		copy(blockAndHash.Hash[:], hash)
		b.PrevBlockHash = sliceToHash(prevBlockHash)
		copy(b.MerkleTree[:], merkleTree)
	} else {
		_ = dbTx.Commit()
		return nil, nil
	}
	err = blockRow.Close()
	if err != nil {
		_ = dbTx.Rollback()
		return nil, err
	}
	txRows, err := dbTx.Query(
		`SELECT txid, typevalue, typevote, Duration, hashlink, Signature 
			FROM Transaction WHERE Transaction.blockhash = $1 ORDER BY Transaction.Index`,
		blockAndHash.Hash[:],
	)
	if err != nil {
		_ = dbTx.Rollback()
		return nil, err
	}
	txs, err := scanTxs(txRows, dbTx)
	if err != nil {
		_ = dbTx.Rollback()
		return nil, err
	}
	err = txRows.Close()
	if err != nil {
		_ = dbTx.Rollback()
		return nil, err
	}
	b.Trans = txs
	b.TransSize = uint32(len(txs))
	err = dbTx.Commit()
	if err != nil {
		_ = dbTx.Rollback()
		return nil, err
	}
	return blockAndHash, nil
}
