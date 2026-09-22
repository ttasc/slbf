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
	Model         LearnedModel[T]
}

type Filter[T any] struct {
	mu        sync.RWMutex
	model     LearnedModel[T]
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

func (f *Filter[T]) MayContain(value T) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Class 1: AI
	if f.model.Predict(value) >= f.threshold {
		return true
	}

	// Class 2: Traditional Bitset
	h1, h2 := f.model.Hash(value)
	return f.backup.ContainsHash(h1, h2)
}

func (f *Filter[T]) Add(value T) {
	f.mu.RLock()
	score := f.model.Predict(value)
	f.mu.RUnlock()

	if score < f.threshold {
		h1, h2 := f.model.Hash(value)

		f.mu.Lock()
		f.backup.AddHash(h1, h2)
		f.mu.Unlock()
	}
}

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
