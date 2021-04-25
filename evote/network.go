package evote

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	rpctypes "github.com/tendermint/tendermint/rpc/jsonrpc/types"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
)

type Network struct {
	workingHosts []string
	allHosts     []string
	curHost      string
}

// Copy used to make separate instance for working in another thread
func (n *Network) Copy() *Network {
	workingHosts := make([]string, len(n.workingHosts))
	copy(workingHosts, n.workingHosts)
	allHosts := make([]string, len(n.allHosts))
	copy(allHosts, n.allHosts)
	return &Network{
		workingHosts,
		allHosts,
		n.curHost,
	}
}

func (n *Network) makeBinaryPostRequest(host string, endPoint string, data []byte) (response []byte, err error) {
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

func (n *Network) makeGetRequest(host string, path string, params url.Values) (response []byte, err error) {
	rawQuery := ""
	if params != nil {
		rawQuery = params.Encode()
	}
	u := &url.URL{Scheme: "http", Host: host, Path: path, RawQuery: rawQuery}
	fmt.Println("request url:", u.String())
	resp, err := http.Get(u.String())
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

func (n *Network) makeInfoRequest(host string, response chan string) {
	resp, err := n.makeGetRequest(host, "/abci_info", nil)
	if err != nil {
		fmt.Printf("network: server answered with error %v, body: %v\n", err, string(resp))
		response <- ""
	} else {
		response <- host
	}
}

func (n *Network) pingHosts(hosts []string) (alive []string) {
	responses := make(chan string, len(hosts))
	for _, host := range hosts {
		go n.makeInfoRequest(host, responses)
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
		n.workingHosts = n.pingHosts(n.allHosts)
		if len(n.workingHosts) == 0 {
			panic("No available validators. Client need a validator to work with")
		}
	}
	n.curHost = n.workingHosts[rand.Int()%len(n.workingHosts)]
}

func (n *Network) createWorkingHosts() {
	n.workingHosts = n.pingHosts(n.allHosts)
	fmt.Println(len(n.workingHosts), "validators online")
	if len(n.workingHosts) == 0 {
		panic("No available validators. Client need a validator to work with")
	}
}

func (n *Network) Init(allHosts []string) {
	n.allHosts = allHosts
	n.workingHosts = allHosts
	n.curHost = n.workingHosts[rand.Int()%len(n.workingHosts)]
}

func (n *Network) PingAll() {
	n.createWorkingHosts()
}

func parseTrans(data []byte) ([]*Transaction, error) {
	transSize := binary.LittleEndian.Uint32(data[:INT_32_SIZE])
	offset := INT_32_SIZE
	txs := make([]*Transaction, 0)
	for i := 0; i < int(transSize); i++ {
		tx := new(Transaction)
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

func (n *Network) BroadcastTxSync(tx []byte) (*rpctypes.RPCResponse, error) {
	respRaw, err := n.makeGetRequest(n.curHost, "/broadcast_tx_sync", url.Values{"tx": {"0x" + hex.EncodeToString(tx)}})
	if err != nil {
		return nil, err
	}
	response := &rpctypes.RPCResponse{}
	err = response.UnmarshalJSON(respRaw)
	if err != nil {
		return nil, err
	}
	return response, err
}

func (n *Network) GetTxsByHashes(hashes [][HASH_SIZE]byte) ([]*Transaction, error) {
	reqData := make([]byte, len(hashes)*HASH_SIZE+INT_32_SIZE)
	binary.LittleEndian.PutUint32(reqData[:INT_32_SIZE], uint32(len(hashes)))
	offset := INT_32_SIZE
	for _, h := range hashes {
		copy(reqData[offset:offset+HASH_SIZE], h[:])
		offset += HASH_SIZE
	}
	data, err := n.makeBinaryPostRequest(n.curHost, "/getTxs", reqData)
	if err != nil {
		return nil, err
	}
	return parseTrans(data)
}

func (n *Network) GetTxsByPkey(pkey [PKEY_SIZE]byte) ([]*Transaction, error) {
	data, err := n.makeBinaryPostRequest(n.curHost, "/getTxsByPubKey", pkey[:])
	if err != nil {
		return nil, err
	}
	return parseTrans(data)
}

func (n *Network) GetUtxosByPkey(pkey [PKEY_SIZE]byte) ([]*UTXO, error) {
	data, err := n.makeBinaryPostRequest(n.curHost, "/getUTXOByPubKey", pkey[:])
	if err != nil {
		return nil, err
	}
	utxosSize := binary.LittleEndian.Uint32(data[:INT_32_SIZE])
	offset := INT_32_SIZE
	utxos := make([]*UTXO, 0)
	for i := 0; i < int(utxosSize); i++ {
		utxo := new(UTXO)
		retCode := utxo.FromBytes(data[offset : offset+UTXO_SIZE])
		if retCode != OK {
			fmt.Println("incorrect utxo from validator")
			return nil, fmt.Errorf("incorrect utxo from validator")
		}
		utxos = append(utxos, utxo)
		offset += UTXO_SIZE
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

func (n *Network) Faucet(amount uint32, pkey [PKEY_SIZE]byte) error {
	data := make([]byte, INT_32_SIZE+PKEY_SIZE)
	binary.LittleEndian.PutUint32(data[:INT_32_SIZE], amount)
	copy(data[INT_32_SIZE:], pkey[:])
	resp, err := http.Post("http://"+n.curHost+"/faucet", "application/octet-stream", bytes.NewReader(data))
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

func (n *Network) VoteResults(hash [HASH_SIZE]byte) (map[[PKEY_SIZE]byte]uint32, error) {
	data, err := n.makeBinaryPostRequest(n.curHost, "/getVoteResult", hash[:])
	if err != nil {
		return nil, err
	}
	itemSize := PKEY_SIZE + INT_32_SIZE
	resLen := len(data) / itemSize
	if len(data)%itemSize != 0 {
		return nil, errors.New("incorrect result len")
	}
	results := make(map[[PKEY_SIZE]byte]uint32)
	for i := 0; i < resLen; i++ {
		var candidate [PKEY_SIZE]byte
		copy(candidate[:], data[i*itemSize:i*itemSize+PKEY_SIZE])
		results[candidate] =
			binary.LittleEndian.Uint32(data[i*itemSize+PKEY_SIZE : i*itemSize+PKEY_SIZE+INT_32_SIZE])
	}
	return results, nil
}
