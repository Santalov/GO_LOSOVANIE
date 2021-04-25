module GO_LOSOVANIE

require (
	github.com/lib/pq v1.10.2
	github.com/manifoldco/promptui v0.8.0
	github.com/stretchr/testify v1.7.0
	github.com/tendermint/tendermint v0.34.10
	go.cypherpunks.ru/gogost/v5 v5.6.0
)

replace go.cypherpunks.ru/gogost/v5 => /home/stas/go/src/gogost-5.6.0

go 1.16
