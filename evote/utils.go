package evote

func SliceToHash(v []byte) [HashSize]byte {
	if v == nil {
		return [HashSize]byte{}
	} else {
		var hash [HashSize]byte
		copy(hash[:], v)
		return hash
	}
}

func SliceToPkey(v []byte) [PkeySize]byte {
	if v == nil {
		return [PkeySize]byte{}
	} else {
		var pkey [PkeySize]byte
		copy(pkey[:], v)
		return pkey
	}
}
