package evote

import (
	"net/http"
	"strconv"
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
// может быть не string потом
type ResponseMsg string

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

// обработчик запроса для тестов

func handleInfo(w http.ResponseWriter, req *http.Request) {

}

// обработчики для запросов от клиентов

func handleGetTx(w http.ResponseWriter, req *http.Request) {

}

func handleGetTxsByPubKey(w http.ResponseWriter, req *http.Request) {

}

func handleGetUTXOByPubKey(w http.ResponseWriter, req *http.Request) {

}

func handleSubmitClientTx(w http.ResponseWriter, req *http.Request) {

}

// обработчики запросов от других валидаторов
// отличаются тем, что работают с бинарями, а так же логикой

func handleSubmitServerTx(w http.ResponseWriter, req *http.Request) {

}

func handleSubmitBlock(w http.ResponseWriter, req *http.Request) {

}

func handleBlockVote(w http.ResponseWriter, req *http.Request) {

}

func handleKickValidatorVote(w http.ResponseWriter, req *http.Request) {

}
