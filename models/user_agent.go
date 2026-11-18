package models

import (
	"encoding/json"
	"math"
)

const (
	uaFnvOffset = 2166136261
	uaFnvPrime  = 16777619
)

// UserAgentClassifier là mô hình phân loại Bot / Scraper độc hại qua chuỗi User-Agent.
// Hiện thực đầy đủ interface slbf.LearnedModel[string].
type UserAgentClassifier struct {
	weights []float64
	size    uint32
	mask    uint32 // Bitmask nếu size là lũy thừa của 2 (size - 1)
}

// NewUserAgentClassifier khởi tạo mô hình với kích thước mảng trọng số mong muốn.
func NewUserAgentClassifier(size int) *UserAgentClassifier {
	if size <= 0 {
		size = 16384 // Mặc định 16K phần tử (~128 KB RAM)
	}
	s := uint32(size)
	var mask uint32
	if (s & (s - 1)) == 0 {
		mask = s - 1
	}
	return &UserAgentClassifier{
		weights: make([]float64, s),
		size:    s,
		mask:    mask,
	}
}

// Predict đánh giá xác suất chuỗi User-Agent là Bot/Scraper độc hại.
// Hàm chạy trên Hot Path: Zero Heap Allocation, độ phức tạp O(L).
func (c *UserAgentClassifier) Predict(ua string) float64 {
	l := len(ua)
	if c.size == 0 || l == 0 {
		return 0
	}

	mask := c.mask
	size := c.size
	useMask := mask != 0

	// 1. Đặc trưng độ dài chuỗi: Bot/Scanner thường có User-Agent rất ngắn
	var score float64
	if useMask {
		score = c.weights[uint32(l)&mask]
	} else {
		score = c.weights[uint32(l)%size]
	}

	// 2. Trích xuất n-gram 3 ký tự và 4 ký tự bằng 1 vòng lặp duy nhất
	if l >= 3 {
		max4 := l - 4
		for i := 0; i <= l-3; i++ {
			// Tính mã băm 3-gram
			h3 := (uint32(uaFnvOffset) ^ uint32(ua[i])) * uaFnvPrime
			h3 = (h3 ^ uint32(ua[i+1])) * uaFnvPrime
			h3 = (h3 ^ uint32(ua[i+2])) * uaFnvPrime

			if useMask {
				score += c.weights[h3&mask]
				// Kế thừa h3 để tính tiếp 4-gram
				if i <= max4 {
					h4 := (h3 ^ uint32(ua[i+3])) * uaFnvPrime
					score += c.weights[h4&mask]
				}
			} else {
				score += c.weights[h3%size]
				if i <= max4 {
					h4 := (h3 ^ uint32(ua[i+3])) * uaFnvPrime
					score += c.weights[h4%size]
				}
			}
		}
	}

	// Hàm kích hoạt Sigmoid chuyển đổi score thành xác suất [0.0, 1.0]
	return 1.0 / (1.0 + math.Exp(-score))
}

// Train huấn luyện mô hình phân loại với tập User-Agent độc (positives) và sạch (negatives).
func (c *UserAgentClassifier) Train(positives, negatives []string, epochs int, lr float64) {
	numPos := len(positives)
	numNeg := len(negatives)
	if numPos == 0 || numNeg == 0 || c.size == 0 {
		return
	}

	// Cân bằng trọng số nếu tập mẫu sạch ít hơn tập mẫu bot nhằm bảo vệ FPR
	negFactor := 1.0
	if numNeg < numPos {
		negFactor = float64(numPos) / float64(numNeg)
	}

	for epoch := 0; epoch < epochs; epoch++ {
		for _, ua := range positives {
			c.update(ua, 1.0, lr)
		}
		for _, ua := range negatives {
			c.update(ua, 0.0, lr*negFactor)
		}
	}
}

func (c *UserAgentClassifier) update(ua string, target, step float64) {
	l := len(ua)
	if l == 0 || c.size == 0 {
		return
	}

	pred := c.Predict(ua)
	err := step * (target - pred)
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
			h3 := (uint32(uaFnvOffset) ^ uint32(ua[i])) * uaFnvPrime
			h3 = (h3 ^ uint32(ua[i+1])) * uaFnvPrime
			h3 = (h3 ^ uint32(ua[i+2])) * uaFnvPrime

			if useMask {
				c.weights[h3&mask] += err
				if i <= max4 {
					h4 := (h3 ^ uint32(ua[i+3])) * uaFnvPrime
					c.weights[h4&mask] += err
				}
			} else {
				c.weights[h3%size] += err
				if i <= max4 {
					h4 := (h3 ^ uint32(ua[i+3])) * uaFnvPrime
					c.weights[h4%size] += err
				}
			}
		}
	}
}

// Hash thực hiện băm kép (Double Hashing) chuỗi User-Agent cho màng lọc Bitset dự phòng.
// Hàm hoàn toàn Zero-Allocation.
func (c *UserAgentClassifier) Hash(ua string) (uint32, uint32) {
	var h1, h2 uint32 = 0x811c9dc5, 0xcbf29ce4
	for i := 0; i < len(ua); i++ {
		b := uint32(ua[i])
		h1 ^= b
		h1 = (h1 << 5) | (h1 >> 27)
		h1 = h1*5 + 0x5bd1e995

		h2 ^= b
		h2 = (h2 << 7) | (h2 >> 25)
		h2 = h2*7 + 0x27d4eb2f
	}
	return h1, h2
}

// MarshalBinary hiện thực chuẩn encoding.BinaryMarshaler để xuất snapshot trạng thái.
func (c *UserAgentClassifier) MarshalBinary() ([]byte, error) {
	return json.Marshal(c.weights)
}

// UnmarshalBinary hiện thực chuẩn encoding.BinaryUnmarshaler để phục hồi trạng thái từ file nhị phân.
func (c *UserAgentClassifier) UnmarshalBinary(data []byte) error {
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
