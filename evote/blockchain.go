package evote

import "time"

/*
1) on block voting:
vote 0 == no vote
vote 1 == yes
vote 2 == no

2) on suspicious validators:
vote == count no vote/bad blocks
vote can be 0 or 1,
if (vote > 1) then process kick start

3) On kick voting:
0 <= vote <= len(validatePkeys)
*/

type ValidatorNode struct {
	pkey [PKEY_SIZE]byte
	addr string // адрес вида 1.1.1.1:1337
}

type Blockchain struct {
	thisKey              *CryptoKeysData
	validators           []*ValidatorNode
	prevBlockHash        [HASH_SIZE]byte
	prevBlockHashes      [MAX_PREV_BLOCK_HASHES][HASH_SIZE]byte
	currentLeader        [PKEY_SIZE]byte
	currentBock          *BlocAndkHash
	unrecordedTrans      []TransAndHash
	nextLeaderVoteTime   uint64
	nextLeaderPeriod     uint64
	blockAppendTime      uint64
	chainSize            uint64
	blockVoting          map[[PKEY_SIZE]byte]int
	kickVoting           map[[PKEY_SIZE]byte]int
	suspiciousValidators map[[PKEY_SIZE]byte]int
}

func (bc *Blockchain) Setup(thisPrv []byte, validators []*ValidatorNode,
	nextVoteTime uint64, nextPeriod uint64, appendTime uint64) {
	//зачатки констуруктора
	var k CryptoKeysData
	k.SetupKeys(thisPrv)
	bc.thisKey = &k

	bc.validators = validators

	bc.nextLeaderVoteTime = nextVoteTime
	bc.nextLeaderPeriod = nextPeriod
	bc.blockAppendTime = appendTime

	for _, validator := range bc.validators {
		bc.blockVoting[validator.pkey] = 0
		bc.kickVoting[validator.pkey] = 0
		bc.suspiciousValidators[validator.pkey] = 0
	}

	var blockInCons BlocAndkHash
	bc.currentBock = &blockInCons

	bc.chainSize = 0

}

func (bc *Blockchain) OnBlockRecive(data []byte, sender [PKEY_SIZE]byte) {
	if sender != bc.currentLeader {
		bc.suspiciousValidators[sender] = 1
		// блок пришел не от лидера
		return
	}
	var b Block
	hash, blockLen := b.Verify(data, bc.prevBlockHash, bc.currentLeader)
	if blockLen != len(data) {
		bc.suspiciousValidators[bc.currentLeader] = 1
		bc.blockVoting[bc.thisKey.pubKeyByte] = 2
		// надо в сеть послать инфу, что блок гавно
	}

	bc.nextLeaderVoteTime = b.timestamp + bc.nextLeaderPeriod
	copy(bc.currentBock.hash[:], hash)
	bc.currentBock.b = &b

	bc.blockVoting[bc.thisKey.pubKeyByte] = 1
	// надо в сеть послать инфу, что блок ok
}

func (bc *Blockchain) OnBlockVote(data []byte) {
	if len(data) != HASH_SIZE+PKEY_SIZE+1 {
		return
	}
	var hash [HASH_SIZE]byte
	var pkey [PKEY_SIZE]byte
	var vote [1]byte
	copy(hash[:], data[:HASH_SIZE])
	copy(pkey[:], data[HASH_SIZE:PKEY_SIZE])
	copy(vote[:], data[HASH_SIZE+PKEY_SIZE:])
	_, ok := bc.blockVoting[pkey]
	if hash == bc.prevBlockHash && ok {
		if vote[0] == 0x01 || vote[0] == 0x02 {
			bc.blockVoting[pkey] = int(vote[0])
		}
	}
}

func (bc *Blockchain) OnKickValidatorVote(data []byte) {
	if len(data) != PKEY_SIZE {
		return
	}
	var kickPkey [PKEY_SIZE]byte
	copy(kickPkey[:], data)
	if kickPkey == bc.thisKey.pubKeyByte {
		return
	}
	bc.kickVoting[kickPkey] += 1
}

func (bc *Blockchain) UpdatePrevHashBlock() {
	for i := MAX_PREV_BLOCK_HASHES - 1; i >= 1; i-- {
		bc.prevBlockHashes[i] = bc.prevBlockHashes[i-1]
	}
	bc.prevBlockHashes[0] = bc.currentBock.hash
}

func (bc *Blockchain) ContainsTransInBlock(hash [HASH_SIZE]byte) bool {
	for _, val := range bc.currentBock.b.trans {
		if hash == val.hash {
			return true
		}
	}
	return false
}

func (bc *Blockchain) UpdateUnrecordedTrans() {
	var newUnrecorded []TransAndHash
	for _, t := range bc.unrecordedTrans {
		if !bc.ContainsTransInBlock(t.hash) {
			newUnrecorded = append(newUnrecorded, t)
		}
	}
}

func (bc *Blockchain) ProcessKick() {
	for k, v := range bc.kickVoting {
		if float32(v)/float32(len(bc.validators)) > 0.5 {
			for i, validator := range bc.validators {
				if validator.pkey == k {
					bc.validators = append(bc.validators[:i], bc.validators[i+1:]...)
					break
				}
			}
		}
	}
	bc.kickVoting = make(map[[PKEY_SIZE]byte]int, 0)
	for _, validator := range bc.validators {
		bc.kickVoting[validator.pkey] = 0
	}
}

func (bc *Blockchain) ClearBlockVoting() {
	bc.blockVoting = make(map[[PKEY_SIZE]byte]int, 0)
	for _, validator := range bc.validators {
		bc.blockVoting[validator.pkey] = 0
	}
}

func (bc *Blockchain) DoTick() {
	bc.ProcessKick()

	bc.currentLeader = bc.validators[bc.chainSize%uint64(len(bc.validators))].pkey
	bc.nextLeaderVoteTime = uint64(time.Now().UnixNano()) + bc.nextLeaderPeriod

	bc.ClearBlockVoting()
	bc.OnThisCreateBlock()

	//ждем

	yesVote, noVote := 0, 0
	for pkey, vote := range bc.blockVoting {
		if vote == 0 {
			bc.suspiciousValidators[pkey] += 1
			if bc.suspiciousValidators[pkey] > 1 {
				//vote kick
			}
			noVote += 1
		} else if vote == 1 {
			yesVote += 1
			bc.suspiciousValidators[pkey] = 0
		} else {
			noVote += 1
		}
	}

	if noVote < yesVote {
		bc.UpdatePrevHashBlock()
		//запись блока в БД
		bc.UpdateUnrecordedTrans()
		bc.chainSize += 1
	} else if bc.currentLeader != bc.thisKey.pubKeyByte {
		bc.suspiciousValidators[bc.currentLeader] += 1
		if bc.suspiciousValidators[bc.currentLeader] > 1 {
			//vote kick
		}
	}

}

func (bc *Blockchain) OnThisCreateBlock() {
	var b Block
	b.CreateBlock(bc.unrecordedTrans[:MAX_TRANS_SIZE], bc.prevBlockHash, bc.thisKey)
	blockBytes := b.ToBytes()
	hash := b.HashBlock(blockBytes)
	copy(bc.currentBock.hash[:], hash)
	bc.currentBock.b = &b

	// послать в сеть блок
}

func (bc *Blockchain) VoteKickValidator(pkey [PKEY_SIZE]byte) {
	bc.kickVoting[pkey] += 1
}
