package evote

import (
	"GO_LOSOVANIE/evote/golosovaniepb"
	"bytes"
	"fmt"
	"github.com/golang/protobuf/proto"
	"time"
)

func OnGetTxsByHashes(db *Database, req *golosovaniepb.RequestTxsByHashes) (code uint32, err error, resp *golosovaniepb.Response) {
	if req == nil || len(req.GetHashes()) == 0 {
		return CodeRequestEmpty, fmt.Errorf("request fields are empty"), nil
	}
	txs, err := db.GetTxsByHashes(req.GetHashes())
	if err != nil {
		return CodeDatabaseFailed, err, nil
	}
	txsByHashes := golosovaniepb.ResponseTxsByHashes{
		Txs: txs,
	}
	return CodeOk, nil, &golosovaniepb.Response{
		Data: &golosovaniepb.Response_TxsByHashes{TxsByHashes: &txsByHashes},
	}
}

func OnGetTxsByPkey(db *Database, req *golosovaniepb.RequestTxsByPkey) (code uint32, err error, resp *golosovaniepb.Response) {
	if req == nil || len(req.Pkey) == 0 {
		return CodeRequestEmpty, fmt.Errorf("request fields are empty"), nil
	}
	if len(req.Pkey) != PkeySize {
		return CodeInvalidDataLen, fmt.Errorf("pkey must be exactly %d bytes", PkeySize), nil
	}
	txs, err := db.GetTxsByPubKey(req.Pkey)
	if err != nil {
		return CodeDatabaseFailed, err, nil
	}
	txsByPkey := golosovaniepb.ResponseTxsByPkey{
		Txs: txs,
	}
	return CodeOk, nil, &golosovaniepb.Response{
		Data: &golosovaniepb.Response_TxsByPkey{TxsByPkey: &txsByPkey},
	}
}

func OnGetUtxosByPkey(db *Database, req *golosovaniepb.RequestUtxosByPkey) (code uint32, err error, resp *golosovaniepb.Response) {
	if req == nil || len(req.Pkey) == 0 {
		return CodeRequestEmpty, fmt.Errorf("request fields are empty"), nil
	}
	if len(req.Pkey) != PkeySize {
		return CodeInvalidDataLen, fmt.Errorf("pkey must be exactly %d bytes", PkeySize), nil
	}
	utxos, err := db.GetUTXOSByPkey(req.Pkey)
	if err != nil {
		return CodeDatabaseFailed, err, nil
	}
	utxosByPkey := golosovaniepb.ResponseUtxosByPkey{
		Utxos: utxos,
	}
	return CodeOk, nil, &golosovaniepb.Response{
		Data: &golosovaniepb.Response_UtxosByPkey{UtxosByPkey: &utxosByPkey},
	}
}

/*
OnFaucet
input: uint32_t moneyRequest + pkey_bytes_str
output: ok/false + err_msg
*/
func OnFaucet(db *Database, n *Network, key *CryptoKeysData, req *golosovaniepb.RequestFaucet) (code uint32, err error, resp *golosovaniepb.Response) {
	if req == nil || len(req.Pkey) != PkeySize {
		return CodeInvalidDataLen, fmt.Errorf("pkey must be exactly %d bytes", PkeySize), nil
	}
	if req.Value <= 0 {
		return CodeInvalidValue, fmt.Errorf("value must be greater than zero"), nil
	}

	utxos, err := db.GetUTXOSByPkey(key.PkeyByte[:])
	if err != nil {
		return CodeDatabaseFailed, err, nil
	}
	var outputs = make(map[[PkeySize]byte]uint32, 0)
	outputs[SliceToPkey(req.Pkey)] = req.Value
	tx, err := CreateTx(utxos, outputs, nil, key, 0, 0, false)
	if err != nil {
		return CodeCannotCreateTx, fmt.Errorf("error while creating transaction"), nil
	}
	txBytes, err := proto.Marshal(tx)
	if err != nil {
		return CodeSerializeErr, err, nil
	}
	go func() {
		err := n.SubmitTx(txBytes)
		if err != nil {
			fmt.Println("error while submitting faucet tx", err)
		}
	}()

	return CodeOk, nil, &golosovaniepb.Response{
		Data: &golosovaniepb.Response_Faucet{
			Faucet: &golosovaniepb.ResponseFaucet{Tx: tx},
		},
	}
}

func getVoteValue(value, typeVote uint32) uint32 {
	if typeVote == OneVoteType {
		return 1
	}
	if typeVote == PercentVoteType {
		return value
	}
	return value
}

func OnGetVoteResult(db *Database, req *golosovaniepb.RequestVoteResult) (code uint32, err error, resp *golosovaniepb.Response) {
	if req == nil || len(req.VoteTxHash) != HashSize {
		return CodeInvalidDataLen, fmt.Errorf("incorrect transaction hash length"), nil
	}
	t, timeStart, err := db.GetTxAndTimeByHash(req.VoteTxHash)
	if err != nil {
		return CodeDatabaseFailed, err, nil
	}
	var body golosovaniepb.TxBody
	err = proto.Unmarshal(t.TxBody, &body)
	if err != nil {
		return CodeParseErr, err, nil
	}
	endTime := timeStart + uint64(time.Second)*uint64(body.Duration)
	utxos, err := db.GetUTXOSByTypeValue(req.VoteTxHash)
	if err != nil {
		return CodeDatabaseFailed, err, nil
	}

	//блок подсчета голосов надо разбить на разные функции, так как
	//в при некоторых случаях один и тот же избиратель может голосовать дважды,
	//может голосовать "за", "против", "воздержался" и т.п.
	//так же может происходит сортировка результатов гослования в зависимости от его типа
	result := make(map[[PkeySize]byte]uint32, 0)
	for _, utxo := range utxos {
		pkey := SliceToPkey(utxo.ReceiverSpendPkey)
		_, contains := result[pkey]
		if bytes.Equal(utxo.ValueType, t.Hash) && utxo.Timestamp < endTime {
			if contains {
				result[pkey] += getVoteValue(utxo.Value, body.VoteType)
			} else {
				result[pkey] = getVoteValue(utxo.Value, body.VoteType)
			}
		}
	}

	for {
		for _, out := range body.Outputs {
			pkey := SliceToPkey(out.ReceiverSpendPkey)
			_, contains := result[pkey]
			if contains {
				delete(result, pkey)
			}
		}
		t, err = db.GetTxByHashLink(t.Hash)
		if err != nil {
			return CodeDatabaseFailed, err, nil
		}
		if t == nil {
			break
		}
	}

	//create bytes result
	fmt.Println("result voting with voteTxHash: ", req.VoteTxHash)
	var res golosovaniepb.ResponseVoteResult
	for pkey, val := range result {
		fmt.Println(pkey, val)
		res.Res = append(
			res.Res,
			&golosovaniepb.ResponseVoteResult_PkeyValue{
				Pkey:  pkey[:],
				Value: val,
			},
		)
	}

	return CodeOk, nil, &golosovaniepb.Response{
		Data: &golosovaniepb.Response_VoteResult{VoteResult: &res},
	}
}
