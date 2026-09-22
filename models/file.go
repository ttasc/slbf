package models

import (
	"encoding/binary"
	"math"
)

type FileHashClassifier struct {
	weights []float64
	size    uint32
}

func NewFileHashClassifier(size int) *FileHashClassifier {
	if size <= 0 {
		size = 8192
	}
	return &FileHashClassifier{
		weights: make([]float64, size),
		size:    uint32(size),
	}
}

func (c *FileHashClassifier) Predict(hash [32]byte) float64 {
	if c.size == 0 {
		return 0
	}
	var score float64
	for i := 0; i < 32; i += 4 {
		chunk := binary.BigEndian.Uint32(hash[i : i+4])
		score += c.weights[chunk%c.size]
	}
	return 1.0 / (1.0 + math.Exp(-score))
}

func (c *FileHashClassifier) Train(positives, negatives [][32]byte, epochs int, lr float64) {
	for epoch := 0; epoch < epochs; epoch++ {
		for _, hash := range positives {
			c.update(hash, 1.0, lr)
		}
		for _, hash := range negatives {
			c.update(hash, 0.0, lr)
		}
	}
}

func (c *FileHashClassifier) update(hash [32]byte, target, lr float64) {
	pred := c.Predict(hash)
	err := lr * (target - pred)
	for i := 0; i < 32; i += 4 {
		chunk := binary.BigEndian.Uint32(hash[i : i+4])
		c.weights[chunk%c.size] += err
	}
}

func (c *FileHashClassifier) Hash(hash [32]byte) (uint32, uint32) {
	var h1, h2 uint32 = 0x12345678, 0x87654321
	for i := 0; i < 32; i++ {
		h1 ^= uint32(hash[i])
		h1 = (h1 << 5) | (h1 >> 27)
		h1 = h1*5 + 0xe6546b64

		h2 ^= uint32(hash[i])
		h2 = (h2 << 3) | (h2 >> 29)
		h2 = h2*7 + 0x9e3779b9
	}
	return h1, h2
}

func (c *FileHashClassifier) Export() []float64 { return c.weights }

func (c *FileHashClassifier) Import(w []float64) { c.weights = w }

