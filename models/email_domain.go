package models

import "math"

const emailFnvPrime = 16777619

type EmailDomainClassifier struct {
	weights []float64
	size    uint32
}

func NewEmailDomainClassifier(size int) *EmailDomainClassifier {
	if size <= 0 {
		size = 4096
	}
	return &EmailDomainClassifier{
		weights: make([]float64, size),
		size:    uint32(size),
	}
}

func (c *EmailDomainClassifier) Predict(email string) float64 {
	atIndex, dotCount, domainLen, valid := c.parse(email)
	if !valid {
		return 0
	}

	score := c.weights[domainLen%c.size] + c.weights[(dotCount*7)%c.size]
	domainStart := atIndex + 1

	if domainLen >= 3 {
		for i := domainStart; i <= len(email)-3; i++ {
			h := uint32(2166136261)
			h = (h ^ uint32(email[i])) * emailFnvPrime
			h = (h ^ uint32(email[i+1])) * emailFnvPrime
			h = (h ^ uint32(email[i+2])) * emailFnvPrime
			score += c.weights[h%c.size]
		}
	}
	return 1.0 / (1.0 + math.Exp(-score))
}

func (c *EmailDomainClassifier) Train(positives, negatives []string, epochs int, lr float64) {
	for epoch := 0; epoch < epochs; epoch++ {
		for _, e := range positives {
			c.update(e, 1.0, lr)
		}
		for _, e := range negatives {
			c.update(e, 0.0, lr)
		}
	}
}

func (c *EmailDomainClassifier) update(email string, target, lr float64) {
	atIndex, dotCount, domainLen, valid := c.parse(email)
	if !valid {
		return
	}
	pred := c.Predict(email)
	err := lr * (target - pred)

	c.weights[domainLen%c.size] += err
	c.weights[(dotCount*7)%c.size] += err

	domainStart := atIndex + 1
	if domainLen >= 3 {
		for i := domainStart; i <= len(email)-3; i++ {
			h := uint32(2166136261)
			h = (h ^ uint32(email[i])) * emailFnvPrime
			h = (h ^ uint32(email[i+1])) * emailFnvPrime
			h = (h ^ uint32(email[i+2])) * emailFnvPrime
			c.weights[h%c.size] += err
		}
	}
}

func (c *EmailDomainClassifier) parse(email string) (atIndex int, dotCount, domainLen uint32, valid bool) {
	atIndex = -1
	l := len(email)
	for i := 0; i < l; i++ {
		if email[i] == '@' {
			atIndex = i
		} else if atIndex != -1 && email[i] == '.' {
			dotCount++
		}
	}
	if atIndex == -1 || atIndex == l-1 {
		return -1, 0, 0, false
	}
	return atIndex, dotCount, uint32(l - atIndex - 1), true
}

func (c *EmailDomainClassifier) Hash(email string) (uint32, uint32) {
	var h1, h2 uint32 = 0x12345678, 0x87654321
	for i := 0; i < len(email); i++ {
		h1 ^= uint32(email[i])
		h1 = (h1 << 5) | (h1 >> 27)
		h1 = h1*5 + 0xe6546b64
		h2 ^= uint32(email[i])
		h2 = (h2 << 3) | (h2 >> 29)
		h2 = h2*7 + 0x9e3779b9
	}
	return h1, h2
}

func (c *EmailDomainClassifier) Export() []float64 { return c.weights }

func (c *EmailDomainClassifier) Import(w []float64) { c.weights = w }
