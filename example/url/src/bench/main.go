package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/ttasc/slbf"
	"github.com/ttasc/slbf/models"
)

func main() {
	trainPosFile := flag.String("train-pos", "", "Positive dataset for training")
	trainNegFile := flag.String("train-neg", "", "Negative dataset for training")
	testPosFile := flag.String("test-pos", "", "Positive dataset for testing (inserted into filter)")
	testNegFile := flag.String("test-neg", "", "Negative dataset for testing (queries to check FPR)")
	loops := flag.Int("loops", 100, "Query loops for stress testing")
	epochs := flag.Int("epochs", 100, "Number of training epochs")
	lr := flag.Float64("lr", 0.01, "Learning rate")
	flag.Parse()

	if *trainPosFile == "" || *trainNegFile == "" || *testPosFile == "" || *testNegFile == "" {
		log.Fatal("Usage: benchmark -train-pos <f> -train-neg <f> -test-pos <f> -test-neg <f> [-loops <int>]")
	}

	// 1. Đọc 4 tập dữ liệu độc lập
	trainPosData := readLines(*trainPosFile)
	trainNegData := readLines(*trainNegFile)
	testPosData := readLines(*testPosFile)
	testNegData := readLines(*testNegFile)

	totalTestPos := len(testPosData)
	totalTestNeg := len(testNegData)
	queryCount := totalTestNeg * *loops

	if totalTestPos == 0 || totalTestNeg == 0 {
		log.Fatal("[-] Test datasets cannot be empty.")
	}

	// 2. Huấn luyện AI trên tập TRAIN
	fmt.Println("[*] Initializing URL model training sequence...")
	model := models.NewURLClassifier(slbf.DefaultAISize)

	start := time.Now()
	for e := 1; e <= *epochs; e++ {
		progress := (float64(e) / float64(*epochs)) * 100
		fmt.Printf("\r[*] [1/5] Training AI using {%s, %s}: Epoch %d/%d (%.1f%%)\033[K", *trainPosFile, *trainNegFile, e, *epochs, progress)
		model.Train(trainPosData, trainNegData, 1, *lr)
	}
	dur := time.Since(start)
	fmt.Printf("\r[+] [1/5] Training AI: 100%% (Done in %v)\033[K\n", dur)


	// Khai báo kích thước công bằng (LBF dùng ít hơn TBF 5 lần)
	tbfBits := uint32(totalTestPos * 10)
	lbfBits := uint32(totalTestPos * 2)

	var m1, m2, m3 runtime.MemStats

	// ============================================
	// PHASE 1: TRADITIONAL BLOOM FILTER
	// ============================================
	runtime.GC()
	runtime.ReadMemStats(&m1)
	tbf, _ := slbf.New(&slbf.Config[string]{
		AISize: slbf.DefaultAISize, Threshold: 2.0, BackupBits: tbfBits, BackupHashesK: 4, Model: model,
	})

	for i, item := range testPosData {
		if i%(totalTestPos/10+1) == 0 {
			fmt.Printf("\r[*] [2/5] Populating TBF with {%s}: %d%%\033[K", *testPosFile, (i*100)/totalTestPos)
		}
		tbf.Add(item)
	}
	fmt.Print("\r[+] [2/5] Populating TBF: 100% (Done)\033[K\n")

	runtime.ReadMemStats(&m2)
	tbfRam := float64(m2.Alloc-m1.Alloc) / 1024

	tbfFp := 0
	startTBF := time.Now()
	for l := 0; l < *loops; l++ {
		fmt.Printf("\r[*] [3/5] Stress Testing TBF using {%s}: Loop %d/%d...\033[K", *testNegFile, l+1, *loops)
		for _, item := range testNegData {
			if tbf.MayContain(item) {
				tbfFp++
			}
		}
	}
	durTBF := time.Since(startTBF)
	fmt.Printf("\r[+] [3/5] Stress Testing TBF: Done (%v)\033[K\n", durTBF)

	// ============================================
	// PHASE 2: LEARNED BLOOM FILTER
	// ============================================
	runtime.GC()
	runtime.ReadMemStats(&m2)
	lbf, _ := slbf.New(&slbf.Config[string]{
		AISize: slbf.DefaultAISize, Threshold: slbf.DefaultThreshold, BackupBits: lbfBits, BackupHashesK: 4, Model: model,
	})

	aiCatches := 0
	for i, item := range testPosData {
		if i%(totalTestPos/10+1) == 0 {
			fmt.Printf("\r[*] [4/5] Inferencing AI & Populating LBF with {%s}: %d%% (AI caught %d)\033[K", *testPosFile, (i*100)/totalTestPos, aiCatches)
		}
		// Đánh giá hiệu suất AI trên tập TEST BẨN (Unseen Data)
		if lbf.Add(item) {
			aiCatches++
		}
	}
	fmt.Printf("\r[+] [4/5] Inferencing AI & Populating LBF: 100%% (Done - AI caught %d)\033[K\n", aiCatches)

	runtime.ReadMemStats(&m3)
	lbfRam := float64(m3.Alloc-m2.Alloc) / 1024

	lbfFp := 0
	startLBF := time.Now()
	for l := 0; l < *loops; l++ {
		fmt.Printf("\r[*] [5/5] Stress Testing LBF using {%s}: Loop %d/%d...\033[K", *testNegFile, l+1, *loops)
		for _, item := range testNegData {
			if lbf.MayContain(item) {
				lbfFp++
			}
		}
	}
	durLBF := time.Since(startLBF)
	fmt.Printf("\r[+] [5/5] Stress Testing LBF: Done (%v)\033[K\n", durLBF)

	// ============================================
	// BÁO CÁO TOÁN HỌC (REPORT)
	// ============================================
	tbfFPR := (float64(tbfFp) / float64(queryCount)) * 100
	lbfFPR := (float64(lbfFp) / float64(queryCount)) * 100
	aiEff := (float64(aiCatches) / float64(totalTestPos)) * 100
	ramSaved := 100.0 - (lbfRam / tbfRam * 100)
	tbfTPS := float64(queryCount) / durTBF.Seconds()
	lbfTPS := float64(queryCount) / durLBF.Seconds()
	speedup := durTBF.Seconds() / durLBF.Seconds()

	fmt.Println("================= BENCHMARK REPORT =================")
	fmt.Printf("Dataset (Positives): %d items (Blacklist)\n", totalTestPos)
	fmt.Printf("Queries (Negatives): %d unseen items x %d loops = %d requests\n", totalTestNeg, *loops, queryCount)
	fmt.Println("----------------------------------------------------")
	fmt.Println("1. ACCURACY (False Positive Rate on UNSEEN data)")
	fmt.Printf("   - Traditional BF : %.4f%% (%d errors)\n", tbfFPR, tbfFp)
	fmt.Printf("   - Learned BF     : %.4f%% (%d errors)\n", lbfFPR, lbfFp)
	fmt.Printf("   >> AI Efficiency : %.2f%% of malicious items caught without Bitset.\n\n", aiEff)

	fmt.Println("2. MEMORY FOOTPRINT (Heap Allocation)")
	fmt.Printf("   - Traditional BF : %.2f KB (Array of %d bits)\n", tbfRam, tbfBits)
	fmt.Printf("   - Learned BF     : %.2f KB (Array of %d bits + AI Weights)\n", lbfRam, lbfBits)
	fmt.Printf("   >> RAM Saved     : %.2f%%\n\n", ramSaved)

	fmt.Println("3. THROUGHPUT (Processing Speed)")
	fmt.Printf("   - Traditional BF : %v (~%.2f Req/sec)\n", durTBF.Seconds(), tbfTPS)
	fmt.Printf("   - Learned BF     : %v (~%.2f Req/sec)\n", durLBF.Seconds(), lbfTPS)
	fmt.Printf("   >> Speedup       : %.2fx faster\n", speedup)
	fmt.Println("====================================================")
}

func readLines(path string) []string {
	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("[-] Error opening file: %v", err)
	}
	defer file.Close()
	var data []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if t := strings.TrimSpace(scanner.Text()); t != "" {
			data = append(data, t)
		}
	}
	return data
}
