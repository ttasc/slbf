package learnedbloom

const (
	// Mức cấu hình hệ thống (System Baselines)
	DefaultAISize        = 16384
	DefaultThreshold     = 0.85
	DefaultBackupBits    = 100000
	DefaultBackupHashesK = 4

	// Hằng số kỹ thuật toán học nội bộ mảng Bitset
	bitElementSize  = 64
	bitElementMask  = 63
	bitElementShift = 6
)
