package evote

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
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

func hashToSlice(v [HASH_SIZE]byte) []byte {
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

// не откатывает транзу при ошибке
func getTxInputsAndOutputs(dbTx *sql.Tx, txHash [HASH_SIZE]byte) ([]TransactionInput, []TransactionOutput, error) {
	inputs := make([]TransactionInput, 0)
	outputs := make([]TransactionOutput, 0)
	inputRows, err := dbTx.Query(
		`SELECT prevtxid, outputindex 
		FROM input WHERE txid = $1 ORDER BY index`,
		txHash,
	)
	if err != nil {
		return nil, nil, err
	}
	for inputRows.Next() {
		var input TransactionInput
		err := inputRows.Scan(&input.prevId, &input.outIndex)
		if err != nil {
			return nil, nil, err
		}
		inputs = append(inputs, input)
	}
	outputRows, err := dbTx.Query(
		`SELECT value, publickeyto 
		FROM output WHERE txid = %s ORDER BY index`,
		txHash,
	)
	if err != nil {
		return nil, nil, err
	}
	for outputRows.Next() {
		var output TransactionOutput
		err := outputRows.Scan(&output.value, &output.pkeyTo)
		if err != nil {
			return nil, nil, err
		}
		outputs = append(outputs, output)
	}
	return inputs, outputs, nil
}

type Database struct {
	db *sql.DB
}

func (d *Database) Init(dbname, user, password, host string, port int) error {
	connStr := fmt.Sprintf("dbname=%s user=%s password=%s host=%s port=%v", dbname, user, password, host, port)
	var err error
	d.db, err = sql.Open("postgres", connStr)
	return err
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) saveNextBlock(block *BlocAndkHash) error {
	dbTx, err := d.db.Begin()
	if err != nil {
		return err
	}
	_, err = dbTx.Exec(
		`INSERT INTO block (blockHash, prevBlockHash, merkleTree, timestamp) VALUES ($1, $2, $3, $4)`,
		block.hash,
		block.b.prevBlockHash,
		block.b.merkleTree,
		block.b.timestamp,
	)
	if err != nil {
		_ = dbTx.Rollback()
		return err
	}
	for i, txAndHash := range block.b.trans {
		txId := txAndHash.hash
		tx := txAndHash.transaction
		// TODO: handle null values
		_, err = dbTx.Exec(
			`INSERT INTO 
			transaction(txid, index, typevalue, typevote, duration, hashlink, signature, blockhash) 
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			txId,
			i,
			hashToSlice(tx.typeValue),
			tx.typeVote,
			tx.duration,
			hashToSlice(tx.hashLink),
			tx.signature,
			block.hash,
		)
		if err != nil {
			_ = dbTx.Rollback()
			return err
		}
		for inputIndex, input := range tx.inputs {
			_, err = dbTx.Exec(
				`INSERT INTO input(txid, index, prevtxid, outputindex) VALUES ($1, $2, $3, $4)`,
				txId,
				inputIndex,
				input.prevId,
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
				txId,
				outputIndex,
				output.value,
				output.pkeyTo,
			)
			if err != nil {
				_ = dbTx.Rollback()
				return err
			}
		}
	}
	err = dbTx.Commit()
	return err
}

// функция может не найти некоторые блоки (если их нет), но ошибки не будет
func (d *Database) getBlocksByHashes(blockHashes [][HASH_SIZE]byte) ([]*BlocAndkHash, error) {
	dbTx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}
	blocksQuery := "SELECT block.blockhash, block.prevBlockHash, block.merkletree, block.timestamp " +
		"FROM block WHERE block.blockhash in (?" + strings.Repeat(",?", len(blockHashes)-1) + ")"
	blockQueryArgs := make([]interface{}, 0)
	for _, h := range blockHashes {
		blockQueryArgs = append(blockQueryArgs, h)
	}
	rows, err := dbTx.Query(blocksQuery, blockQueryArgs...)
	if err != nil {
		_ = dbTx.Rollback()
		return nil, err
	}
	defer rows.Close()
	blocks := make([]*BlocAndkHash, 0)
	for rows.Next() {
		blockAndHash := new(BlocAndkHash)
		b := new(Block)
		blockAndHash.b = b
		blocks = append(blocks, blockAndHash)
		err := rows.Scan(&blockAndHash.hash, &b.prevBlockHash, &b.merkleTree, &b.timestamp)
		if err != nil {
			_ = dbTx.Rollback()
			return nil, err
		}
		txRows, err := dbTx.Query(
			`SELECT txid, typevalue, typevote, duration, hashlink, signature 
			FROM transaction WHERE transaction.blockhash = $1 ORDER BY transaction.index`,
			blockAndHash.hash,
		)
		if err != nil {
			_ = dbTx.Rollback()
			return nil, err
		}
		defer txRows.Close()
		for txRows.Next() {
			var typeValue, hashLink []byte
			var txAndHash TransAndHash
			txAndHash.transaction = new(Transaction)
			err := txRows.Scan(
				&txAndHash.hash,
				&typeValue,
				&txAndHash.transaction.typeVote,
				&txAndHash.transaction.duration,
				&hashLink,
				&txAndHash.transaction.signature,
			)
			if err != nil {
				_ = dbTx.Rollback()
				return nil, err
			}
			txAndHash.transaction.typeValue = sliceToHash(typeValue)
			txAndHash.transaction.hashLink = sliceToHash(hashLink)
			txAndHash.transaction.inputs, txAndHash.transaction.outputs, err = getTxInputsAndOutputs(dbTx, txAndHash.hash)
			if err != nil {
				_ = dbTx.Rollback()
				return nil, err
			}
			b.trans = append(b.trans, txAndHash)
		}
	}
	err = dbTx.Commit()
	if err != nil {
		return nil, err
	}
	return blocks, nil
}

func (d *Database) getTxsByHashes(txHashes [][HASH_SIZE]byte) ([]TransAndHash, error) {
	dbTx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}
	txQuery := "SELECT txid, typevalue, typevote, duration, hashLink, signature " +
		"FROM transaction WHERE transaction.txid in (?" + strings.Repeat(",?", len(txHashes)-1) + ")"
	txQueryArgs := make([]interface{}, 0)
	for _, h := range txHashes {
		txQueryArgs = append(txQueryArgs, h)
	}
	txRows, err := dbTx.Query(txQuery, txQueryArgs...)
	if err != nil {
		_ = dbTx.Rollback()
		return nil, err
	}
	defer txRows.Close()
	txs := make([]TransAndHash, 0)
	for txRows.Next() {
		var typeValue, hashLink []byte
		var txAndHash TransAndHash
		txAndHash.transaction = new(Transaction)
		err := txRows.Scan(
			&txAndHash.hash,
			&typeValue,
			&txAndHash.transaction.typeVote,
			&txAndHash.transaction.duration,
			&hashLink,
			&txAndHash.transaction.signature,
		)
		if err != nil {
			_ = dbTx.Rollback()
			return nil, err
		}
		txAndHash.transaction.typeValue = sliceToHash(typeValue)
		txAndHash.transaction.hashLink = sliceToHash(hashLink)
		txAndHash.transaction.inputs, txAndHash.transaction.outputs, err = getTxInputsAndOutputs(dbTx, txAndHash.hash)
		if err != nil {
			_ = dbTx.Rollback()
			return nil, err
		}
		txs = append(txs, txAndHash)
	}
	return txs, nil
}
