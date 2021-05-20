package evote

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	abcitypes "github.com/tendermint/tendermint/abci/types"
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

func (n *Network) makeGetRequest(host string, path string, params url.Values) (response []byte, err error) {
	rawQuery := ""
	if params != nil {
		rawQuery = params.Encode()
	}
	u := &url.URL{Scheme: "http", Host: host, Path: path, RawQuery: rawQuery}
	//fmt.Println("request url:", u.String())
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
	for range hosts {
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
	transSize := binary.LittleEndian.Uint32(data[:Int32Size])
	offset := Int32Size
	txs := make([]*Transaction, 0)
	for i := 0; i < int(transSize); i++ {
		tx := new(Transaction)
		txLen := tx.FromBytes(data[offset:])
		if txLen > 0 {
			offset += txLen
			txs = append(txs, tx)
		} else {
			return nil, fmt.Errorf("incorrect transaction in response from validator")
		}
	}
	return txs, nil
}

func toRpcResp(respRaw []byte, err error) (*rpctypes.RPCResponse, error) {
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

func toRpcResult(respRaw []byte, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	resp, err := toRpcResp(respRaw, err)
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	return resp.Result.MarshalJSON()
}

// WrappedResponse is used only because response contains unknown filed "response"
type WrappedResponse struct {
	Response *abcitypes.ResponseQuery `json:"response"`
}

func toResponseQuery(result []byte, err error) (*abcitypes.ResponseQuery, error) {
	if err != nil {
		return nil, err
	}
	var wrappedResponse WrappedResponse
	//fmt.Println("result", string(result))
	err = json.Unmarshal(result, &wrappedResponse)
	//fmt.Println("toResponseQuery", err)
	if err != nil {
		return nil, err
	}
	return wrappedResponse.Response, err
}

func (n *Network) abciQueryResponse(path string, data []byte) (*abcitypes.ResponseQuery, error) {
	//fmt.Println("request to path", path, "with binary data", data)
	return toResponseQuery(
		toRpcResult(
			n.makeGetRequest(
				n.curHost, "/abci_query",
				url.Values{
					"path": {"\"" + path + "\""},
					"data": {"0x" + hex.EncodeToString(data)},
				},
			),
		),
	)
}

// this function is used server does not send error codes, so we can just use value
func (n *Network) abciQueryValue(path string, data []byte) ([]byte, error) {
	resp, err := n.abciQueryResponse(path, data)
	if err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("validator answered with not null code %d, log %v", resp.Code, resp.Log)
	}
	return resp.Value, err
}

func (n *Network) BroadcastTxSync(tx []byte) ([]byte, error) {
	return toRpcResult(
		n.makeGetRequest(
			n.curHost,
			"/broadcast_tx_sync",
			url.Values{
				"tx": {"0x" + hex.EncodeToString(tx)},
			}),
	)
}

func (n *Network) GetTxsByHashes(hashes [][HashSize]byte) ([]*Transaction, error) {
	reqData := make([]byte, len(hashes)*HashSize+Int32Size)
	binary.LittleEndian.PutUint32(reqData[:Int32Size], uint32(len(hashes)))
	offset := Int32Size
	for _, h := range hashes {
		copy(reqData[offset:offset+HashSize], h[:])
		offset += HashSize
	}
	data, err := n.abciQueryValue("getTxs", reqData)
	if err != nil {
		return nil, err
	}
	return parseTrans(data)
}

func (n *Network) GetTxsByPkey(pkey [PkeySize]byte) ([]*Transaction, error) {
	data, err := n.abciQueryValue("getTxsByPubKey", pkey[:])
	if err != nil {
		return nil, err
	}
	return parseTrans(data)
}

func (n *Network) GetUtxosByPkey(pkey [PkeySize]byte) ([]*UTXO, error) {
	data, err := n.abciQueryValue("getUtxosByPubKey", pkey[:])
	if err != nil {
		return nil, err
	}
	utxosSize := binary.LittleEndian.Uint32(data[:Int32Size])
	offset := Int32Size
	utxos := make([]*UTXO, 0)
	for i := 0; i < int(utxosSize); i++ {
		utxo := new(UTXO)
		retCode := utxo.FromBytes(data[offset : offset+UtxoSize])
		if retCode != OK {
			return nil, fmt.Errorf("incorrect utxo from validator")
		}
		utxos = append(utxos, utxo)
		offset += UtxoSize
	}
	return utxos, nil
}

func (n *Network) SubmitTx(tx []byte) error {
	_, err := n.BroadcastTxSync(tx)
	return err
}

func (n *Network) Faucet(amount uint32, pkey [PkeySize]byte) error {
	data := make([]byte, Int32Size+PkeySize)
	binary.LittleEndian.PutUint32(data[:Int32Size], amount)
	copy(data[Int32Size:], pkey[:])
	resp, err := n.abciQueryResponse("faucet", data)
	if err != nil {
		return err
	} else {
		if resp.Code != 0 {
			return fmt.Errorf("validator answered with code %v, log: %v\n", resp.Code, resp.Log)
		} else {
			return nil
		}
	}
}

func (n *Network) VoteResults(hash [HashSize]byte) (map[[PkeySize]byte]uint32, error) {
	data, err := n.abciQueryValue("getVoteResult", hash[:])
	if err != nil {
		return nil, err
	}
	itemSize := PkeySize + Int32Size
	resLen := len(data) / itemSize
	if len(data)%itemSize != 0 {
		return nil, errors.New("incorrect result len")
	}
	results := make(map[[PkeySize]byte]uint32)
	for i := 0; i < resLen; i++ {
		var candidate [PkeySize]byte
		copy(candidate[:], data[i*itemSize:i*itemSize+PkeySize])
		results[candidate] =
			binary.LittleEndian.Uint32(data[i*itemSize+PkeySize : i*itemSize+PkeySize+Int32Size])
	}
	return results, nil
}
