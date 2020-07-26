package evote

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	chanSize = 1000
)

// сообщение, которое будет передаваться Blockchain из серверной части сети
// нужно, чтобы передавать бинарники с указанием на того, кто их отправил, и с каналом для ответа
type NetworkMsg struct {
	data     []byte
	from     string           // адресс отправителя в формате 1.3.3.7:1488
	response chan ResponseMsg // отсюда сервак ожидает получить результат проверки сообщения
}

type NetworkByteMsg struct {
	data     []byte
	from     string            // адресс отправителя в формате 1.3.3.7:1488
	response chan ByteResponse // отсюда сервак ожидает получить результат проверки сообщения
}

type NetworkChannels struct {
	// Из этих каналов сообщения будет забирать Blockchain
	// класть в них сообщения будет сервер
	blocks              chan NetworkMsg
	txsValidator        chan NetworkMsg // транзы от других валидаторов
	txsClient           chan NetworkMsg // транзы от клиентов
	blockVotes          chan NetworkMsg
	kickValidatorVote   chan NetworkMsg
	appendValidatorVote chan NetworkMsg
	faucet			    chan NetworkMsg
	appendViewer        chan NetworkByteMsg
	blockAfter          chan NetworkByteMsg
	getUtxosByPkey      chan NetworkByteMsg
	getTxsByPkey        chan NetworkByteMsg
	getTxsByHashes      chan NetworkByteMsg
}

// этими сообщениями Blockchain сообщает результаты проверки
type ResponseMsg struct {
	ok    bool
	error string
}

type ByteResponse struct {
	ok    bool
	data  []byte
	error string
}

type Network struct {
	chs NetworkChannels
	// здесь мб еще поля будут
}

func (n *Network) Init() *NetworkChannels {

	// определение обработчиков запросов
	http.Handle("/info", http.HandlerFunc(n.handleInfo))
	http.Handle("/getTxs", http.HandlerFunc(n.handleGetTxs))
	http.Handle("/getTxsByPubKey", http.HandlerFunc(n.handleGetTxsByPubKey))
	http.Handle("/getUTXOByPubKey", http.HandlerFunc(n.handleGetUTXOByPubKey))
	http.Handle("/submitClientTx", http.HandlerFunc(n.handleSubmitClientTx))
	http.Handle("/submitValidatorTx", http.HandlerFunc(n.handleSubmitValidatorTx))
	http.Handle("/submitBlock", http.HandlerFunc(n.handleSubmitBlock))
	http.Handle("/blockVote", http.HandlerFunc(n.handleBlockVote))
	http.Handle("/kickValidatorVote", http.HandlerFunc(n.handleKickValidatorVote))
	http.Handle("/appendViewer", http.HandlerFunc(n.handleAppendViewer))
	http.Handle("/blockAfter", http.HandlerFunc(n.handleBlockAfter))
	http.Handle("/voteAppendValidator", http.HandlerFunc(n.handleAppendValidatorVote))
	http.Handle("/faucet", http.HandlerFunc(n.handleFaucet))


	n.chs = NetworkChannels{
		make(chan NetworkMsg, chanSize),
		make(chan NetworkMsg, chanSize),
		make(chan NetworkMsg, chanSize),
		make(chan NetworkMsg, chanSize),
		make(chan NetworkMsg, chanSize),
		make(chan NetworkMsg, chanSize),
		make(chan NetworkMsg, chanSize),
		make(chan NetworkByteMsg, chanSize),
		make(chan NetworkByteMsg, chanSize),
		make(chan NetworkByteMsg, chanSize),
		make(chan NetworkByteMsg, chanSize),
		make(chan NetworkByteMsg, chanSize),

	}
	return &n.chs
}

// Serve запускает сервер и блокирует поток, перед вызовом Serve надо вызвать Init
func (n *Network) Serve(addr string) {
	fmt.Println("Starting server at ", addr)
	err := http.ListenAndServe(addr, nil)
	panic(err)
}

func sendBinary(url string, data []byte, ch chan *http.Response) {
	resp, err := http.Post(url, "application/octet-stream", bytes.NewReader(data))
	if err != nil {
		fmt.Printf("network err: %v\n", err)
	} else {
		if resp.StatusCode != http.StatusOK {
			fmt.Println()
			body, _ := ioutil.ReadAll(resp.Body)
			fmt.Printf("network: server answered with error %v, body: %v\n", resp.Status, string(body))
		}
	}
	ch <- resp
}

func (n *Network) sendBinaryToAll(hosts []string, data []byte, endpoint string) {
	responses := make(chan *http.Response, len(hosts))
	for _, host := range hosts {
		go sendBinary("http://"+host+endpoint, data, responses)
	}
	for range hosts {
		<-responses
	}
}

func (n *Network) SendBlockToAll(hosts []string, data []byte) {
	n.sendBinaryToAll(hosts, data, "/submitBlock")
}

func (n *Network) SendTxToAll(hosts []string, data []byte) {
	n.sendBinaryToAll(hosts, data, "/submitValidatorTx")
}

func (n *Network) SendVoteToAll(hosts []string, data []byte) {
	n.sendBinaryToAll(hosts, data, "/blockVote")
}

func (n *Network) SendVoteAppendValidatorMsgToAll(hosts []string, data []byte) {
	n.sendBinaryToAll(hosts, data, "/voteAppendValidator")
}

func (n *Network) SendKickMsgToAll(hosts []string, data []byte) {
	n.sendBinaryToAll(hosts, data, "/kickValidatorVote")
}

func makeInfoRequest(host string, response chan string) {
	resp, err := http.Get("http://" + host + "/info")
	if err != nil {
		fmt.Println("network err: ", err)
		response <- ""
	} else {
		if resp.StatusCode != http.StatusOK {
			fmt.Println()
			body, _ := ioutil.ReadAll(resp.Body)
			fmt.Printf("network: server answered with error %v, body: %v\n", resp.Status, string(body))
			response <- ""
		} else {
			response <- host
		}
	}
}

// вернет массив адресов живых хостов
func (n *Network) PingHosts(hosts []string) (alive []string) {
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

// отправит запрос на /blockAfter
func (n *Network) GetBlockAfter(host string, hash [HASH_SIZE]byte) (block []byte, err error) {
	resp, err := http.Post("http://"+host+"/blockAfter", "application/octet-stream", bytes.NewReader(hash[:]))
	if err != nil {
		fmt.Printf("network err: %v\n", err)
		return nil, err
	} else {
		body, _ := ioutil.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			fmt.Printf("network: server answered with error %v, body: %v\n", resp.Status, string(body))
			return nil, fmt.Errorf(string(body))
		} else {
			block = body
			return block, nil
		}
	}
}

func (n *Network) SendAppendViewerMsg(host string, data []byte) (pkey []byte, err error) {
	resp, err := http.Post("http://"+host+"/appendViewer", "application/octet-stream", bytes.NewReader(data))
	if err != nil {
		fmt.Printf("network err: %v\n", err)
		return nil, err
	} else {
		body, _ := ioutil.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			fmt.Printf("network: server answered with error %v, body: %v\n", resp.Status, string(body))
			return nil, fmt.Errorf(string(body))
		} else {
			pkey = body
			return pkey, nil
		}
	}
}

var successResp = []byte("{\"success\":true}")

// вспомогательные функции

func response(w http.ResponseWriter, resp ResponseMsg) {
	if !resp.ok {
		http.Error(w, resp.error, http.StatusBadRequest)
	} else {
		_, err := fmt.Fprint(w, successResp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		}
	}
}

// обработчики для запросов от клиентов

func (n *Network) handleInfo(w http.ResponseWriter, _ *http.Request) {
	infoMsg := struct {
		Time time.Time `json:"time"`
	}{}
	msg, err := json.Marshal(&infoMsg)
	if err != nil {
		http.Error(w, "server error"+err.Error(), http.StatusServiceUnavailable)
		return
	}
	_, err = fmt.Fprint(w, msg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
}

func handleBinary(msgChan chan NetworkMsg, w http.ResponseWriter, req *http.Request) {
	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}
	ch := make(chan ResponseMsg)
	msgChan <- NetworkMsg{
		data:     data,
		from:     req.Host,
		response: ch,
	}
	resp := <-ch
	response(w, resp)
}

func handleBinaryWithResponse(msgChan chan NetworkByteMsg, w http.ResponseWriter, req *http.Request) {
	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}
	ch := make(chan ByteResponse)
	msgChan <- NetworkByteMsg{
		data:     data,
		from:     req.Host,
		response: ch,
	}
	resp := <-ch
	if !resp.ok {
		http.Error(w, resp.error, http.StatusBadRequest)
	} else {
		_, err := w.Write(resp.data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		}
	}
}

func (n *Network) handleGetTxs(w http.ResponseWriter, req *http.Request) {
	handleBinaryWithResponse(n.chs.getTxsByHashes, w, req)
}

func (n *Network) handleGetTxsByPubKey(w http.ResponseWriter, req *http.Request) {
	handleBinaryWithResponse(n.chs.getTxsByPkey, w, req)
}

func (n *Network) handleGetUTXOByPubKey(w http.ResponseWriter, req *http.Request) {
	handleBinaryWithResponse(n.chs.getUtxosByPkey, w, req)
}

func (n *Network) handleSubmitClientTx(w http.ResponseWriter, req *http.Request) {
	handleBinary(n.chs.txsClient, w, req)
}

// обработчики запросов от других валидаторов
// отличаются тем, что работают с бинарями, а так же логикой

func (n *Network) handleSubmitValidatorTx(w http.ResponseWriter, req *http.Request) {
	handleBinary(n.chs.txsValidator, w, req)
}

func (n *Network) handleSubmitBlock(w http.ResponseWriter, req *http.Request) {
	handleBinary(n.chs.blocks, w, req)
}

func (n *Network) handleFaucet(w http.ResponseWriter, req *http.Request) {
	handleBinary(n.chs.faucet, w, req)
}

func (n *Network) handleBlockVote(w http.ResponseWriter, req *http.Request) {
	handleBinary(n.chs.blockVotes, w, req)
}

func (n *Network) handleAppendValidatorVote(w http.ResponseWriter, req *http.Request) {
	handleBinary(n.chs.appendValidatorVote, w, req)
}

func (n *Network) handleKickValidatorVote(w http.ResponseWriter, req *http.Request) {
	handleBinary(n.chs.kickValidatorVote, w, req)
}

func (n *Network) handleAppendViewer(w http.ResponseWriter, req *http.Request) {
	handleBinaryWithResponse(n.chs.appendViewer, w, req)
}

func (n *Network) handleBlockAfter(w http.ResponseWriter, req *http.Request) {
	handleBinaryWithResponse(n.chs.blockAfter, w, req)
}
