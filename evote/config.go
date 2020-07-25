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
	BlockAppendTime time.Duration `json:"block_append_time"`
	BlockVotingTime time.Duration `json:"block_voting_time"`
	JustWaitingTime time.Duration `json:"just_waiting_time"`
}

type localConfigRaw struct {
	Pkey string `json:"pkey"`
	Prv  string `json:"prv"`
	Addr string `json:"addr"`
}

type GlobalConfig struct {
	Validators      []*ValidatorNode
	BlockAppendTime time.Duration
	BlockVotingTime time.Duration
	JustWaitingTime time.Duration
}

type LocalConfig struct {
	Pkey [PKEY_SIZE]byte
	Prv  []byte
	Addr string
}

func LoadConfig(pathToGlobalConfig, pathToLocalConfig string) (*GlobalConfig, *LocalConfig, error) {
	gConf, err1 := LoadGlobalConfig(pathToGlobalConfig)
	lConf, err2 := LoadLocalConfig(pathToLocalConfig)
	if err1 != nil {
		return nil, nil, err1
	}
	if err2 != nil {
		return nil, nil, err2
	}
	return gConf, lConf, nil
}

func LoadLocalConfig(pathToLocalConfig string) (*LocalConfig, error) {
	local, err := ioutil.ReadFile(pathToLocalConfig)
	if err != nil {
		return nil, err
	}
	var lConfRaw localConfigRaw
	err = json.Unmarshal(local, &lConfRaw)
	if err != nil {
		return nil, err
	}
	var lConf LocalConfig
	myPkey, err := hex.DecodeString(lConfRaw.Pkey)
	if err != nil {
		return nil, err
	}
	myPrv, err := hex.DecodeString(lConfRaw.Prv)
	if err != nil {
		return nil, err
	}
	copy(lConf.Pkey[:], myPkey)
	lConf.Prv = myPrv
	lConf.Addr = lConfRaw.Addr
	return &lConf, nil
}

func LoadGlobalConfig(pathToGlobalConfig string) (*GlobalConfig, error) {
	global, err := ioutil.ReadFile(pathToGlobalConfig)
	if err != nil {
		return nil, err
	}
	var gConfRaw globalConfigRaw
	err = json.Unmarshal(global, &gConfRaw)
	if err != nil {
		return nil, err
	}
	var gConf GlobalConfig
	for _, validatorRaw := range gConfRaw.Validators {
		pkey, err := hex.DecodeString(validatorRaw.Pkey)
		if err != nil {
			return nil, err
		}
		validator := &ValidatorNode{}
		copy(validator.pkey[:], pkey)
		validator.addr = validatorRaw.Addr
		gConf.Validators = append(gConf.Validators, validator)
		gConf.BlockAppendTime = gConfRaw.BlockAppendTime
		gConf.BlockVotingTime = gConfRaw.BlockVotingTime
		gConf.JustWaitingTime = gConfRaw.JustWaitingTime
	}
	return &gConf, nil
}
