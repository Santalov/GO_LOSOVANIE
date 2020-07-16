package evote

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

const (
	CHAN_SIZE = 1000
)

// сообщение, которое будет передаваться Blockchain из серверной части сети
// нужно, чтобы передавать бинарники с указанием на того, кто их отправил, и с каналом для ответа
type NetworkMsg struct {
	data     []byte
	from     string           // адресс отправителя в формате 1.3.3.7:1488
	response chan ResponseMsg // отсюда сервак ожидает получить результат проверки сообщения
}

type NetworkChannels struct {
	// Из этих каналов сообщения будет забирать Blockchain
	// класть в них сообщения будет сервер
	blocks            chan NetworkMsg
	txsValidator      chan NetworkMsg // транзы от других валидаторов
	txsClient         chan NetworkMsg // транзы от клиентов
	blockVotes        chan NetworkMsg
	kickValidatorVote chan NetworkMsg
}

// этими сообщениями Blockchain сообщает результаты проверки
type ResponseMsg struct {
	ok    bool
	error string
}

type Network struct {
	chs NetworkChannels
}

func (n *Network) Init() *NetworkChannels {

	n.chs = NetworkChannels{
		make(chan NetworkMsg, CHAN_SIZE),
		make(chan NetworkMsg, CHAN_SIZE),
		make(chan NetworkMsg, CHAN_SIZE),
		make(chan NetworkMsg, CHAN_SIZE),
		make(chan NetworkMsg, CHAN_SIZE),
	}
	return &n.chs
}

// Serve запускает сервер и блокирует поток, перед вызовом Serve надо вызвать Init
func (n *Network) Serve() {
	err := http.ListenAndServe(":"+strconv.Itoa(PORT), nil)
	panic(err)
}

var successResp = []byte("{\"success\":true}")

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

func (n *Network) handleGetTx(w http.ResponseWriter, req *http.Request) {
	http.Error(w, "service unavailable", http.StatusServiceUnavailable)
}

func (n *Network) handleGetTxsByPubKey(w http.ResponseWriter, req *http.Request) {
	http.Error(w, "service unavailable", http.StatusServiceUnavailable)
}

func (n *Network) handleGetUTXOByPubKey(w http.ResponseWriter, req *http.Request) {
	http.Error(w, "service unavailable", http.StatusServiceUnavailable)
}

func (n *Network) handleSubmitClientTx(w http.ResponseWriter, req *http.Request) {
	rawData, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "cannot read the body: "+err.Error(), http.StatusBadRequest)
		return
	}
	type clientSubmitTx struct {
		Tx string `json:"tx"`
	}
	var parsedData clientSubmitTx
	err = json.Unmarshal(rawData, &parsedData)
	if err != nil {
		http.Error(w, "transaction required as tx field in json in body, example: {\"tx\":\"1bf12...\"}", http.StatusBadRequest)
		return
	}
	tx, err := hex.DecodeString(parsedData.Tx)
	if err != nil {
		http.Error(w, "incorrect hex in tx: "+err.Error(), http.StatusBadRequest)
	}
	ch := make(chan ResponseMsg)
	n.chs.txsClient <- NetworkMsg{
		data:     tx,
		from:     req.Host,
		response: ch,
	}
	resp := <-ch
	if !resp.ok {
		http.Error(w, resp.error, http.StatusBadRequest)
	} else {
		_, err := fmt.Fprint(w, successResp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		}
	}
}

// обработчики запросов от других валидаторов
// отличаются тем, что работают с бинарями, а так же логикой

func (n *Network) handleSubmitServerTx(w http.ResponseWriter, req *http.Request) {

}

func (n *Network) handleSubmitBlock(w http.ResponseWriter, req *http.Request) {

}

func (n *Network) handleBlockVote(w http.ResponseWriter, req *http.Request) {

}

func (n *Network) handleKickValidatorVote(w http.ResponseWriter, req *http.Request) {

}
