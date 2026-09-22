package models

import "math"

const tokenFnvPrime = 16777619

type TokenClassifier struct {
	weights []float64
	size    uint32
}

func NewTokenClassifier(size int) *TokenClassifier {
	if size <= 0 {
		size = 4096
	}
	return &TokenClassifier{
		weights: make([]float64, size),
		size:    uint32(size),
	}
}

func (c *TokenClassifier) Predict(token string) float64 {
	l := len(token)
	if c.size == 0 || l == 0 {
		return 0
	}
	score := c.weights[uint32(l)%c.size]

	scanLen := 16
	if l < 16 {
		scanLen = l
	}
	h := uint32(2166136261)
	for i := l - 1; i >= l-scanLen; i-- {
		h = (h ^ uint32(token[i])) * tokenFnvPrime
		score += c.weights[h%c.size]
	}
	return 1.0 / (1.0 + math.Exp(-score))
}

func (c *TokenClassifier) Train(positives, negatives []string, epochs int, lr float64) {
	for epoch := 0; epoch < epochs; epoch++ {
		for _, t := range positives {
			c.update(t, 1.0, lr)
		}
		for _, t := range negatives {
			c.update(t, 0.0, lr)
		}
	}
}

func (c *TokenClassifier) update(token string, target, lr float64) {
	l := len(token)
	if l == 0 {
		return
	}
	pred := c.Predict(token)
	err := lr * (target - pred)

	c.weights[uint32(l)%c.size] += err
	scanLen := 16
	if l < 16 {
		scanLen = l
	}
	h := uint32(2166136261)
	for i := l - 1; i >= l-scanLen; i-- {
		h = (h ^ uint32(token[i])) * tokenFnvPrime
		c.weights[h%c.size] += err
	}
}

func (c *TokenClassifier) Hash(token string) (uint32, uint32) {
	var h1, h2 uint32 = 0x12345678, 0x87654321
	for i := 0; i < len(token); i++ {
		h1 ^= uint32(token[i])
		h1 = (h1 << 5) | (h1 >> 27)
		h1 = h1*5 + 0xe6546b64
		h2 ^= uint32(token[i])
		h2 = (h2 << 3) | (h2 >> 29)
		h2 = h2*7 + 0x9e3779b9
	}
	return h1, h2
}

// Export xuất trọng số ra mảng để lưu xuống file JSON/Gob.
func (c *TokenClassifier) Export() []float64 { return c.weights }

// Import nạp trọng số từ file vào model.
func (c *TokenClassifier) Import(w []float64) { c.weights = w }

