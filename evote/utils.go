package evote

func hostsExceptGiven(validators []*ValidatorNode, pkey [PKEY_SIZE]byte) []string {
	hosts := make([]string, 0)
	for _, validator := range validators {
		if validator.Pkey != pkey {
			hosts = append(hosts, validator.Addr)
		}
	}
	return hosts
}

func remakeActiveHostsExceptMe(activeHostsExceptMe []string, activeValidators []*ValidatorNode, thisValidator *ValidatorNode) []string {
	activeHostsMap := make(map[string]bool)
	for _, host := range activeHostsExceptMe {
		activeHostsMap[host] = true
	}
	for _, validator := range activeValidators {
		activeHostsMap[validator.Addr] = true
	}
	activeHostsMap[thisValidator.Addr] = false
	activeHostsExceptMe = make([]string, 0)
	for addr, flag := range activeHostsMap {
		if flag {
			activeHostsExceptMe = append(activeHostsExceptMe, addr)
		}
	}
	return activeHostsExceptMe
}

func appendValidator(to, from []*ValidatorNode, node *ValidatorNode) []*ValidatorNode {
	toMap := make(map[*ValidatorNode]bool)
	for _, validator := range to {
		toMap[validator] = true
	}
	toMap[node] = true
	validators := make([]*ValidatorNode, 0)
	for _, validator := range from {
		if toMap[validator] {
			validators = append(validators, validator)
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
		if v.Pkey != pkey {
			newValids = append(newValids, v)
		}
	}
	return newValids
}

func containsTransInBlock(b *Block, hash [HASH_SIZE]byte) bool {
	for _, val := range b.Trans {
		if hash == val.Hash {
			return true
		}
	}
	return false
}

func getVoteValue(value, typeVote uint32) int32 {
	if typeVote == ONE_VOTE_TYPE {
		return 1
	}
	if typeVote == PERCENT_VOTE_TYPE {
		return int32(value)
	}
	return int32(value)
}
