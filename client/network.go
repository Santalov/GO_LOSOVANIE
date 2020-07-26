package main

import (
	"GO_LOSOVANIE/evote"
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
)

type Network struct {
	workingHosts []string
	allHosts     []string
	curHost      string
}

func makeBinaryRequest(host string, endPoint string, data []byte) (response []byte, err error) {
	resp, err := http.Post("http://"+host+endPoint, "application/octet-stream", bytes.NewReader(data))
	if err != nil {
		fmt.Println("request err: ", err)
		return nil, err
	} else {
		if resp.StatusCode != http.StatusOK {
			body, _ := ioutil.ReadAll(resp.Body)
			fmt.Printf("validator answered with error %v, body: %v\n", resp.Status, string(body))
			return nil, fmt.Errorf(string(body))
		} else {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println("request err: body cannot be read, err:")
				return nil, err
			}
			return body, nil
		}
	}
}

func makeInfoRequest(host string, response chan string) {
	resp, err := http.Get("http://" + host + "/info")
	if err != nil {
		response <- ""
	} else {
		if resp.StatusCode != http.StatusOK {
			response <- ""
		} else {
			response <- host
		}
	}
}

func pingHosts(hosts []string) (alive []string) {
	responses := make(chan string, len(hosts))
	for _, host := range hosts {
		go makeInfoRequest(host, responses)
	}
	for _ = range hosts {
		h := <-responses
		if h != "" {
			alive = append(alive, h)
		}
	}
	return
}

func (n *Network) SelectNextHost() {
	clearedHosts := make([]string, 0)
	for i, host := range n.workingHosts {
		if host != n.curHost {
			// remove not working host
			clearedHosts = append(clearedHosts, host)
		} else {
			// select next host
			n.curHost = n.workingHosts[(i+1)%len(n.workingHosts)]
		}
	}
	n.workingHosts = clearedHosts
	if len(n.workingHosts) == 0 {
		n.workingHosts = pingHosts(n.allHosts)
		if len(n.workingHosts) == 0 {
			panic("No available validators. Client need a validator to work with")
		}
	}
}

func (n *Network) createWorkingHosts() {
	n.workingHosts = pingHosts(n.allHosts)
	fmt.Println(len(n.workingHosts), "validators online")
	if len(n.workingHosts) == 0 {
		panic("No available validators. Client need a validator to work with")
	}
}

func (n *Network) Init(allHosts []string) {
	n.allHosts = allHosts
	n.createWorkingHosts()
	n.curHost = n.workingHosts[rand.Int()%len(n.workingHosts)]
}

func parseTrans(data []byte) ([]*evote.Transaction, error) {
	transSize := binary.LittleEndian.Uint32(data[:evote.INT_32_SIZE])
	offset := evote.INT_32_SIZE
	txs := make([]*evote.Transaction, 0)
	for i := 0; i < int(transSize); i++ {
		tx := new(evote.Transaction)
		txLen := tx.FromBytes(data[offset:])
		if txLen > 0 {
			offset += txLen
			txs = append(txs, tx)
		} else {
			fmt.Println("incorrect transaction in response from validator")
			return nil, fmt.Errorf("incorrect transaction in response from validator")
		}
	}
	return txs, nil
}

func (n *Network) GetTxsByHashes(hashes [][evote.HASH_SIZE]byte) ([]*evote.Transaction, error) {
	reqData := make([]byte, len(hashes)*evote.HASH_SIZE+evote.INT_32_SIZE)
	binary.LittleEndian.PutUint32(reqData[:evote.INT_32_SIZE], uint32(len(hashes)))
	offset := evote.INT_32_SIZE
	for _, h := range hashes {
		copy(reqData[offset:offset+evote.HASH_SIZE], h[:])
		offset += evote.HASH_SIZE
	}
	data, err := makeBinaryRequest(n.curHost, "/getTxs", reqData)
	if err != nil {
		return nil, err
	}
	return parseTrans(data)
}

func (n *Network) GetTxsByPkey(pkey [evote.PKEY_SIZE]byte) ([]*evote.Transaction, error) {
	data, err := makeBinaryRequest(n.curHost, "/getTxsByPubKey", pkey[:])
	if err != nil {
		return nil, err
	}
	return parseTrans(data)
}

func (n *Network) GetUtxosByPkey(pkey [evote.PKEY_SIZE]byte) ([]*evote.UTXO, error) {
	data, err := makeBinaryRequest(n.curHost, "/getUTXOByPubKey", pkey[:])
	if err != nil {
		return nil, err
	}
	utxosSize := binary.LittleEndian.Uint32(data[:evote.INT_32_SIZE])
	offset := evote.INT_32_SIZE
	utxos := make([]*evote.UTXO, 0)
	for i := 0; i < int(utxosSize); i++ {
		utxo := new(evote.UTXO)
		retCode := utxo.FromBytes(data[offset : offset+evote.UTXO_SIZE])
		if retCode != evote.OK {
			fmt.Println("incorrect utxo from validator")
			return nil, fmt.Errorf("incorrect utxo from validator")
		}
		utxos = append(utxos, utxo)
		offset += evote.UTXO_SIZE
	}
	return utxos, nil
}

func (n *Network) SubmitTx(tx []byte) error {
	resp, err := http.Post("http://"+n.curHost+"/submitClientTx", "application/octet-stream", bytes.NewReader(tx))
	if err != nil {
		fmt.Println("request err: ", err)
		return err
	} else {
		if resp.StatusCode != http.StatusOK {
			body, _ := ioutil.ReadAll(resp.Body)
			fmt.Printf("validator answered with error %v, body: %v\n", resp.Status, string(body))
			return fmt.Errorf(string(body))
		} else {
			return nil
		}
	}
}
