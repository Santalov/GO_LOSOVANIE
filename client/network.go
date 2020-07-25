package client

import (
	"GO_LOSOVANIE/evote"
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
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
		fmt.Println("network err: ", err)
		response <- ""
	} else {
		if resp.StatusCode != http.StatusOK {
			body, _ := ioutil.ReadAll(resp.Body)
			fmt.Printf("network: server answered with error %v, body: %v\n", resp.Status, string(body))
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

func (n *Network) selectNextHost() {
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
	if len(n.workingHosts) == 0 {
		panic("No available validators. Client need a validator to work with")
	}
}

func (n *Network) Init(allHosts []string, key [evote.PKEY_SIZE]byte) {
	n.allHosts = allHosts
	n.createWorkingHosts()
}

func (n *Network) GetTxsByPkey(pkey [evote.PKEY_SIZE]byte) ([]*evote.Transaction, error) {
	data, err := makeBinaryRequest(n.curHost, "/getTxsByPubKey", pkey[:])
	if err != nil {
		return nil, err
	}
	transSize := binary.LittleEndian.Uint32(data[:evote.INT_32_SIZE])
	offset := evote.INT_32_SIZE
	txs := make([]*evote.Transaction, 0)
	for i := 0; i < int(transSize); i++ {
		tx := new(evote.Transaction)
		txLen := tx.FromBytes(data[offset:])
		if txLen > 0 {
			offset += txLen
		} else {
			return nil, fmt.Errorf("incorrect transaction in response from validator")
		}
		txs = append(txs, tx)
	}
	return txs, nil
}
