// slbf - Simple Learned Bloom Filter
package slbf

import (
	"bytes"
	"encoding"
	"encoding/gob"
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
		return nil, errors.New("slbf: model is required")
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

func (f *Filter[T]) Add(value T) bool {
	f.mu.RLock()
	score := f.model.Predict(value)
	f.mu.RUnlock()

	if score < f.threshold {
		h1, h2 := f.model.Hash(value)

		f.mu.Lock()
		f.backup.AddHash(h1, h2)
		f.mu.Unlock()

		return false
	}
	return true
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

// filterSnapshot là cấu trúc trung gian để đóng gói toàn bộ trạng thái hệ thống
type filterSnapshot struct {
	Threshold float64
	M, K      uint32
	Bitset    []uint64
	ModelData []byte // Chứa dữ liệu của Model (JSON, Gob, v.v. tùy model quyết định)
}

// MarshalBinary đóng gói toàn bộ hệ thống
func (f *Filter[T]) MarshalBinary() ([]byte, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	snap := filterSnapshot{
		Threshold: f.threshold,
		M:         f.backup.m,
		K:         f.backup.k,
		Bitset:    f.backup.bitset,
	}

	if marshaler, ok := any(f.model).(encoding.BinaryMarshaler); ok {
		modelData, err := marshaler.MarshalBinary()
		if err != nil {
			return nil, err
		}
		snap.ModelData = modelData
	} else {
		return nil, errors.New("slbf: model does not support MarshalBinary")
	}

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(snap); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnmarshalBinary giải nén và nạp hệ thống lên RAM (Dùng cho Client)
func (f *Filter[T]) UnmarshalBinary(data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	var snap filterSnapshot
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&snap); err != nil {
		return err
	}

	f.threshold = snap.Threshold
	f.backup = &backupFilter{
		m:      snap.M,
		k:      snap.K,
		bitset: snap.Bitset,
	}

	if len(snap.ModelData) > 0 {
		if unmarshaler, ok := any(f.model).(encoding.BinaryUnmarshaler); ok {
			if err := unmarshaler.UnmarshalBinary(snap.ModelData); err != nil {
				return err
			}
		} else {
			return errors.New("slbf: binary contains model data but current model does not support UnmarshalBinary")
		}
	}
	return nil
}
