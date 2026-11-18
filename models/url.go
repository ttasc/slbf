package models

import (
	"encoding/json"
	"math"
)

const (
	urlFnvOffset = 2166136261
	urlFnvPrime  = 16777619
)

type URLClassifier struct {
	weights []float64
	size    uint32
	mask    uint32 // Bitmask nếu size là lũy thừa của 2 (size - 1)
}

func NewURLClassifier(size int) *URLClassifier {
	if size <= 0 {
		size = 16384
	}
	s := uint32(size)
	var mask uint32
	if (s & (s - 1)) == 0 {
		mask = s - 1
	}
	return &URLClassifier{
		weights: make([]float64, s),
		size:    s,
		mask:    mask,
	}
}

func (c *URLClassifier) Predict(url string) float64 {
	l := len(url)
	if c.size == 0 || l == 0 {
		return 0
	}

	mask := c.mask
	size := c.size
	useMask := mask != 0

	var score float64
	if useMask {
		score = c.weights[uint32(l)&mask]
	} else {
		score = c.weights[uint32(l)%size]
	}

	if l >= 3 {
		max4 := l - 4
		for i := 0; i <= l-3; i++ {
			// Tính 3-gram
			h3 := (uint32(urlFnvOffset) ^ uint32(url[i])) * urlFnvPrime
			h3 = (h3 ^ uint32(url[i+1])) * urlFnvPrime
			h3 = (h3 ^ uint32(url[i+2])) * urlFnvPrime

			if useMask {
				score += c.weights[h3&mask]
				// Kế thừa h3 để tính tiếp 4-gram mà không băm lại từ đầu
				if i <= max4 {
					h4 := (h3 ^ uint32(url[i+3])) * urlFnvPrime
					score += c.weights[h4&mask]
				}
			} else {
				score += c.weights[h3%size]
				if i <= max4 {
					h4 := (h3 ^ uint32(url[i+3])) * urlFnvPrime
					score += c.weights[h4%size]
				}
			}
		}
	}

	return 1.0 / (1.0 + math.Exp(-score))
}

func (c *URLClassifier) Train(positives, negatives []string, epochs int, lr float64) {
	for epoch := 0; epoch < epochs; epoch++ {
		for _, url := range positives {
			c.update(url, 1.0, lr)
		}
		for _, url := range negatives {
			c.update(url, 0.0, lr)
		}
	}
}

func (c *URLClassifier) update(url string, target, lr float64) {
	l := len(url)
	if l == 0 || c.size == 0 {
		return
	}

	pred := c.Predict(url)
	err := lr * (target - pred)
	if err == 0 {
		return
	}

	mask := c.mask
	size := c.size
	useMask := mask != 0

	if useMask {
		c.weights[uint32(l)&mask] += err
	} else {
		c.weights[uint32(l)%size] += err
	}

	if l >= 3 {
		max4 := l - 4
		for i := 0; i <= l-3; i++ {
			h3 := (uint32(urlFnvOffset) ^ uint32(url[i])) * urlFnvPrime
			h3 = (h3 ^ uint32(url[i+1])) * urlFnvPrime
			h3 = (h3 ^ uint32(url[i+2])) * urlFnvPrime

			if useMask {
				c.weights[h3&mask] += err
				if i <= max4 {
					h4 := (h3 ^ uint32(url[i+3])) * urlFnvPrime
					c.weights[h4&mask] += err
				}
			} else {
				c.weights[h3%size] += err
				if i <= max4 {
					h4 := (h3 ^ uint32(url[i+3])) * urlFnvPrime
					c.weights[h4%size] += err
				}
			}
		}
	}
}

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

// MarshalBinary hiện thực chuẩn encoding.BinaryMarshaler
func (c *URLClassifier) MarshalBinary() ([]byte, error) {
	return json.Marshal(c.weights)
}

// UnmarshalBinary hiện thực chuẩn encoding.BinaryUnmarshaler
func (c *URLClassifier) UnmarshalBinary(data []byte) error {
	if err := json.Unmarshal(data, &c.weights); err != nil {
		return err
	}
	c.size = uint32(len(c.weights))
	if (c.size & (c.size - 1)) == 0 && c.size > 0 {
		c.mask = c.size - 1
	} else {
		c.mask = 0
	}
	return nil
}
