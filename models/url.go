package models

import "math"

const (
	urlFnvOffset = 2166136261
	urlFnvPrime  = 16777619
)

type URLClassifier struct {
	weights []float64
	size    uint32
}

func NewURLClassifier(size int) *URLClassifier {
	if size <= 0 {
		size = 16384
	}
	return &URLClassifier{
		weights: make([]float64, size),
		size:    uint32(size),
	}
}

func (c *URLClassifier) Predict(url string) float64 {
	l := len(url)
	if c.size == 0 || l == 0 {
		return 0
	}

	score := c.weights[uint32(l)%c.size]
	if l >= 3 {
		for i := 0; i <= l-3; i++ {
			h := uint32(urlFnvOffset)
			h = (h ^ uint32(url[i])) * urlFnvPrime
			h = (h ^ uint32(url[i+1])) * urlFnvPrime
			h = (h ^ uint32(url[i+2])) * urlFnvPrime
			score += c.weights[h%c.size]
		}
	}
	if l >= 4 {
		for i := 0; i <= l-4; i++ {
			h := uint32(urlFnvOffset)
			h = (h ^ uint32(url[i])) * urlFnvPrime
			h = (h ^ uint32(url[i+1])) * urlFnvPrime
			h = (h ^ uint32(url[i+2])) * urlFnvPrime
			h = (h ^ uint32(url[i+3])) * urlFnvPrime
			score += c.weights[h%c.size]
		}
	}

	return 1.0 / (1.0 + math.Exp(-score))
}

func (c *URLClassifier) Train(positives, negatives []string, epochs int, lr float64) {
	for range epochs {
		for _, url := range positives {
			c.update(url, 1.0, lr)
		}
		for _, url := range negatives {
			c.update(url, 0.0, lr)
		}
	}
}

// Lặp lại logic duyệt chuỗi để giữ Zero-Allocation (Bù CPU lấy RAM, an toàn cho GC)
func (c *URLClassifier) update(url string, target, lr float64) {
	l := len(url)
	if l == 0 {
		return
	}
	pred := c.Predict(url)
	err := lr * (target - pred)

	c.weights[uint32(l)%c.size] += err
	if l >= 3 {
		for i := 0; i <= l-3; i++ {
			h := uint32(urlFnvOffset)
			h = (h ^ uint32(url[i])) * urlFnvPrime
			h = (h ^ uint32(url[i+1])) * urlFnvPrime
			h = (h ^ uint32(url[i+2])) * urlFnvPrime
			c.weights[h%c.size] += err
		}
	}
	if l >= 4 {
		for i := 0; i <= l-4; i++ {
			h := uint32(urlFnvOffset)
			h = (h ^ uint32(url[i])) * urlFnvPrime
			h = (h ^ uint32(url[i+1])) * urlFnvPrime
			h = (h ^ uint32(url[i+2])) * urlFnvPrime
			h = (h ^ uint32(url[i+3])) * urlFnvPrime
			c.weights[h%c.size] += err
		}
	}
}

// Hash: Cặp Murmur3 cho chuỗi
func (c *URLClassifier) Hash(url string) (uint32, uint32) {
	var h1, h2 uint32 = 0x12345678, 0x87654321
	for i := 0; i < len(url); i++ {
		h1 ^= uint32(url[i])
		h1 = (h1 << 5) | (h1 >> 27)
		h1 = h1*5 + 0xe6546b64

		h2 ^= uint32(url[i])
		h2 = (h2 << 3) | (h2 >> 29)
		h2 = h2*7 + 0x9e3779b9
	}
	return h1, h2
}

// Export xuất trọng số ra mảng để lưu xuống file JSON/Gob.
func (c *URLClassifier) Export() []float64 { return c.weights }

// Import nạp trọng số từ file vào model.
func (c *URLClassifier) Import(w []float64) { c.weights = w }
