package models

import "math"

type IPClassifier struct {
	weights []float64
	size    uint32
}

func NewIPClassifier(size int) *IPClassifier {
	if size <= 0 {
		size = 1024
	}
	return &IPClassifier{
		weights: make([]float64, size),
		size:    uint32(size),
	}
}

func (c *IPClassifier) Predict(ip uint32) float64 {
	if c.size == 0 {
		return 0
	}
	score := c.weights[(ip>>24)&0xFF%c.size] +
		c.weights[(ip>>16)&0xFF%c.size] +
		c.weights[(ip>>8)&0xFF%c.size] +
		c.weights[ip&0xFF%c.size]

	return 1.0 / (1.0 + math.Exp(-score))
}

func (c *IPClassifier) Train(positives, negatives []uint32, epochs int, lr float64) {
	if c.size == 0 {
		return
	}
	for epoch := 0; epoch < epochs; epoch++ {
		for _, ip := range positives {
			c.update(ip, 1.0, lr)
		}
		for _, ip := range negatives {
			c.update(ip, 0.0, lr)
		}
	}
}

func (c *IPClassifier) update(ip uint32, target, lr float64) {
	i1 := (ip >> 24) & 0xFF % c.size
	i2 := (ip >> 16) & 0xFF % c.size
	i3 := (ip >> 8) & 0xFF % c.size
	i4 := ip & 0xFF % c.size

	score := c.weights[i1] + c.weights[i2] + c.weights[i3] + c.weights[i4]
	pred := 1.0 / (1.0 + math.Exp(-score))
	err := lr * (target - pred)

	c.weights[i1] += err
	c.weights[i2] += err
	c.weights[i3] += err
	c.weights[i4] += err
}

func (c *IPClassifier) Hash(ip uint32) (uint32, uint32) {
	h1 := ip * 0xcc9e2d51
	h1 = (h1 << 15) | (h1 >> 17)
	h1 *= 0x1b873593

	h2 := ip * 0x85ebca6b
	h2 = (h2 << 13) | (h2 >> 19)
	h2 *= 0xc2b2ae35

	return h1, h2
}
