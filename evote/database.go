package evote

import (
	"GO_LOSOVANIE/evote/golosovaniepb"
	"database/sql"
	"fmt"
	"github.com/golang/protobuf/proto"
	_ "github.com/lib/pq"
	"strconv"
	"strings"
)

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
func getTxInputsAndOutputs(
	dbTx *sql.Tx,
	txId int,
) (
	[]*golosovaniepb.Input,
	[]*golosovaniepb.Output,
	error,
) {
	var inputs []*golosovaniepb.Input
	var outputs []*golosovaniepb.Output
	inputRows, err := dbTx.Query(
		`SELECT txs.txHash, ins.outputIndex 
		FROM (
		    SELECT input.prevTxId, input.outputIndex, input.Index FROM input WHERE input.txId = $1
		    ) as ins 
		JOIN transaction as txs ON ins.prevTxId = txs.txId 
		ORDER BY ins.Index`,
		txId,
	)
	if err != nil {
		return nil, nil, err
	}
	for inputRows.Next() {
		var input golosovaniepb.Input
		var prevHash []byte
		err := inputRows.Scan(&prevHash, &input.OutputIndex)
		if err != nil {
			return nil, nil, err
		}
		input.PrevTxHash = prevHash
		inputs = append(inputs, &input)
	}
	err = inputRows.Close()
	if err != nil {
		return nil, nil, err
	}
	outputRows, err := dbTx.Query(
		`SELECT output.value, output.receiverSpendPkey, output.receiverScanPkey
				FROM output WHERE output.txId = $1 
				ORDER BY output.Index`,
		txId,
	)
	if err != nil {
		return nil, nil, err
	}
	for outputRows.Next() {
		var output golosovaniepb.Output
		var receiverSpendPkey, receiverScanPkey []byte
		err := outputRows.Scan(&output.Value, &receiverSpendPkey, &receiverScanPkey)
		if err != nil {
			return nil, nil, err
		}
		output.ReceiverSpendPkey = receiverSpendPkey
		output.ReceiverScanPkey = receiverScanPkey
		outputs = append(outputs, &output)
	}
	err = outputRows.Close()
	if err != nil {
		return nil, nil, err
	}
	return inputs, outputs, nil
}

// rows MUST be with columns: txId, txHash, hashLink, valueType, voteType, duration,  senderEphemeralPkey, votersSumPkey, signature
// функция не делает RollBack при ошибке
func scanTxs(txRows *sql.Rows, dbTx *sql.Tx) ([]*golosovaniepb.Transaction, error) {
	txIds := make([]int, 0)
	txs := make([]*golosovaniepb.TxBody, 0)
	hashes := make([][]byte, 0)
	sigs := make([][]byte, 0)
	for txRows.Next() {
		var txHash, hashLink, valueType, senderEphemeralPkey, votersSumPkey, signature []byte
		var txBody golosovaniepb.TxBody
		var txId int
		err := txRows.Scan(
			&txId,
			&txHash,
			&hashLink,
			&valueType,
			&txBody.VoteType,
			&txBody.Duration,
			&senderEphemeralPkey,
			&votersSumPkey,
			&signature,
		)
		if err != nil {
			return nil, err
		}
		txBody.HashLink = hashLink
		txBody.ValueType = valueType
		txBody.SenderEphemeralPkey = senderEphemeralPkey
		txBody.VotersSumPkey = votersSumPkey
		txIds = append(txIds, txId)
		txs = append(txs, &txBody)
		hashes = append(hashes, txHash)
		sigs = append(sigs, signature)
	}
	err := txRows.Close()
	if err != nil {
		return nil, err
	}
	txsFull := make([]*golosovaniepb.Transaction, len(txs))
	for i, tx := range txs {
		tx.Inputs, tx.Outputs, err = getTxInputsAndOutputs(dbTx, txIds[i])
		if err != nil {
			return nil, err
		}
		bodyBytes, err := proto.Marshal(tx)
		if err != nil {
			return nil, err
		}
		txsFull[i] = &golosovaniepb.Transaction{
			TxBody: bodyBytes,
			Hash:   hashes[i],
			Sig:    sigs[i],
		}
	}
	return txsFull, nil
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

func (d *Database) SaveNextBlock(block *golosovaniepb.Block) error {
	dbTx, err := d.db.Begin()
	if err != nil {
		return err
	}
	var blockId int
	if len(block.BlockHeader.PrevBlockHash) == 0 {
		// first block
		err = dbTx.QueryRow(
			`INSERT INTO block (height, blockHash, MerkleTree, proposerPkey, Timestamp) 
				VALUES (0, $1, $2, $3, $4) RETURNING blockId`,
			block.Hash,
			block.BlockHeader.MerkleTree,
			block.BlockHeader.ProposerPkey,
			block.BlockHeader.Timestamp,
		).Scan(&blockId)
	} else {
		err = dbTx.QueryRow(
			`INSERT INTO block (height, blockHash, PrevBlockId, MerkleTree, proposerPkey, Timestamp) 
				(SELECT block.height + 1, $1, block.blockId, $3, $4, $5 
				FROM block WHERE block.blockHash = $2) 
				RETURNING blockId`,
			block.Hash,
			block.BlockHeader.PrevBlockHash,
			block.BlockHeader.MerkleTree,
			block.BlockHeader.ProposerPkey,
			block.BlockHeader.Timestamp,
		).Scan(&blockId)
	}
	if err != nil {
		_ = dbTx.Rollback()
		return err
	}
	for i, tx := range block.Transactions {
		var txBody golosovaniepb.TxBody
		err = proto.Unmarshal(tx.TxBody, &txBody)
		if err != nil {
			_ = dbTx.Rollback()
			return err
		}
		var txId int
		err = dbTx.QueryRow(
			`INSERT INTO 
			Transaction (blockId, index, txHash, hashLink, valueType, voteType, duration, senderEphemeralPkey, votersSumPkey, signature) 
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			RETURNING txId`,
			blockId,
			i,
			tx.Hash,
			txBody.HashLink,
			txBody.ValueType,
			txBody.VoteType,
			txBody.Duration,
			txBody.SenderEphemeralPkey,
			txBody.VotersSumPkey,
			tx.Sig,
		).Scan(&txId)
		if err != nil {
			_ = dbTx.Rollback()
			return err
		}
		for inputIndex, input := range txBody.Inputs {
			_, err = dbTx.Exec(
				`INSERT INTO input(txId, index, prevTxId, outputIndex) 
						SELECT $1, $2, transaction.txId, $4 FROM transaction WHERE transaction.txHash = $3`,
				txId,
				inputIndex,
				input.PrevTxHash,
				input.OutputIndex,
			)
			if err != nil {
				_ = dbTx.Rollback()
				return err
			}
		}
		for outputIndex, output := range txBody.Outputs {
			_, err = dbTx.Exec(
				`INSERT INTO output(txId, index, value, receiverSpendPkey, receiverScanPkey) VALUES ($1, $2, $3, $4, $5)`,
				txId,
				outputIndex,
				output.Value,
				output.ReceiverSpendPkey,
				output.ReceiverScanPkey,
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

// GetBlocksByHashes функция может не найти некоторые блоки (если их нет), но ошибки не будет
// так же эти блоке не появятся в возращаемом срезе
func (d *Database) GetBlocksByHashes(blockHashes [][]byte) ([]*golosovaniepb.Block, error) {
	dbTx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}
	blocksQuery := "SELECT b.blockId, b.blockHash, pb.blockHash, b.merkleTree, b.proposerPkey, b.Timestamp " +
		"FROM block as b LEFT JOIN block as pb ON pb.blockId = b.prevBlockId WHERE b.blockHash in (" + buildInLookup(1, len(blockHashes)+1) + ")"
	blockQueryArgs := make([]interface{}, len(blockHashes))
	for i := range blockHashes {
		blockQueryArgs[i] = blockHashes[i]
	}
	blockRows, err := dbTx.Query(blocksQuery, blockQueryArgs...)
	if err != nil {
		_ = dbTx.Rollback()
		return nil, err
	}
	blocks := make([]*golosovaniepb.Block, 0)
	blockIds := make([]int, 0)
	for blockRows.Next() {
		var header golosovaniepb.BlockHeader
		var block golosovaniepb.Block
		var blockId int
		err := blockRows.Scan(
			&blockId,
			&block.Hash,
			&header.PrevBlockHash,
			&header.MerkleTree,
			&header.ProposerPkey,
			&header.Timestamp,
		)
		if err != nil {
			_ = dbTx.Rollback()
			return nil, err
		}
		block.BlockHeader = &header
		blocks = append(blocks, &block)
		blockIds = append(blockIds, blockId)
	}
	err = blockRows.Close()
	if err != nil {
		_ = dbTx.Rollback()
		return nil, err
	}

	for i, b := range blocks {
		txRows, err := dbTx.Query(
			`SELECT txId, txHash, hashLink, valueType, voteType, duration,  senderEphemeralPkey, votersSumPkey, signature 
			FROM Transaction WHERE Transaction.blockId = $1 ORDER BY Transaction.Index`,
			blockIds[i],
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
		b.Transactions = txs
	}

	err = dbTx.Commit()
	if err != nil {
		_ = dbTx.Rollback()
		return nil, err
	}
	return blocks, nil
}

func (d *Database) GetBlockByHash(hash []byte) (*golosovaniepb.Block, error) {
	blocks, err := d.GetBlocksByHashes([][]byte{hash})
	if err != nil {
		return nil, err
	}
	if len(blocks) < 1 {
		return nil, nil
	} else if len(blocks) == 1 {
		return blocks[0], nil
	} else {
		panic("got too much blocks")
	}
}

func (d *Database) GetTxByHash(hash []byte) (*golosovaniepb.Transaction, error) {
	txs, err := d.GetTxsByHashes([][]byte{hash})
	if err != nil {
		return nil, err
	}
	if len(txs) < 1 {
		return nil, nil
	} else if len(txs) == 1 {
		return txs[0], nil
	} else {
		panic("got too much transactions")
	}
}

// GetTxsByHashes не все транзы из перечисленных в txHashes могут быть в ответе
// (если таких транз нет в бд)
func (d *Database) GetTxsByHashes(txHashes [][]byte) ([]*golosovaniepb.Transaction, error) {
	dbTx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}
	txQuery := "SELECT txId, txHash, hashLink, valueType, voteType, duration,  senderEphemeralPkey, votersSumPkey, signature " +
		"FROM Transaction WHERE Transaction.txHash in (" + buildInLookup(1, len(txHashes)+1) + ")"
	txQueryArgs := make([]interface{}, len(txHashes))
	for i := range txHashes {
		txQueryArgs[i] = txHashes[i]
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

func (d *Database) GetTxAndTimeByHash(hash []byte) (*golosovaniepb.Transaction, uint64, error) {
	dbTx, err := d.db.Begin()
	if err != nil {
		return nil, 0, err
	}
	// txId, txHash, hashLink, valueType, voteType, duration,  senderEphemeralPkey, votersSumPkey, signature
	txRow, err := dbTx.Query(
		`
		SELECT block.timestamp, transaction.txid, transaction.txHash, transaction.hashLink, transaction.valueType,
		       transaction.voteType, transaction.duration, transaction.senderEphemeralPkey,
		       transaction.votersSumPkey, transaction.signature
		FROM block, transaction WHERE Transaction.txHash = $1 and block.blockId = transaction.blockId`,
		hash,
	)

	if err != nil {
		_ = dbTx.Rollback()
		return nil, 0, err
	}

	if txRow.Next() {
		var timestamp uint64
		var txId int
		var txBody golosovaniepb.TxBody
		var tx golosovaniepb.Transaction
		err := txRow.Scan(
			&timestamp,
			&txId,
			&tx.Hash,
			&txBody.HashLink,
			&txBody.ValueType,
			&txBody.VoteType,
			&txBody.Duration,
			&txBody.SenderEphemeralPkey,
			&txBody.VotersSumPkey,
			&tx.Sig,
		)
		if err != nil {
			_ = dbTx.Rollback()
			return nil, 0, err
		}
		err = txRow.Close()
		if err != nil {
			_ = dbTx.Rollback()
			return nil, 0, err
		}
		txBody.Inputs, txBody.Outputs, err = getTxInputsAndOutputs(dbTx, txId)
		if err != nil {
			_ = dbTx.Rollback()
			return nil, 0, err
		}
		err = dbTx.Commit()
		if err != nil {
			_ = dbTx.Rollback()
			return nil, 0, err
		}
		bodyBytes, err := proto.Marshal(&txBody)
		if err != nil {
			_ = dbTx.Rollback()
			return nil, 0, err
		}
		tx.TxBody = bodyBytes
		return &tx, timestamp, nil
	} else {
		_ = dbTx.Rollback()
		return nil, 0, err
	}
}

func (d *Database) GetTxByHashLink(hashLink []byte) (*golosovaniepb.Transaction, error) {
	dbTx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}
	txRow, err := dbTx.Query(
		`
		SELECT transaction.txid, transaction.txHash, transaction.hashLink, transaction.valueType,
		       transaction.voteType, transaction.duration, transaction.senderEphemeralPkey,
		       transaction.votersSumPkey, transaction.signature
		FROM transaction WHERE transaction.hashLink = $1`,
		hashLink,
	)

	if err != nil {
		_ = dbTx.Rollback()
		return nil, err
	}

	if txRow.Next() {
		var txId int
		var txBody golosovaniepb.TxBody
		var tx golosovaniepb.Transaction
		err := txRow.Scan(
			&txId,
			&tx.Hash,
			&txBody.HashLink,
			&txBody.ValueType,
			&txBody.VoteType,
			&txBody.Duration,
			&txBody.SenderEphemeralPkey,
			&txBody.VotersSumPkey,
			&tx.Sig,
		)
		if err != nil {
			_ = dbTx.Rollback()
			return nil, err
		}
		err = txRow.Close()
		if err != nil {
			_ = dbTx.Rollback()
			return nil, err
		}
		txBody.Inputs, txBody.Outputs, err = getTxInputsAndOutputs(dbTx, txId)
		if err != nil {
			_ = dbTx.Rollback()
			return nil, err
		}
		err = dbTx.Commit()
		if err != nil {
			_ = dbTx.Rollback()
			return nil, err
		}
		bodyBytes, err := proto.Marshal(&txBody)
		if err != nil {
			_ = dbTx.Rollback()
			return nil, err
		}
		tx.TxBody = bodyBytes
		return &tx, nil
	} else {
		_ = dbTx.Rollback()
		return nil, err
	}
}

func (d *Database) GetTxsByPubKey(pkey []byte) ([]*golosovaniepb.Transaction, error) {
	dbTx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}
	// txId, txHash, hashLink, valueType, voteType, duration,  senderEphemeralPkey, votersSumPkey, signature
	txRows, err := dbTx.Query(
		`SELECT txId, txHash, hashLink, valueType, voteType, duration,  senderEphemeralPkey, votersSumPkey, signature 
		FROM Transaction 
		WHERE EXISTS(
	   		SELECT * FROM output 
	   		WHERE output.txid = transaction.txid and (output.receiverSpendPkey = $1 or output.receiverScanPkey = $1)
		) 
		union 
		SELECT txId, txHash, hashLink, valueType, voteType, duration,  senderEphemeralPkey, votersSumPkey, signature
		FROM Transaction 
		WHERE Transaction.txid IN (
	   		SELECT isspentbytx from output 
	   		WHERE (output.receiverSpendPkey = $1 or output.receiverScanPkey = $1)
		)`,
		pkey,
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

func (d *Database) getUTXOS(sqlQuery string, params []interface{}) ([]*golosovaniepb.Utxo, error) {
	dbTx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}
	utxosRows, err := d.db.Query(sqlQuery, params...)
	if err != nil {
		_ = dbTx.Rollback()
		return nil, err
	}
	utxos := make([]*golosovaniepb.Utxo, 0)
	for utxosRows.Next() {
		var utxo golosovaniepb.Utxo
		err := utxosRows.Scan(
			&utxo.Timestamp,
			&utxo.ValueType,
			&utxo.TxHash,
			&utxo.Index,
			&utxo.Value,
			&utxo.ReceiverSpendPkey,
			&utxo.ReceiverScanPkey,
		)
		if err != nil {
			_ = dbTx.Rollback()
			return nil, err
		}
		utxos = append(utxos, &utxo)
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

func (d *Database) GetUTXOSByPkey(pkey []byte) ([]*golosovaniepb.Utxo, error) {
	// условие transaction.voteType = 0 нужно, чтобы не выбрать valueType транзы создания голосования, который всегда нулевой
	// valueType выходов транзы создания голосования - её хеш, для этого нужен второй селект после union
	return d.getUTXOS(
		`SELECT block.timestamp, transaction.valueType, transaction.txHash, 
			output.Index, output.Value, output.receiverSpendPkey, output.receiverScanPkey 
			from block,transaction, output  
			WHERE transaction.voteType = 0 
				and transaction.txid = output.txid and block.blockId = transaction.blockId 
				and (output.receiverSpendPkey = $1 or output.receiverScanPkey = $1) and output.isspentbytx is null
			UNION
			SELECT block.timestamp, transaction.txHash, transaction.txHash, 
			output.Index, output.Value, output.receiverSpendPkey, output.receiverScanPkey
			from block, transaction, output
			WHERE transaction.voteType != 0 
				and transaction.txid = output.txid and block.blockId = transaction.blockId
				and (output.receiverSpendPkey = $1 or output.receiverScanPkey = $1) and output.isspentbytx is null`,
		[]interface{}{pkey},
	)
}

func (d *Database) GetUtxosByTxHash(txHash []byte) ([]*golosovaniepb.Utxo, error) {
	return d.getUTXOS(
		`SELECT block.timestamp, transaction.valueType, transaction.txHash, 
			output.Index, output.Value, output.receiverSpendPkey, output.receiverScanPkey 
			from block,transaction, output  
			WHERE transaction.voteType = 0 
				and transaction.txid = output.txid and block.blockId = transaction.blockId 
				and transaction.txHash = $1 and output.isspentbytx is null
			UNION
			SELECT block.timestamp, transaction.txHash, transaction.txHash, 
			output.Index, output.Value, output.receiverSpendPkey, output.receiverScanPkey
			from block, transaction, output
			WHERE transaction.voteType != 0 
				and transaction.txid = output.txid and block.blockId = transaction.blockId
				and transaction.txHash = $1 and output.isspentbytx is null`,
		[]interface{}{txHash},
	)
}

func (d *Database) GetUTXOSByTypeValue(typeValue []byte) ([]*golosovaniepb.Utxo, error) {
	return d.getUTXOS(
		`SELECT block.timestamp, transaction.valueType, transaction.txHash, 
			output.Index, output.Value, output.receiverSpendPkey, output.receiverScanPkey 
			from block,transaction, output  
			WHERE transaction.voteType = 0 
				and transaction.txid = output.txid and block.blockId = transaction.blockId 
				and transaction.valueType = $1 and output.isspentbytx is null`,
		[]interface{}{typeValue},
	)
}

// GetBlockAfter если следующего блока нет, ошибки не будет, вернется nil, nil
func (d *Database) GetBlockAfter(blockHash []byte) (*golosovaniepb.Block, error) {
	dbTx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}
	var blockRow *sql.Rows
	if len(blockHash) != 0 {
		blockRow, err = dbTx.Query(
			`SELECT b.blockId, b.blockhash, pb.blockHash, b.merkletree, b.proposerPkey, b.Timestamp 
			FROM block as b JOIN block as pb ON b.prevBlockId = pb.blockId WHERE pb.blockHash = $1`,
			blockHash,
		)
	} else {
		// получение первого блока
		blockRow, err = dbTx.Query(
			`SELECT block.blockId, block.blockhash, NULL, block.merkletree, block.proposerPkey, block.Timestamp 
			FROM block WHERE block.prevBlockId IS NULL`,
		)
	}
	if err != nil {
		_ = dbTx.Rollback()
		return nil, err
	}
	var header golosovaniepb.BlockHeader
	var block golosovaniepb.Block
	var blockId int
	if blockRow.Next() {
		err := blockRow.Scan(
			&blockId,
			&block.Hash,
			&header.PrevBlockHash,
			&header.MerkleTree,
			&header.ProposerPkey,
			&header.Timestamp,
		)
		if err != nil {
			_ = dbTx.Rollback()
			return nil, err
		}
		block.BlockHeader = &header
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
		`SELECT txId, txHash, hashLink, valueType, voteType, duration,  senderEphemeralPkey, votersSumPkey, signature 
			FROM Transaction WHERE Transaction.blockId = $1 ORDER BY Transaction.Index`,
		blockId,
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
	block.Transactions = txs
	err = dbTx.Commit()
	if err != nil {
		_ = dbTx.Rollback()
		return nil, err
	}
	return &block, nil
}
