#!/usr/bin/env bash

PROTOC_COMP=./protoc/bin/protoc
PROTOC_VERSION="libprotoc 3.17.1"

if [[ ! -f "$PROTOC_COMP" ]]; then
  printf "no protoc compiler found, run install.sh first\n"
  exit 1
fi

if [[ $($PROTOC_COMP --version) != $PROTOC_VERSION ]]; then
  printf "incorrect protoc version, required $PROTOC_VERSION\nrun install.sh\n"
  exit 1
fi

if [[ "$@" == "" ]]; then
  printf "Script requires a target name to build. Available targets: core\n"
  exit 1
fi

for project in "$@"; do
  if [[ $project == "core" ]]; then
    rm -rf ../evote/golosovaniepb
    mkdir -p ../evote/golosovaniepb
    $PROTOC_COMP -I=./ --go_opt=module=GO_LOSOVANIE --go_out=../ *.proto
  else
    printf "Unknown opt. Available: core\n"
    exit 1
  fi
done
