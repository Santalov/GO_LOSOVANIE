package evote

func hostsExceptGiven(validators []*ValidatorNode, pkey [PKEY_SIZE]byte) []string {
	hosts := make([]string, 0)
	for _, validator := range validators {
		if validator.pkey != pkey {
			hosts = append(hosts, validator.addr)
		}
	}
	return hosts
}

func appendValidator(to, from []*ValidatorNode, node *ValidatorNode) []*ValidatorNode {
	var validators []*ValidatorNode
	if len(to) == 0 {
		validators = append(validators, node)
		return validators
	}
	j := 0
	for i := 0; i < len(from); i++ {
		if from[i] == node {
			validators = append(to[:j], node)
			validators = append(validators, to[j:]...)
			break
		}
		if to[j] == from[i] {
			j++
		}
	}
	return validators
}

func removeAddr(addrs []string, rmAddr string) []string {
	newAddrs := make([]string, 0)
	for _, a := range addrs {
		if a != rmAddr {
			newAddrs = append(newAddrs, a)
		}
	}
	return newAddrs
}

func removePkey(validators []*ValidatorNode, pkey [PKEY_SIZE]byte) []*ValidatorNode {
	newValids := make([]*ValidatorNode, 0)
	for _, v := range validators {
		if v.pkey != pkey {
			newValids = append(newValids, v)
		}
	}
	return newValids
}

func containsTransInBlock(b *Block, hash [HASH_SIZE]byte) bool {
	for _, val := range b.trans {
		if hash == val.hash {
			return true
		}
	}
	return false
}
