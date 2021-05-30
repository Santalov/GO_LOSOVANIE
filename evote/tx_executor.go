package evote

import (
	"GO_LOSOVANIE/evote/golosovaniepb"
	"bytes"
	"fmt"
	"github.com/golang/protobuf/proto"
	"time"
)

type TxExecutor struct {
	Transactions   []*golosovaniepb.Transaction
	Timestamp      time.Time
	BlockProposer  [PkeySize]byte
	db             *Database
	processedTrans map[[HashSize]byte]bool
}

func NewTxExecutor(db *Database) *TxExecutor {
	return &TxExecutor{db: db}
}

func (t *TxExecutor) Reset() {
	// TODO: reset database CheckTxState or DeliverTxState
	t.Transactions = nil
	t.BlockProposer = ZeroArrayPkey
	t.processedTrans = make(map[[HashSize]byte]bool)
}

func (t *TxExecutor) BeginBlock(timestamp time.Time, blockProposer [PkeySize]byte) {
	t.Timestamp = timestamp
	t.BlockProposer = blockProposer
}

// AppendTx used in DeliverTx and CheckTx abci methods
// ignoreDuplicates=true tells to approve transactions, that have already been approved
// TODO: check duplicate handling rules for tendermint. Should i use flags in request from tendermint?
// TODO: implement double spending check inside the same block
func (t *TxExecutor) AppendTx(data []byte, ignoreDuplicates bool) (code uint32) {
	var tx golosovaniepb.Transaction
	err := proto.Unmarshal(data, &tx)
	if err != nil {
		fmt.Println("parse tx err: ", err)
		return CodeParseErr
	}
	if len(tx.TxBody) > MaxTxSize {
		fmt.Println("err: tx too large")
		return CodeTxTooLarge
	}
	var body golosovaniepb.TxBody
	err = proto.Unmarshal(tx.TxBody, &body)
	if err != nil {
		fmt.Println("parse txBody err: ", err)
		return CodeParseErr
	}

	if !bytes.Equal(tx.Hash, Hash(tx.TxBody)) {
		fmt.Println("hashes not equal")
		return CodeHashesDontMatch
	}

	hashBytes := SliceToHash(tx.Hash)
	if ignoreDuplicates {
		if t.processedTrans[hashBytes] {
			return CodeOk
		}
	}

	if len(body.Outputs) == 0 {
		fmt.Println("err: no outputs")
		return CodeNoOutputs
	}

	if len(body.HashLink) != 0 && body.VoteType != 0 {
		// coinbase transactions have non-zero hashlink, pointing on a block, for which reward is being distributed
		fmt.Println("err: trans with non-zero HashLink has incorrect TypeValue/TypeVote fields")
		return CodeHashLinkAndTypeVoteTogether
	}

	if len(body.HashLink) != 0 && len(body.Inputs) == 0 {
		if len(body.HashLink) != HashSize {
			fmt.Println("err: invalid hash size")
			return CodeHashLinkInvalidLen
		}
		// this is coinbase tx, need to check receiver and double spending
		if len(body.Outputs) != 1 {
			fmt.Println("err: coinbase tx has incorrect number of outputs")
			return CodeCoinbaseTxNoOutput
		}
		if body.Duration != 0 {
			fmt.Println("err: unexpected duration in coinbase tx")
			return CodeCoinbaseUnexpectedDuration
		}
		if len(body.SenderEphemeralPkey) != 0 {
			fmt.Println("err: unexpected sender ephemeral pkey in coinbase tx")
			return CodeCoinbaseUnexpectedSenderEphemeralPkey
		}
		if len(body.VotersSumPkey) != 0 {
			fmt.Println("err: unexpected voters sum pkey in coinbase tx")
			return CodeCoinbaseUnexpectedVotersSumPkey
		}

		pkey := body.Outputs[0].ReceiverSpendPkey
		if len(body.Outputs[0].ReceiverScanPkey) != 0 {
			fmt.Println("err: unexpected scan key in coinbase tx")
			return CodeUnexpectedScanKey
		}
		rewardValue := body.Outputs[0].Value
		rewardBlock := body.HashLink
		duplicate, err := t.db.GetTxByHashLink(rewardBlock)
		if err != nil {
			panic(err)
		}
		if duplicate != nil {
			fmt.Println("err: already has coinbase tx for the same block")
			return CodeDoubleCoinbaseForSameBlock
		}
		block, err := t.db.GetBlockByHash(rewardBlock)
		if err != nil {
			panic(err)
		}
		if block == nil {
			fmt.Println("err: no block for coinbase tx")
			return CodeCoinbaseNoBlock
		}
		if !bytes.Equal(block.BlockHeader.ProposerPkey, pkey) {
			fmt.Println("err: coinbase tx proposer mismatch")
			return CodeCoinbaseProposerMismatch
		}
		if rewardValue != RewardCoins {
			fmt.Println("err: incorrect reward")
			return CodeCoinbaseIncorrectReward
		}
		return t.verifySigAndAppend(&tx, hashBytes, pkey)
	}

	var inputsSum, outputsSum uint32
	var pkey []byte
	for i, input := range body.Inputs {
		var correspondingUtxo *golosovaniepb.Utxo
		utxos, err := t.db.GetUtxosByTxHash(input.PrevTxHash)
		if err != nil {
			fmt.Println("database failed", err)
			return CodeDatabaseFailed
		}
		// проверка, что вход - непотраченный выход дургой транзы
		for _, utxo := range utxos {
			if utxo.Index == input.OutputIndex {
				correspondingUtxo = utxo
				if i == 0 {
					pkey = correspondingUtxo.ReceiverSpendPkey
				} else {
					if !bytes.Equal(pkey, correspondingUtxo.ReceiverSpendPkey) {
						fmt.Println("err: input not owned by a sender")
						return CodeInputNotOwn
					}
				}
				break
			}
		}
		if correspondingUtxo == nil {
			fmt.Println("err: double spending in tx")
			return CodeDoubleSpending
		}
		inputsSum += correspondingUtxo.Value
		// проверка, что в одной транзе не смешиваются разные typeValue
		if len(body.HashLink) == 0 && body.VoteType == 0 && !bytes.Equal(correspondingUtxo.ValueType, body.ValueType) {
			fmt.Println("err: incorrect typeValue in input", input)
			return CodeMixingTypeValue
		}
		// проверка, что в транзакции создания голосования не используются голоса
		if body.VoteType != 0 && len(correspondingUtxo.ValueType) != 0 {
			fmt.Println("err: cannot use votes as funding for creating new voting")
			return CodeVotesUsedAsFunding
		}
	}
	var outputsWithScanKey int
	for _, output := range body.Outputs {
		outputsSum += output.Value
		if len(output.ReceiverScanPkey) != 0 {
			outputsWithScanKey += 1
		}
	}
	if len(body.Outputs) != outputsWithScanKey && outputsWithScanKey != 0 {
		fmt.Println("err: outputs have both nil and not nil scan keys")
		return CodeOutputsHaveBothNilAndNotNilScanKeys
	}
	if outputsSum != inputsSum {
		fmt.Printf("err: outputs sum %v is not matching than inputs sum %v\n", outputsSum, inputsSum)
		return CodeInputsNotMatchOutputs
	}
	if body.VoteType != 0 {
		// транзакция создания голосования
		// проверка что HashLink == nil выше
		// valueType не нужно проверять, в цикле по инпутам есть проверка,
		// что он нулевой у всех инпутов и одинаковый с телом транзакции
		if len(body.SenderEphemeralPkey) != 0 {
			fmt.Println("err: create voting tx has unexpected sender ephemeral pkey")
			return CodeCreateVoteTxUnexpectedSenderEphemeralPkey
		}
		if len(body.VotersSumPkey) != 0 {
			fmt.Println("err: create voting tx has unexpected voters sum pkey")
			return CodeCreateVoteTxUnexpectedVotersSumPkey
		}
	} else if len(body.HashLink) != 0 {
		// транзакция дополнения голосования
		// или дополнения инициализации голосования
		fmt.Println("err: not supported")
		return CodeNotSupported
	} else {
		if len(body.ValueType) != 0 {
			// транзакция отправки голоса или инициализации голосования
			if len(body.SenderEphemeralPkey) != 0 && len(body.VotersSumPkey) != 0 {
				// транзакция инициализации голосования
				if len(body.SenderEphemeralPkey) != PkeySize {
					fmt.Println("err: invalid sender ephemeral pkey len")
					return CodeInvalidSenderEphemeralPkeyLen
				}
				if len(body.VotersSumPkey) != PkeySize {
					fmt.Println("err: invalid voters sum pkey len")
					return CodeInvalidVotersSumPkeyLen
				}
				if outputsWithScanKey != 0 {
					fmt.Println("err: init vote tx must have nil scan pkeys")
					return CodeInitVoteMustHaveNilScanPkeys
				}
				createVoteTx, err := t.db.GetTxByHash(body.ValueType)
				if err != nil {
					fmt.Println("database failed", err)
					return CodeDatabaseFailed
				}
				var createVoteBody golosovaniepb.TxBody
				err = proto.Unmarshal(createVoteTx.TxBody, &createVoteBody)
				if err != nil {
					fmt.Println("parse create vote body error", err)
					return CodeParseErr
				}
				// ValueType должен ссылаться на транзакцию создания голосования
				if createVoteBody.VoteType == 0 {
					fmt.Println("err: create vote tx, to which valueType points, is not create vote tx")
					return CodeValueTypeInvalid
				}
				// проверка, что число избирателей сохранилось во время инициализации
				// вообще проверка не нужна, так как покрывается проверкой voters_sum_pkey
				// однако для более читаемых ошибок она присутствует
				if len(createVoteBody.Outputs) != len(body.Outputs) {
					fmt.Println("err: init vote tx participant number differs from one in create vote tx")
					return CodeInvalidVoteParticipantNumber
				}
				// проверить, что sign соответствует публичному ключу валидатора,
				// в блок предложенный которым попала транзакция создания голосования

				// проверить voters_sum_pkey
			}
		}
	}

	return t.verifySigAndAppend(&tx, hashBytes, pkey)
}

func (t *TxExecutor) verifySigAndAppend(
	tx *golosovaniepb.Transaction, hashBytes [HashSize]byte, pkey []byte,
) (code uint32) {
	if len(tx.Sig) != SigSize {
		return CodeInvalidSignatureLen
	}
	if !VerifyData(tx.TxBody, tx.Sig, pkey) {
		fmt.Println("err: signature doesnt match")
		return CodeInvalidSignature
	}
	t.Transactions = append(t.Transactions, tx)
	t.processedTrans[hashBytes] = true
	return CodeOk
}
