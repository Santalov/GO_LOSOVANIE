package evote

import (
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
)

type ValidatorJson struct {
	GolosovaniePkey   string `json:"golosovanie_pkey"`
	TendermintAddress string `json:"tendermint_address"`
	IpAndPort         string `json:"ip_and_port"`
}

type ValidatorsSetJson struct {
	Validators []*ValidatorJson `json:"validators"`
}

type PrivateKeyJson struct {
	Pkey string `json:"pkey"`
	Prv  string `json:"prv"`
}

func LoadValidators(pathToValidatorsKeys string) ([]*ValidatorNode, error) {
	data, err := ioutil.ReadFile(pathToValidatorsKeys)
	if err != nil {
		return nil, err
	}
	var validatorsRaw ValidatorsSetJson
	err = json.Unmarshal(data, &validatorsRaw)
	if err != nil {
		return nil, err
	}
	validators := make([]*ValidatorNode, 0)
	for _, v := range validatorsRaw.Validators {
		addrSlice, err := hex.DecodeString(v.TendermintAddress)
		if err != nil {
			return nil, err
		}
		pkeySlice, err := hex.DecodeString(v.GolosovaniePkey)
		if err != nil {
			return nil, err
		}
		var addr [TM_ADDR_SIZE]byte
		copy(addr[:], addrSlice)
		var pkey [PKEY_SIZE]byte
		copy(pkey[:], pkeySlice)
		node := &ValidatorNode{
			Pkey:           pkey,
			IpAndPort:      v.IpAndPort,
			TendermintAddr: addr,
		}
		validators = append(validators, node)
	}
	return validators, nil
}

func LoadPrivateKey(pathToPrivateKey string) ([]byte, error) {
	data, err := ioutil.ReadFile(pathToPrivateKey)
	if err != nil {
		return nil, err
	}
	var keysRaw PrivateKeyJson
	err = json.Unmarshal(data, &keysRaw)
	if err != nil {
		return nil, err
	}
	prv, err := hex.DecodeString(keysRaw.Prv)
	if err != nil {
		return nil, err
	}
	return prv, nil
}
