# Структура транзакции и блока

## Константы
SIG_SIZE = 64
PKEY_SIZE = 33
HASH_SIZE  = 32

## Транзакция

```javascript
type TransactionInput struct {
	prevId [HASH_SIZE]byte
	outIndex uint32
}

type TransactionOutput struct {
	value uint32
	pkeyTo [PKEY_SIZE]byte
}

type Transaction struct {
	inputSize uint32
	inputs []TransactionInput
	outputSize uint32
	outputs []TransactionOutput
	typeValue [HASH_SIZE]byte
	typeVote uint32
	duration uint32
	hashLink [HASH_SIZE]byte
	signature [SIG_SIZE]byte
}

соглашения на счет output_pkey:
все pkey везде, т.е. В КОДЕ ТОЖЕ, храняться в виде байтовой строки
первый байт равен 0x03 или 0x02, другие 32 байта равны байтовой строке
ключа pkeyX

hashLink используется в создании голсования, в случае
если число голосующих не может помеситься в одну транзу
в hashLink транзы n содержиться хэш транзакции n+1
корневой хэшлинк равен нулю

typeValue при создании транзакции создании голосования:
1) Все typeValue в оutput == null byte array
2) typeValue в последующих транзакциях = hash(root Transaction)
3) Проверка на корректность:
	Если inputs транзакции имеет специальный тип создания
	голосования(опредлеяется по спец. кошельку в output),
	то typeValue должно быть равно prevId.
	Иначе typeValue текущей транзакции должно быть равно
	typeValue input транзакции.

Transaction, Input, Output - структуры, которые сериализуются в байтовоую строку.
Порядок записи полей определен выше.
Проверка транзы:
1) Проверка корректной запись полей
2) Проверка для каждого input на то, что транзакция с хэшом input.prevId содержится
   в блокчейне, параметр outIndex верный.
3) Проверка для каждого input на то, что значение value в output'ах транзы input.prevId на кошелек
   Transaction.pubKey не меньше, чем значение value в output данной транзы.
4) Проверка на то, что данная транза подписана правильно. Хэш для подписи текущей транзы считаеться как хэш
от всех полей транзы, но Transaction.signature равна нулевой строке байт, т.е. 0x00^64

При создании транзы создания голосования output с специальным pkey кладется в outputs []TransactionOutput с индексом 0
```

## Блок
```javascript
TransInBlock struct {
	hash []byte // 32 bytes
	transaction *Transaction
}

Block struct {
	prevBlockHash [HASH_SIZE]byte
	merkleTree [HASH_SIZE]byte
	timestamp uint64
	transSize uint32
	trans []TransInBlock
}


При создании блока награда за майниг кладется в транзу с индексом 0
```
