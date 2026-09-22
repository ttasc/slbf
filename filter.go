package learnedbloom

// backupFilter chỉ thao tác trên toán hạng nhị phân.
type backupFilter struct {
	bitset []uint64
	m      uint32
	k      uint32
}

func newBackupFilter(m, k uint32) *backupFilter {
	if m == 0 || k == 0 {
		m, k = 1, 1 // Ngừa panic divide by zero
	}
	return &backupFilter{
		bitset: make([]uint64, (m+bitElementMask)/bitElementSize),
		m:      m,
		k:      k,
	}
}

func (bf *backupFilter) AddHash(h1, h2 uint32) {
	for i := uint32(0); i < bf.k; i++ {
		idx := (h1 + i*h2) % bf.m
		bf.bitset[idx>>bitElementShift] |= 1 << (idx & bitElementMask)
	}
}

func (bf *backupFilter) ContainsHash(h1, h2 uint32) bool {
	for i := uint32(0); i < bf.k; i++ {
		idx := (h1 + i*h2) % bf.m
		if (bf.bitset[idx>>bitElementShift] & (1 << (idx & bitElementMask))) == 0 {
			return false
		}
	}
	return true
}
