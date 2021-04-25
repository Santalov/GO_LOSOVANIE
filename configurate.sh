#!/usr/bin/env bash

# Script will create local GO_LOSOVANIE network configuration for four validators
# this script assumes, that you have tendermint installed
# tendermint p2p ports are configured to be 26500, 26501, 26502, 26503
# tendermint rpc ports are configured to be 26600, 26601, ...
# GO_LOSOVANIE database prots are configured to be 54300, 54301, ...
# GO_LOSOVANIE ABCI socket addresses are configured to be unix://golosovanie00.sock, unix://golosovanie01.sock ...
rm -rf ./config &&
  tendermint testnet &&
  mv ./mytestnet ./config &&
  for num in 0 1 2 3; do
    tendermint show_node_id --home ./config/node$num | tr -d "\n\r" >>./config/persistent_peers &&
      TM_HOST="localhost:2750"$num &&
      RPC_HOST="localhost:2760"$num &&
      SOCK_ADDR="golosovanie0"$num".sock" &&
      ABCI_ADDR="unix://"$SOCK_ADDR &&
      printf "@"$TM_HOST >>./config/persistent_peers &&
      if [[ $num -lt 3 ]]; then
        printf "," >>./config/persistent_peers
      fi
    # file with chain private key
    go run utils/gen_key_pair/main.go >>config/node$num/golosovanie_private_key.json &&
      # temporary file, used by util, that creates validatorsKeys config
      printf $RPC_HOST >>./config/node$num/ip_and_port &&
      printf " cd ../database &&\n docker build -t db .\n docker run -p '5430"$num":5432' --rm db" >>./config/db$num.sh &&
      chmod +x ./config/db$num.sh &&
      printf " rm -f $SOCK_ADDR &&\n go run GO_LOSOVANIE -v validators.json -k node$num/golosovanie_private_key.json -p 5430$num -s "$ABCI_ADDR >>./config/val$num.sh &&
      chmod +x ./config/val$num.sh
  done &&
  PERSISTENT_PEERS=$(cat ./config/persistent_peers) &&
  for num in 0 1 2 3; do
    TM_HOST="localhost:2750"$num &&
      RPC_HOST="localhost:2760"$num &&
      SOCK_ADDR="golosovanie0"$num".sock" &&
      ABCI_ADDR="unix://"$SOCK_ADDR &&
      printf " tendermint unsafe_reset_all --home ./node$num &&\n tendermint node --home ./node$num --proxy_app=$ABCI_ADDR --rpc.laddr=tcp://$RPC_HOST --p2p.laddr=tcp://$TM_HOST --p2p.persistent_peers=\"$PERSISTENT_PEERS\"" >>./config/tm$num.sh &&
      chmod +x ./config/tm$num.sh
  done &&
  rm ./config/persistent_peers &&
  go run utils/merge_validators_configs/main.go -t config/ >>config/validators.json
