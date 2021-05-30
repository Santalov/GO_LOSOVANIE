package evote

import (
	"GO_LOSOVANIE/evote/golosovaniepb"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/proto"
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

func (n *Network) abciQueryValueProto(path string, req *golosovaniepb.Request) (*golosovaniepb.Response, error) {
	reqBytes, err := proto.Marshal(req)
	if err != nil {
		return nil, err
	}
	data, err := n.abciQueryValue(path, reqBytes)
	if err != nil {
		return nil, err
	}
	var resp golosovaniepb.Response
	err = proto.Unmarshal(data, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
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

func (n *Network) GetTxsByHashes(hashes [][]byte) ([]*golosovaniepb.Transaction, error) {
	req := golosovaniepb.Request{
		Data: &golosovaniepb.Request_TxsByHashes{
			TxsByHashes: &golosovaniepb.RequestTxsByHashes{
				Hashes: hashes,
			},
		},
	}
	resp, err := n.abciQueryValueProto("getTxs", &req)
	if err != nil {
		return nil, err
	}
	return resp.GetTxsByHashes().GetTxs(), nil
}

func (n *Network) GetTxsByPkey(pkey []byte) ([]*golosovaniepb.Transaction, error) {
	req := golosovaniepb.Request{
		Data: &golosovaniepb.Request_TxsByPkey{
			TxsByPkey: &golosovaniepb.RequestTxsByPkey{
				Pkey: pkey,
			},
		},
	}
	resp, err := n.abciQueryValueProto("getTxsByPubKey", &req)
	if err != nil {
		return nil, err
	}
	return resp.GetTxsByPkey().GetTxs(), nil
}

func (n *Network) GetUtxosByPkey(pkey []byte) ([]*golosovaniepb.Utxo, error) {
	req := golosovaniepb.Request{
		Data: &golosovaniepb.Request_UtxosByPkey{
			UtxosByPkey: &golosovaniepb.RequestUtxosByPkey{
				Pkey: pkey,
			},
		},
	}
	resp, err := n.abciQueryValueProto("getUtxosByPubKey", &req)
	if err != nil {
		return nil, err
	}
	return resp.GetUtxosByPkey().GetUtxos(), nil
}

func (n *Network) SubmitTx(tx []byte) error {
	_, err := n.BroadcastTxSync(tx)
	return err
}

func (n *Network) Faucet(amount uint32, pkey []byte) error {
	req := golosovaniepb.Request{
		Data: &golosovaniepb.Request_Faucet{
			Faucet: &golosovaniepb.RequestFaucet{
				Pkey:  pkey,
				Value: amount,
			},
		},
	}
	_, err := n.abciQueryValueProto("faucet", &req)
	return err
}

func (n *Network) VoteResults(hash []byte) (map[[PkeySize]byte]uint32, error) {
	req := golosovaniepb.Request{
		Data: &golosovaniepb.Request_VoteResult{
			VoteResult: &golosovaniepb.RequestVoteResult{
				VoteTxHash: hash,
			},
		},
	}
	resp, err := n.abciQueryValueProto("getVoteResult", &req)
	if err != nil {
		return nil, err
	}
	results := make(map[[PkeySize]byte]uint32)
	for _, v := range resp.GetVoteResult().GetRes() {
		pkey := SliceToPkey(v.Pkey)
		results[pkey] = v.Value
	}
	return results, nil
}
