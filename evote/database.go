package evote

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
)

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
			tx.typeValue,
			tx.typeVote,
			tx.duration,
			tx.hashLink,
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
