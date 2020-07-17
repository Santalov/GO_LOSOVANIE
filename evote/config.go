package evote

import (
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"time"
)

type globalConfigRaw struct {
	Validators []struct {
		Pkey string `json:"pkey"`
		Addr string `json:"addr"`
	} `json:"validators"`
	NextLeaderPeriod time.Duration `json:"next_leader_period"`
	BlockAppendTime  time.Duration `json:"block_append_time"`
}

type localConfigRaw struct {
	Pkey string `json:"pkey"`
	Prv  string `json:"prv"`
	Addr string `json:"addr"`
}

type GlobalConfig struct {
	Validators       []*ValidatorNode
	NextLeaderPeriod time.Duration
	BlockAppendTime  time.Duration
}

type LocalConfig struct {
	Pkey [PKEY_SIZE]byte
	Prv  []byte
	Addr string
}

func LoadConfig(pathToGlobalConfig, pathToLocalConfig string) (*GlobalConfig, *LocalConfig, error) {
	global, err := ioutil.ReadFile(pathToGlobalConfig)
	if err != nil {
		return nil, nil, err
	}
	local, err := ioutil.ReadFile(pathToLocalConfig)
	if err != nil {
		return nil, nil, err
	}
	var gConfRaw globalConfigRaw
	var lConfRaw localConfigRaw
	err = json.Unmarshal(global, &gConfRaw)
	if err != nil {
		return nil, nil, err
	}
	err = json.Unmarshal(local, &lConfRaw)
	if err != nil {
		return nil, nil, err
	}
	var gConf GlobalConfig
	var lConf LocalConfig
	for _, validatorRaw := range gConfRaw.Validators {
		pkey, err := hex.DecodeString(validatorRaw.Pkey)
		if err != nil {
			return nil, nil, err
		}
		validator := &ValidatorNode{}
		copy(validator.pkey[:], pkey)
		validator.addr = validatorRaw.Addr
		gConf.Validators = append(gConf.Validators, validator)
		gConf.NextLeaderPeriod = gConfRaw.NextLeaderPeriod
		gConf.BlockAppendTime = gConfRaw.BlockAppendTime
	}
	myPkey, err := hex.DecodeString(lConfRaw.Pkey)
	if err != nil {
		return nil, nil, err
	}
	myPrv, err := hex.DecodeString(lConfRaw.Prv)
	if err != nil {
		return nil, nil, err
	}
	copy(lConf.Pkey[:], myPkey)
	lConf.Prv = myPrv
	lConf.Addr = lConfRaw.Addr
	return &gConf, &lConf, nil
}
