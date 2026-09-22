package learnedbloom

import (
	"errors"
	"sync"
)

type Config[T any] struct {
	AISize        int
	Threshold     float64
	BackupBits    uint32
	BackupHashesK uint32
	Model         LearnedModel[T] // Chỉ cần duy nhất 1 Dependency
}

// Filter quản lý cấu trúc LBF, Thread-Safe tuyệt đối.
type Filter[T any] struct {
	mu        sync.RWMutex
	model     LearnedModel[T] // Gộp chung AI và Hasher
	backup    *backupFilter
	threshold float64
}

func New[T any](cfg *Config[T]) (*Filter[T], error) {
	if cfg.Model == nil {
		return nil, errors.New("learnedbloom: Model is required")
	}

	return &Filter[T]{
		model:     cfg.Model,
		backup:    newBackupFilter(cfg.BackupBits, cfg.BackupHashesK),
		threshold: cfg.Threshold,
	}, nil
}

// MayContain siêu tốc độ (Hot Path)
func (f *Filter[T]) MayContain(value T) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Lớp 1: AI
	if f.model.Predict(value) >= f.threshold {
		return true
	}

	// Lớp 2: Bitset truyền thống (Model tự cung cấp hash)
	h1, h2 := f.model.Hash(value)
	return f.backup.ContainsHash(h1, h2)
}

// Add khóa ghi an toàn tối đa
func (f *Filter[T]) Add(value T) {
	f.mu.RLock()
	score := f.model.Predict(value)
	f.mu.RUnlock()

	if score < f.threshold {
		// Model tự băm data của nó trước khi nhét vào bitset
		h1, h2 := f.model.Hash(value)

		f.mu.Lock()
		f.backup.AddHash(h1, h2)
		f.mu.Unlock()
	}
}

// SwapModel cập nhật Zero-Downtime
func (f *Filter[T]) SwapModel(newModel LearnedModel[T], historicalPositives []T) {
	tempBackup := newBackupFilter(f.backup.m, f.backup.k)
	for _, item := range historicalPositives {
		if newModel.Predict(item) < f.threshold {
			h1, h2 := newModel.Hash(item)
			tempBackup.AddHash(h1, h2)
		}
	}

	f.mu.Lock()
	f.model = newModel
	f.backup = tempBackup
	f.mu.Unlock()
}
