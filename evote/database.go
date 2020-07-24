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

// Transaction.typeVote и Transaction.typeValue не преобразуются к nil, так как на них не завязана ссылочная целостность
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
		FROM input WHERE txid = $1 ORDER BY index`,
		txHash[:],
	)
	if err != nil {
		return nil, nil, err
	}
	for inputRows.Next() {
		var input TransactionInput
		var prevId []byte
		err := inputRows.Scan(&prevId, &input.outIndex)
		if err != nil {
			return nil, nil, err
		}
		copy(input.prevId[:], prevId)
		inputs = append(inputs, input)
	}
	err = inputRows.Close()
	if err != nil {
		return nil, nil, err
	}
	outputRows, err := dbTx.Query(
		`SELECT value, publickeyto 
		FROM output WHERE txid = $1 ORDER BY index`,
		txHash[:],
	)
	if err != nil {
		return nil, nil, err
	}
	for outputRows.Next() {
		var output TransactionOutput
		var pkey []byte
		err := outputRows.Scan(&output.value, &pkey)
		if err != nil {
			return nil, nil, err
		}
		copy(output.pkeyTo[:], pkey)
		outputs = append(outputs, output)
	}
	err = outputRows.Close()
	if err != nil {
		return nil, nil, err
	}
	return inputs, outputs, nil
}

// rows must be like: txid, typevalue, typevote, duration, hashlink, signature
// функция не делает RollBack при ошибке
func scanTxs(txRows *sql.Rows, dbTx *sql.Tx) ([]TransAndHash, error) {
	txs := make([]TransAndHash, 0)
	for txRows.Next() {
		var typeValue, hashLink, hash, signature []byte
		var txAndHash TransAndHash
		txAndHash.transaction = new(Transaction)
		err := txRows.Scan(
			&hash,
			&typeValue,
			&txAndHash.transaction.typeVote,
			&txAndHash.transaction.duration,
			&hashLink,
			&signature,
		)
		if err != nil {
			return nil, err
		}
		txAndHash.transaction.typeValue = sliceToHash(typeValue)
		txAndHash.transaction.hashLink = sliceToHash(hashLink)
		copy(txAndHash.hash[:], hash)
		copy(txAndHash.transaction.signature[:], signature)
		txs = append(txs, txAndHash)
	}
	err := txRows.Close()
	if err != nil {
		return nil, err
	}
	for _, txAndHash := range txs {
		tx := txAndHash.transaction
		tx.inputs, tx.outputs, err = getTxInputsAndOutputs(dbTx, txAndHash.hash)
		if err != nil {
			return nil, err
		}
		tx.inputSize = uint32(len(tx.inputs))
		tx.outputSize = uint32(len(tx.outputs))
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
		`INSERT INTO block (blockHash, prevBlockHash, merkleTree, timestamp) VALUES ($1, $2, $3, $4)`,
		block.hash[:],
		hashToSlice(block.b.prevBlockHash),
		block.b.merkleTree[:],
		block.b.timestamp,
	)
	if err != nil {
		_ = dbTx.Rollback()
		return err
	}
	for i, txAndHash := range block.b.trans {
		txId := txAndHash.hash
		tx := txAndHash.transaction
		_, err = dbTx.Exec(
			`INSERT INTO 
			transaction(txid, index, typevalue, typevote, duration, hashlink, signature, blockhash) 
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			txId[:],
			i,
			hashToSlice(tx.typeValue),
			tx.typeVote,
			tx.duration,
			hashToSlice(tx.hashLink),
			tx.signature[:],
			block.hash[:],
		)
		if err != nil {
			_ = dbTx.Rollback()
			return err
		}
		for inputIndex, input := range tx.inputs {
			_, err = dbTx.Exec(
				`INSERT INTO input(txid, index, prevtxid, outputindex) VALUES ($1, $2, $3, $4)`,
				txId[:],
				inputIndex,
				input.prevId[:],
				input.outIndex,
			)
			if err != nil {
				_ = dbTx.Rollback()
				return err
			}
		}
		for outputIndex, output := range tx.outputs {
			_, err = dbTx.Exec(
				`INSERT INTO output(txid, index, value, publickeyto) VALUES ($1, $2, $3, $4)`,
				txId[:],
				outputIndex,
				output.value,
				output.pkeyTo[:],
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
	blocksQuery := "SELECT block.blockhash, block.prevBlockHash, block.merkletree, block.timestamp " +
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
		blockAndHash.b = b
		// создание срезов постгри не поддерживает массивы
		var hash, prevBlockHash, merkleTree []byte
		err := blockRows.Scan(&hash, &prevBlockHash, &merkleTree, &b.timestamp)
		if err != nil {
			_ = dbTx.Rollback()
			return nil, err
		}
		copy(blockAndHash.hash[:], hash)
		b.prevBlockHash = sliceToHash(prevBlockHash)
		copy(b.merkleTree[:], merkleTree)
		blocks = append(blocks, blockAndHash)
	}
	err = blockRows.Close()
	if err != nil {
		_ = dbTx.Rollback()
		return nil, err
	}

	for _, blockAndHash := range blocks {
		b := blockAndHash.b
		txRows, err := dbTx.Query(
			`SELECT txid, typevalue, typevote, duration, hashlink, signature 
			FROM transaction WHERE transaction.blockhash = $1 ORDER BY transaction.index`,
			blockAndHash.hash[:],
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
		b.trans = txs
		b.transSize = uint32(len(txs))
	}

	err = dbTx.Commit()
	if err != nil {
		_ = dbTx.Rollback()
		return nil, err
	}
	return blocks, nil
}

// не все транзы из перечисленных в txHashes могут быть в ответе
// (если таких транз нет в бд)
func (d *Database) GetTxsByHashes(txHashes [][HASH_SIZE]byte) ([]TransAndHash, error) {
	dbTx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}
	txQuery := "SELECT txid, typevalue, typevote, duration, hashLink, signature " +
		"FROM transaction WHERE transaction.txid in (" + buildInLookup(1, len(txHashes)+1) + ")"
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

func (d *Database) GetTxsByPubKey(pkey [PKEY_SIZE]byte) ([]TransAndHash, error) {
	dbTx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}
	txRows, err := dbTx.Query(
		`SELECT txid, typevalue, typevote, duration, hashLink, signature 
		FROM transaction 
		WHERE EXISTS(
	   		SELECT * FROM output 
	   		WHERE output.txid = transaction.txid and output.publickeyto = $1
		) 
		union 
		SELECT txid, typevalue, typevote, duration, hashLink, signature 
		FROM transaction 
		WHERE transaction.txid IN (
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
		var txid, pkeyTo []byte
		utxo := new(UTXO)
		err := utxosRows.Scan(&txid, &utxo.index, &utxo.value, &pkeyTo)
		if err != nil {
			_ = dbTx.Rollback()
			return nil, err
		}
		copy(utxo.txId[:], txid)
		copy(utxo.pkeyTo[:], pkeyTo)
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
	return d.getUTXOS(
		`SELECT txid, index, value, publicKeyTo from output 
			WHERE publickeyto = $1 and isspentbytx is null`,
		[]interface{}{pkey[:]},
	)
}

func (d *Database) GetUTXOSByTxId(txid [HASH_SIZE]byte) ([]*UTXO, error) {
	return d.getUTXOS(
		`SELECT txid, index, value, publicKeyTo from output 
			WHERE txid = $1 and isspentbytx is null`,
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
			`SELECT block.blockhash, block.prevBlockHash, block.merkletree, block.timestamp 
			FROM block WHERE block.prevBlockHash = $1`,
			blockHash[:],
		)
	} else {
		// получение первого блока
		blockRow, err = dbTx.Query(
			`SELECT block.blockhash, block.prevBlockHash, block.merkletree, block.timestamp 
			FROM block WHERE block.prevBlockHash IS NULL`,
		)
	}
	if err != nil {
		_ = dbTx.Rollback()
		return nil, err
	}
	blockAndHash := new(BlocAndkHash)
	b := new(Block)
	blockAndHash.b = b
	if blockRow.Next() {
		var hash, prevBlockHash, merkleTree []byte
		err := blockRow.Scan(&hash, &prevBlockHash, &merkleTree, &b.timestamp)
		if err != nil {
			_ = dbTx.Rollback()
			return nil, err
		}
		copy(blockAndHash.hash[:], hash)
		b.prevBlockHash = sliceToHash(prevBlockHash)
		copy(b.merkleTree[:], merkleTree)
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
		`SELECT txid, typevalue, typevote, duration, hashlink, signature 
			FROM transaction WHERE transaction.blockhash = $1 ORDER BY transaction.index`,
		blockAndHash.hash[:],
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
	b.trans = txs
	b.transSize = uint32(len(txs))
	err = dbTx.Commit()
	if err != nil {
		_ = dbTx.Rollback()
		return nil, err
	}
	return blockAndHash, nil
}
