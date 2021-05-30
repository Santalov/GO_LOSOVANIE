rm -rf ./protoc &&
mkdir -p ./protoc &&
cd ./protoc &&
wget https://github.com/protocolbuffers/protobuf/releases/download/v3.17.1/protoc-3.17.1-linux-x86_64.zip &&
unzip protoc-3.17.1-linux-x86_64.zip &&
rm protoc-3.17.1-linux-x86_64.zip &&
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.26.0