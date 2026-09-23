package slbf

// LearnedModel định nghĩa tiêu chuẩn khắt khe cho một Plugin nhúng vào hệ thống LBF.
// Mọi Model đóng góp đều phải tuân thủ hợp đồng các điểm này.
type LearnedModel[T any] interface {
	// Predict tính toán xác suất bẩn/sạch bằng AI trên RAM (Zero-Allocation).
	Predict(value T) float64

	// Train huấn luyện lại trọng số mô hình.
	Train(positives, negatives []T, epochs int, lr float64)

	// Hash chịu trách nhiệm băm dữ liệu kiểu T thành 2 số nguyên 32-bit.
	// Dùng cho hệ thống Bloom Filter dự phòng. Thuật toán do Model tự quyết định.
	Hash(value T) (uint32, uint32)

	// Export xuất trọng số ra mảng để lưu xuống file JSON/Gob.
	Export() []float64

	// Import nạp trọng số từ file vào model.
	Import(w []float64)
}
