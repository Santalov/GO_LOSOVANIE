package evote

import (
	"encoding/binary"
	"fmt"
	"time"
)

func OnGetTxsByHashes(db *Database, data []byte) (code uint32, err error, value []byte) {
	if len(data) < Int32Size {
		return CodeInvalidDataLen, fmt.Errorf("too short data"), nil
	}
	hashesNum := binary.LittleEndian.Uint32(data[:Int32Size])
	if int(hashesNum)*HashSize+Int32Size != len(data) {
		return CodeInvalidDataLen, fmt.Errorf("data len doesn't match hashes counter"), nil
	}
	offset := Int32Size
	hashes := make([][HashSize]byte, 0)
	for offset+HashSize <= len(data) {
		hash := [HashSize]byte{}
		copy(hash[:], data[offset:offset+HashSize])
		hashes = append(hashes, hash)
		offset += HashSize
	}
	txs, err := db.GetTxsByHashes(hashes)
	if err != nil {
		return CodeDatabaseFailed, err, nil
	}
	txsPacked := make([]byte, Int32Size)
	binary.LittleEndian.PutUint32(txsPacked, uint32(len(txs)))
	for _, tx := range txs {
		txsPacked = append(txsPacked, tx.Transaction.ToBytes()...)
	}
	return CodeOk, nil, txsPacked
}

func OnGetTxsByPkey(db *Database, data []byte) (code uint32, err error, value []byte) {
	if len(data) != PkeySize {
		return CodeInvalidDataLen, fmt.Errorf("incorrect pkey length"), nil
	}
	pkey := [PkeySize]byte{}
	copy(pkey[:], data)
	txs, err := db.GetTxsByPubKey(pkey)
	if err != nil {
		return CodeDatabaseFailed, err, nil
	}
	txsPacked := make([]byte, Int32Size)
	binary.LittleEndian.PutUint32(txsPacked, uint32(len(txs)))
	for _, tx := range txs {
		txsPacked = append(txsPacked, tx.Transaction.ToBytes()...)
	}
	return CodeOk, nil, txsPacked
}

func OnGetUtxosByPkey(db *Database, data []byte) (code uint32, err error, value []byte) {
	if len(data) != PkeySize {
		return CodeInvalidDataLen, fmt.Errorf("incorrect pkey length"), nil
	}
	pkey := [PkeySize]byte{}
	copy(pkey[:], data)
	utxos, err := db.GetUTXOSByPkey(pkey)
	if err != nil {
		return CodeDatabaseFailed, err, nil
	}
	utxosPacked := make([]byte, Int32Size)
	binary.LittleEndian.PutUint32(utxosPacked, uint32(len(utxos)))
	for _, utxo := range utxos {
		utxosPacked = append(utxosPacked, utxo.ToBytes()...)
	}
	return CodeOk, nil, utxosPacked
}

/*
OnFaucet
input: uint32_t moneyRequest + pkey_bytes_str
output: ok/false + err_msg
*/
func OnFaucet(db *Database, n *Network, key *CryptoKeysData, data []byte) (code uint32, err error, value []byte) {
	if len(data) != Int32Size+PkeySize {
		return CodeInvalidDataLen, fmt.Errorf("expected data to be int and pkey, but length does not match"), nil
	}

	var pkey [PkeySize]byte
	var amount = binary.LittleEndian.Uint32(data[:Int32Size])
	copy(pkey[:], data[Int32Size:])
	utxos, err := db.GetUTXOSByPkey(key.PkeyByte)
	if err != nil {
		return CodeDatabaseFailed, err, nil
	}
	var t Transaction
	var outputs = make(map[[PkeySize]byte]uint32, 0)
	outputs[pkey] = amount
	errCreate := t.CreateTrans(utxos, outputs, ZeroArrayHash, key, 0, 0, false)
	if errCreate != OK {
		return uint32(errCreate), fmt.Errorf("error while creating transaction"), nil
	}
	transBytes := t.ToBytes()
	go func() {
		err := n.SubmitTx(transBytes)
		if err != nil {
			fmt.Println("error while submitting faucet tx", err)
		}
	}()

	return CodeOk, nil, transBytes
}

func getVoteValue(value, typeVote uint32) int32 {
	if typeVote == OneVoteType {
		return 1
	}
	if typeVote == PercentVoteType {
		return int32(value)
	}
	return int32(value)
}

func OnGetVoteResult(db *Database, data []byte) (code uint32, err error, value []byte) {
	if len(data) != HashSize {
		return CodeInvalidDataLen, fmt.Errorf("incorrect transaction hash length"), nil
	}
	var mainHash [HashSize]byte
	copy(mainHash[:], data[:])
	t, timeStart, err := db.GetTxAndTimeByHash(mainHash)
	if err != nil {
		return CodeDatabaseFailed, err, nil
	}
	endTime := timeStart + uint64(time.Second)*uint64(t.Transaction.Duration)
	utxos, err := db.GetUTXOSByTypeValue(mainHash)
	if err != nil {
		return CodeDatabaseFailed, err, nil
	}

	//блок подсчета голосов надо разбить на разные функции, так как
	//в при некоторых случаях один и тот же избиратель может голосовать дважды,
	//может голосовать "за", "против", "воздержался" и т.п.
	//так же может происходит сортировка результатов гослования в зависимости от его типа
	result := make(map[[PkeySize]byte]int32, 0)
	for _, utxo := range utxos {
		_, contains := result[utxo.PkeyTo]
		if utxo.TypeValue == t.Hash && utxo.Timestamp < endTime {
			if contains {
				result[utxo.PkeyTo] += getVoteValue(utxo.Value, t.Transaction.TypeVote)
			} else {
				result[utxo.PkeyTo] = getVoteValue(utxo.Value, t.Transaction.TypeVote)
			}
		}
	}

	for {
		for _, out := range t.Transaction.Outputs {
			_, contains := result[out.PkeyTo]
			if contains {
				delete(result, out.PkeyTo)
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
	var resBytes []byte
	var valBytes [Int32Size]byte
	fmt.Println("result voting with mainHash: ", mainHash)
	for pkey, val := range result {
		fmt.Println(pkey, val)
		binary.LittleEndian.PutUint32(valBytes[:], uint32(val))
		resBytes = append(resBytes, pkey[:]...)
		resBytes = append(resBytes, valBytes[:]...)
	}
	fmt.Println()

	return CodeOk, nil, resBytes
}
