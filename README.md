# GO_LOSOVANIE

Подробная документация в папке `docs`

## Запуск
База данных завернута в Docker, сам сервер написан на Go.

### База данных
Перед запуском бд нужно [установить Docker](https://docs.docker.com/engine/install/ubuntu/).
Запуск для разработки (данные в бд будут удалены сразу после выключения):

```bash
    cd database/
    chmod +x ./start-dev.sh
    ./start-dev.sh
```

Запуск для прода (данные будут сохраняться в докер-контейнере)

```bash
    cd database/
    docker build -t db .
    docker run -p '5432:5432' db
```

Команда `docker run -p '5432:5432' db` указывает, на каком порту бд будет ожидать соединения.
Если, например, нужно сменить порт на `1337`, то команда приобретает
вид `docker run -p '1337:5432' db`

### Go
При написании использовался go1.14 Linux/amd64
Для запуска необходимо 
1. настроить $GOPATH, [инструкция на офф сайте](https://golang.org/doc/gopath_code.html). 
Если вкратце, то нужно, чтобы `GOPATH = ~/go`, проект лежал в `$GOPATH/src/GO_LOSOVANIE`,
а криптолибы (см. пункт 2), лежали в `$GOPATH/src/go.cypherpunks.ru`.
2. Поставить библиотеку [gogost](http://www.gogost.cypherpunks.ru/Download.html#Download).
Из скачанного при установке архива нужно переместить папку `src/go.cypherpunks.ru` в `$GOPATH/src/go.cypherpunks.ru`.
3. Поставить либу для работы с бд [pq](go get github.com/lib/pq) командой `go get github.com/lib/pq`.
