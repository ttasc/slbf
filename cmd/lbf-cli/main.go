package main

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/ttasc/lbf"
	"github.com/ttasc/lbf/models"
)

const banner = `
======================================================
  🧠 LEARNED BLOOM FILTER CLI (Zero-Allocation) 🚀
======================================================
`

type Weightable interface {
	Export() []float64
	Import([]float64)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "train":
		runTrain(args)
	case "check":
		runCheck(args)
	case "bench":
		runBench(args)
	default:
		fmt.Printf("Lỗi: Lệnh không hợp lệ '%s'\n", command)
		printUsage()
	}
}

func printUsage() {
	fmt.Println(banner)
	fmt.Println("Sử dụng: lbf-cli <lệnh> [các cờ]")
	fmt.Println("\nCác lệnh khả dụng:")
	fmt.Println("  train   Huấn luyện mô hình AI và xuất ra file weights.json")
	fmt.Println("  check   Kiểm tra một phần tử đơn lẻ (Có/Không có AI)")
	fmt.Println("  bench   Đánh giá toàn diện: Tốc độ, Tiêu thụ RAM & ĐỘ CHÍNH XÁC (FPR)")
}

// =====================================================================
// 1. LỆNH TRAIN & 2. LỆNH CHECK
// =====================================================================
func runTrain(args []string) {
	fs := flag.NewFlagSet("train", flag.ExitOnError)
	modelType := fs.String("model", "url", "Loại model")
	posFile := fs.String("pos", "", "File TXT mẫu bẩn")
	negFile := fs.String("neg", "", "File TXT mẫu sạch")
	epochs := fs.Int("epochs", 5, "Số vòng lặp")
	lr := fs.Float64("lr", 0.05, "Tốc độ học")
	outFile := fs.String("out", "weights.json", "File lưu trọng số")
	fs.Parse(args)

	if *posFile == "" || *negFile == "" {
		log.Fatal("Phải cung cấp đủ -pos và -neg")
	}

	fmt.Printf("[*] Chuẩn bị huấn luyện model '%s'\n", *modelType)
	switch *modelType {
	case "url":
		model := models.NewURLClassifier(learnedbloom.DefaultAISize)
		trainAndSave(model, readAndParse(*posFile, parseString), readAndParse(*negFile, parseString), *epochs, *lr, *outFile)
	case "ip":
		model := models.NewIPClassifier(learnedbloom.DefaultAISize)
		trainAndSave(model, readAndParse(*posFile, parseIP), readAndParse(*negFile, parseIP), *epochs, *lr, *outFile)
	default:
		log.Fatalf("Lỗi: Model '%s' chưa được hỗ trợ trong demo này!", *modelType)
	}
}

func runCheck(args []string) {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	modelType := fs.String("model", "url", "Loại model")
	weightsFile := fs.String("weights", "weights.json", "File trọng số AI")
	target := fs.String("target", "", "Dữ liệu cần tra cứu")
	disableAI := fs.Bool("disable-ai", false, "Chạy chế độ Traditional Bloom Filter")
	fs.Parse(args)

	if *target == "" {
		log.Fatal("Vui lòng cung cấp -target")
	}
	fmt.Printf("\n--- CHẾ ĐỘ: %s ---\n", map[bool]string{true: "TRADITIONAL (AI TẮT)", false: "LEARNED BF (AI BẬT)"}[*disableAI])

	switch *modelType {
	case "url":
		model := models.NewURLClassifier(learnedbloom.DefaultAISize)
		loadWeights(model, *weightsFile)
		executeModel(model, parseString(*target), *disableAI)
	case "ip":
		model := models.NewIPClassifier(learnedbloom.DefaultAISize)
		loadWeights(model, *weightsFile)
		executeModel(model, parseIP(*target), *disableAI)
	}
}

// =====================================================================
// 3. LỆNH BENCH: ĐÁNH GIÁ TOÀN DIỆN (CÓ LOG PROGRESS)
// =====================================================================
func runBench(args []string) {
	fs := flag.NewFlagSet("bench", flag.ExitOnError)
	modelType := fs.String("model", "url", "Loại model")
	weightsFile := fs.String("weights", "weights.json", "File trọng số AI")
	posFile := fs.String("pos", "", "File chứa dữ liệu ĐỘC HẠI")
	negFile := fs.String("neg", "", "File chứa dữ liệu SẠCH")
	loops := fs.Int("loops", 100, "Số vòng lặp query để test tốc độ")
	fs.Parse(args)

	if *posFile == "" || *negFile == "" {
		log.Fatal("Lệnh benchmark yêu cầu cả -pos và -neg.")
	}

	fmt.Printf("[*] Đang khởi tạo Benchmark cho model '%s'\n", *modelType)

	switch *modelType {
	case "url":
		model := models.NewURLClassifier(learnedbloom.DefaultAISize)
		loadWeights(model, *weightsFile)
		benchModel(model, readAndParse(*posFile, parseString), readAndParse(*negFile, parseString), *loops)
	case "ip":
		model := models.NewIPClassifier(learnedbloom.DefaultAISize)
		loadWeights(model, *weightsFile)
		benchModel(model, readAndParse(*posFile, parseIP), readAndParse(*negFile, parseIP), *loops)
	}
}

// =====================================================================
// CÁC HÀM HELPER & CORE LOGIC
// =====================================================================

func trainAndSave[T any](model learnedbloom.LearnedModel[T], pos, neg []T, epochs int, lr float64, outFile string) {
	start := time.Now()
	fmt.Print("[*] Đang tiến hành huấn luyện (Training)... ")
	model.Train(pos, neg, epochs, lr)
	fmt.Printf("Xong! (%v)\n", time.Since(start))

	if wModel, ok := any(model).(Weightable); ok {
		data, _ := json.Marshal(wModel.Export())
		os.WriteFile(outFile, data, 0644)
	}
}

func executeModel[T any](model learnedbloom.LearnedModel[T], target T, disableAI bool) {
	cfg := &learnedbloom.Config[T]{
		AISize:        learnedbloom.DefaultAISize,
		Threshold:     learnedbloom.DefaultThreshold,
		BackupBits:    100000,
		BackupHashesK: 4,
		Model:         model,
	}
	if disableAI {
		cfg.Threshold = 2.0
	}
	filter, _ := learnedbloom.New(cfg)

	start := time.Now()
	result := filter.MayContain(target)
	latency := time.Since(start)

	lbl := "✅ SẠCH"
	if result {
		lbl = "🛑 BẨN"
	}
	fmt.Printf("Mục tiêu      : %v\nKết quả TỔNG  : %t -> %s\nĐộ trễ        : %v\n", target, result, lbl, latency)
}

// benchModel: TRÁI TIM CỦA BÀI KIỂM TRA TOÀN DIỆN VỚI HIỆU ỨNG PROGRESS BAR
func benchModel[T any](model learnedbloom.LearnedModel[T], posData, negData []T, loops int) {
	if len(posData) == 0 || len(negData) == 0 {
		log.Fatal("Dữ liệu test không được rỗng!")
	}

	totalItems := uint32(len(posData))
	testQueryCount := len(negData) * loops

	tbfBits := totalItems * 10
	lbfBits := totalItems * 2

	var m1, m2, m3 runtime.MemStats
	fmt.Println("\n--- BẮT ĐẦU CHẠY STRESS TEST (Vui lòng đợi) ---")

	// ---------------------------------------------------
	// 1. ĐO LƯỜNG TRADITIONAL BLOOM FILTER (TẮT AI)
	// ---------------------------------------------------
	runtime.GC()
	runtime.ReadMemStats(&m1)
	tbf, _ := learnedbloom.New(&learnedbloom.Config[T]{
		AISize: learnedbloom.DefaultAISize, Threshold: 2.0, BackupBits: tbfBits, BackupHashesK: 4, Model: model,
	})

	// Tiến độ Nạp TBF
	posLen := len(posData)
	for i, item := range posData {
		if i%(posLen/20+1) == 0 { // Update UI 20 lần (mỗi 5%)
			fmt.Printf("\r[1/4] Nạp dữ liệu vào Traditional BF: %d%%    ", (i*100)/posLen)
		}
		tbf.Add(item)
	}
	fmt.Printf("\r[1/4] Nạp dữ liệu vào Traditional BF: 100%% ✅\n")

	runtime.ReadMemStats(&m2)
	tbfRam := m2.Alloc - m1.Alloc

	// Tiến độ Query TBF
	tbfFalsePositives := 0
	startTBF := time.Now()
	for l := 0; l < loops; l++ {
		// Dùng \r để đè dòng cũ, thêm khoảng trắng ở cuối để xóa text thừa
		fmt.Printf("\r[2/4] Stress Test Traditional BF (Vòng %d/%d)...      ", l+1, loops)
		for _, item := range negData {
			if tbf.MayContain(item) {
				tbfFalsePositives++
			}
		}
	}
	durTBF := time.Since(startTBF)
	fmt.Printf("\r[2/4] Stress Test Traditional BF: Xong! (%v) ✅      \n", durTBF)

	// ---------------------------------------------------
	// 2. ĐO LƯỜNG LEARNED BLOOM FILTER (CÓ AI)
	// ---------------------------------------------------
	runtime.GC()
	runtime.ReadMemStats(&m2)
	lbf, _ := learnedbloom.New(&learnedbloom.Config[T]{
		AISize: learnedbloom.DefaultAISize, Threshold: 0.85, BackupBits: lbfBits, BackupHashesK: 4, Model: model,
	})

	// Tiến độ Nạp LBF & Inference
	aiCatches := 0
	for i, item := range posData {
		if i%(posLen/20+1) == 0 {
			fmt.Printf("\r[3/4] AI xử lý và nạp dữ liệu LBF: %d%%    ", (i*100)/posLen)
		}
		if model.Predict(item) >= 0.85 {
			aiCatches++
		}
		lbf.Add(item)
	}
	fmt.Printf("\r[3/4] AI xử lý và nạp dữ liệu LBF: 100%% ✅\n")

	runtime.ReadMemStats(&m3)
	lbfRam := m3.Alloc - m2.Alloc

	// Tiến độ Query LBF
	lbfFalsePositives := 0
	startLBF := time.Now()
	for l := 0; l < loops; l++ {
		fmt.Printf("\r[4/4] Stress Test Learned BF (Vòng %d/%d)...      ", l+1, loops)
		for _, item := range negData {
			if lbf.MayContain(item) {
				lbfFalsePositives++
			}
		}
	}
	durLBF := time.Since(startLBF)
	fmt.Printf("\r[4/4] Stress Test Learned BF: Xong! (%v) ✅      \n", durLBF)

	// ---------------------------------------------------
	// BÁO CÁO
	// ---------------------------------------------------
	fmt.Println("\n================= 📊 BÁO CÁO BENCHMARK TOÀN DIỆN =================")
	fmt.Printf("Dữ liệu nạp (Bẩn) : %d items\n", len(posData))
	fmt.Printf("Truy vấn (Sạch)   : %d items x %d loops = %d requests\n", len(negData), loops, testQueryCount)
	fmt.Println("------------------------------------------------------------------")

	tbfFPR := (float64(tbfFalsePositives) / float64(testQueryCount)) * 100
	lbfFPR := (float64(lbfFalsePositives) / float64(testQueryCount)) * 100
	aiCatchRate := (float64(aiCatches) / float64(len(posData))) * 100

	fmt.Println("🎯 1. ĐỘ CHÍNH XÁC (Tỷ lệ chặn nhầm - False Positive Rate):")
	fmt.Printf("   - Traditional BF : %.4f%% (Lỗi %d lần)\n", tbfFPR, tbfFalsePositives)
	fmt.Printf("   - Learned BF     : %.4f%% (Lỗi %d lần)\n", lbfFPR, lbfFalsePositives)
	fmt.Printf("   >> Trí tuệ AI    : Đã tự nhận diện đúng %.2f%% mã độc không cần lưu vào Bitset!\n", aiCatchRate)

	fmt.Println("\n💾 2. TIÊU THỤ RAM TĨNH (Memory Footprint):")
	fmt.Printf("   - Traditional BF : %.2f KB (Mảng %d bits)\n", float64(tbfRam)/1024, tbfBits)
	fmt.Printf("   - Learned BF     : %.2f KB (Mảng %d bits + AI Weights)\n", float64(lbfRam)/1024, lbfBits)
	fmt.Printf("   >> Tiết kiệm     : %.2f%%\n", 100.0-(float64(lbfRam)/float64(tbfRam)*100))

	fmt.Println("\n⚡ 3. TỐC ĐỘ XỬ LÝ (Throughput):")
	fmt.Printf("   - Traditional BF : %v (~%.2f Req/sec)\n", durTBF, float64(testQueryCount)/durTBF.Seconds())
	fmt.Printf("   - Learned BF     : %v (~%.2f Req/sec)\n", durLBF, float64(testQueryCount)/durLBF.Seconds())
	fmt.Printf("   >> Tốc độ        : Nhanh hơn %.2f lần\n", float64(durTBF)/float64(durLBF))
	fmt.Println("==================================================================")
}

// ---------------------------------------------------------------------
// Utils: Có Progress Update khi đọc file lớn
// ---------------------------------------------------------------------
func readAndParse[T any](path string, parser func(string) T) []T {
	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("\nLỗi mở file %s: %v", path, err)
	}
	defer file.Close()

	var data []T
	scanner := bufio.NewScanner(file)
	count := 0

	fmt.Printf("\r[*] Đang nạp file %s...", path)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text != "" {
			data = append(data, parser(text))
			count++
			if count%50000 == 0 { // In log tiến độ cứ mỗi 50k dòng để tránh đứng hình
				fmt.Printf("\r[*] Đang nạp file %s... (%d dòng)", path, count)
			}
		}
	}
	// Ghi đè thông báo khi đọc xong, thêm khoảng trắng để xóa text cũ
	fmt.Printf("\r[+] Đã nạp thành công %d dòng từ %s               \n", count, path)
	return data
}

func loadWeights(model Weightable, path string) {
	data, err := os.ReadFile(path)
	if err == nil {
		var w []float64
		if err := json.Unmarshal(data, &w); err == nil {
			model.Import(w)
		}
	} else {
		fmt.Printf("[!] Cảnh báo: Không tìm thấy %s, model sẽ chạy với trọng số 0 (Random).\n", path)
	}
}
func parseString(s string) string { return s }
func parseIP(s string) uint32 {
	ip := net.ParseIP(s).To4()
	return binary.BigEndian.Uint32(ip)
}
func parseHash(s string) [32]byte {
	b, _ := hex.DecodeString(s)
	var arr [32]byte
	copy(arr[:], b)
	return arr
}
