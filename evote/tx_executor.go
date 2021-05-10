package evote

import (
	"fmt"
	"time"
)

type TxExecutor struct {
	Transactions   []TransAndHash
	Timestamp      time.Time
	BlockProposer  [PKEY_SIZE]byte
	db             *Database
	processedTrans map[[HASH_SIZE]byte]bool
}

func NewTxExecutor(db *Database) *TxExecutor {
	return &TxExecutor{db: db}
}

func (t *TxExecutor) Reset() {
	// TODO: reset database CheckTxState or DeliverTxState
	t.Transactions = nil
	t.BlockProposer = ZERO_ARRAY_PKEY
	t.processedTrans = make(map[[HASH_SIZE]byte]bool)
}

func (t *TxExecutor) BeginBlock(timestamp time.Time, blockProposer [PKEY_SIZE]byte) {
	t.Timestamp = timestamp
	t.BlockProposer = blockProposer
}

// AppendTx used in DeliverTx and CheckTx abci methods
// ignoreDuplicates=true tells to approve transactions, that have already been approved
// TODO: check duplicate handling rules for tendermint. Should i use flags in request from tendermint?
// TODO: implement double spending check inside the same block
func (t *TxExecutor) AppendTx(data []byte, ignoreDuplicates bool) (code uint32) {
	var tx Transaction
	var transSize = tx.FromBytes(data)
	txHash := Hash(data[:transSize])
	txAndHash := TransAndHash{}
	copy(txAndHash.Hash[:], txHash)
	txAndHash.Transaction = &tx

	if ignoreDuplicates {
		if t.processedTrans[txAndHash.Hash] {
			return CODE_OK
		}
	}

	if transSize == ERR_TRANS_SIZE {
		fmt.Println("err: tx not parsed")
		return CODE_PARSE_ERR
	}

	if tx.OutputSize == 0 {
		fmt.Println("err: no outputs")
		return CODE_NO_OUTPUTS
	}

	if tx.HashLink != ZERO_ARRAY_HASH && tx.TypeVote != 0 {
		// coinbase transactions have non-zero hashlink, pointing on a block, for which reward is being distributed
		fmt.Println("err: trans with non-zero HashLink has incorrect TypeValue/TypeVote fields")
		return CODE_HASH_LINK_AND_TYPE_VOTE_TOGETHER
	}

	if tx.HashLink != ZERO_ARRAY_HASH && tx.InputSize == 0 {
		// this is coinbase tx, need to check receiver and double spending
		if tx.OutputSize != 1 {
			return CODE_COINBASE_TX_NO_OUTPUT
		}

		pkey := tx.Outputs[0].PkeyTo
		rewardValue := tx.Outputs[0].Value
		rewardBlock := tx.HashLink
		duplicate, err := t.db.GetTxByHashLink(rewardBlock)
		if err != nil {
			panic(err)
		}
		if duplicate != nil {
			return CODE_DOUBLE_COINBASE_FOR_SAME_BLOCK
		}
		block, err := t.db.GetBlockByHash(rewardBlock)
		if err != nil {
			panic(err)
		}
		if block == nil {
			return CODE_COINBASE_NO_BLOCK
		}
		if block.B.proposerPkey != pkey {
			return CODE_COINBASE_PROPOSER_MISMATCH
		}
		if rewardValue != REWARD {
			return CODE_COINBASE_INCORRECT_REWARD
		}
		return t.verifySigAndAppend(data, transSize, txAndHash, pkey)
	}

	var inputsSum, outputsSum uint32
	var pkey [PKEY_SIZE]byte
	for i, input := range tx.Inputs {
		var correspondingUtxo *UTXO
		utxos, err := t.db.GetUTXOSByTxId(input.PrevId)
		if err != nil {
			panic(err)
		}
		// проверка, что вход - непотраченный выход дургой транзы
		for _, utxo := range utxos {
			if utxo.Index == input.OutIndex {
				correspondingUtxo = utxo
				if i == 0 {
					pkey = correspondingUtxo.PkeyTo
				} else {
					if pkey != correspondingUtxo.PkeyTo {
						fmt.Println("err: input not owned by sender")
					}
				}
				break
			}
		}
		if correspondingUtxo == nil {
			fmt.Println("err: double spending in tx")
			return CODE_DOBULE_SPENDING
		}
		inputsSum += correspondingUtxo.Value
		// проверка, что в одной транзе не смешиваются разные typeValue
		if tx.HashLink == ZERO_ARRAY_HASH && tx.TypeVote == 0 && correspondingUtxo.TypeValue != tx.TypeValue {
			fmt.Println("err: incorrect typeValue in input", input)
			return CODE_MIXING_TYPE_VALUE
		}

	}
	for _, output := range tx.Outputs {
		outputsSum += output.Value
	}
	if outputsSum != inputsSum {
		fmt.Printf("err: outputs sum %v is not matching than inputs sum %v\n", outputsSum, inputsSum)
		return CODE_INPUTS_NOT_MATCH_OUTPUTS
	}
	return t.verifySigAndAppend(data, transSize, txAndHash, pkey)
}

func (t *TxExecutor) verifySigAndAppend(
	data []byte, transSize int, txAndHash TransAndHash, pkey [PKEY_SIZE]byte,
) (code uint32) {
	if !VerifyData(data[:transSize-SIG_SIZE], txAndHash.Transaction.Signature[:], pkey) {
		fmt.Println("err: signature doesnt match")
		return CODE_INVALID_SIGNATURE
	}
	t.Transactions = append(t.Transactions, txAndHash)
	t.processedTrans[txAndHash.Hash] = true
	return CODE_OK
}
