package main

import (
	"bufio"
	"encoding/json"
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
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "train":
		runTrain(args)
	case "bench":
		runBench(args)
	default:
		fmt.Printf("[-] Error: Invalid command '%s'\n", command)
		printUsage()
	}
}

func printUsage() {
	fmt.Println("Usage: slbf-url <command> [flags]")
	fmt.Println("\nCommands:")
	fmt.Println("  train   Train the AI URL model and output weights.json")
	fmt.Println("  bench   Run comprehensive benchmark (FPR, RAM, Throughput)")
}

// =====================================================================
// COMMAND: TRAIN
// =====================================================================
func runTrain(args []string) {
	fs := flag.NewFlagSet("train", flag.ExitOnError)
	posFile := fs.String("pos", "", "File containing positive samples (malicious URLs)")
	negFile := fs.String("neg", "", "File containing negative samples (benign URLs)")
	epochs := fs.Int("epochs", 5, "Number of training epochs")
	lr := fs.Float64("lr", 0.05, "Learning rate")
	outFile := fs.String("out", "weights.json", "Output weights file")
	fs.Parse(args)

	if *posFile == "" || *negFile == "" {
		log.Fatal("[-] Both -pos and -neg flags are required.")
	}

	fmt.Println("[*] Initializing URL model training sequence...")
	model := models.NewURLClassifier(slbf.DefaultAISize)

	posData := readLines(*posFile)
	negData := readLines(*negFile)

	trainAndSave(model, posData, negData, *epochs, *lr, *outFile)
}

// =====================================================================
// COMMAND: BENCH
// =====================================================================
func runBench(args []string) {
	fs := flag.NewFlagSet("bench", flag.ExitOnError)
	weightsFile := fs.String("weights", "weights.json", "Trained AI weights file")
	posFile := fs.String("pos", "", "Positive dataset (to populate the filter)")
	negFile := fs.String("neg", "", "Negative dataset (to query and test FPR)")
	loops := fs.Int("loops", 100, "Query loops for stress testing")
	fs.Parse(args)

	if *posFile == "" || *negFile == "" {
		log.Fatal("[-] Benchmark requires both -pos and -neg datasets.")
	}

	fmt.Println("[*] Initializing benchmark suite for URL model...")
	model := models.NewURLClassifier(slbf.DefaultAISize)
	loadWeights(model, *weightsFile)

	posData := readLines(*posFile)
	negData := readLines(*negFile)

	benchModel(model, posData, negData, *loops)
}

// =====================================================================
// CORE LOGIC & BENCHMARK SUITE
// =====================================================================

func trainAndSave(model slbf.LearnedModel[string], pos, neg []string, epochs int, lr float64, outFile string) {
	fmt.Println("\n--- TRAINING EXECUTION ---")
	start := time.Now()

	for e := 1; e <= epochs; e++ {
		progress := (float64(e) / float64(epochs)) * 100
		fmt.Printf("\r[*] Optimizing AI weights: Epoch %d/%d (%.1f%%)    ", e, epochs, progress)
		model.Train(pos, neg, 1, lr)
	}

	dur := time.Since(start)
	fmt.Printf("\r[*] Optimizing AI weights: 100%% (Done in %v)          \n", dur)

	// Direct call to Export() - No runtime type assertion needed!
	data, _ := json.Marshal(model.Export())
	os.WriteFile(outFile, data, 0644)
	fmt.Printf("[+] Weights successfully exported to: %s\n", outFile)
}

func benchModel(model slbf.LearnedModel[string], posData, negData []string, loops int) {
	if len(posData) == 0 || len(negData) == 0 {
		log.Fatal("[-] Datasets cannot be empty.")
	}

	totalItems := uint32(len(posData))
	testQueryCount := len(negData) * loops

	tbfBits := totalItems * 10
	lbfBits := totalItems * 2

	var m1, m2, m3 runtime.MemStats
	fmt.Println("\n--- BENCHMARK EXECUTION ---")

	// ---------------------------------------------------
	// PHASE 1: TRADITIONAL BLOOM FILTER
	// ---------------------------------------------------
	runtime.GC()
	runtime.ReadMemStats(&m1)
	tbf, _ := slbf.New(&slbf.Config[string]{
		AISize: slbf.DefaultAISize, Threshold: 2.0, BackupBits: tbfBits, BackupHashesK: 4, Model: model,
	})

	posLen := len(posData)
	for i, item := range posData {
		if i%(posLen/20+1) == 0 {
			fmt.Printf("\r[*] [1/4] Populating Traditional BF: %d%%    ", (i*100)/posLen)
		}
		tbf.Add(item)
	}
	fmt.Print("\r[+] [1/4] Populating Traditional BF: 100% (Done)    \n")

	runtime.ReadMemStats(&m2)
	tbfRam := m2.Alloc - m1.Alloc

	tbfFalsePositives := 0
	startTBF := time.Now()
	for l := 0; l < loops; l++ {
		fmt.Printf("\r[*] [2/4] Stress Testing Traditional BF: Loop %d/%d...    ", l+1, loops)
		for _, item := range negData {
			if tbf.MayContain(item) {
				tbfFalsePositives++
			}
		}
	}
	durTBF := time.Since(startTBF)
	fmt.Printf("\r[+] [2/4] Stress Testing Traditional BF: Done (%v)        \n", durTBF)

	// ---------------------------------------------------
	// PHASE 2: LEARNED BLOOM FILTER
	// ---------------------------------------------------
	runtime.GC()
	runtime.ReadMemStats(&m2)
	lbf, _ := slbf.New(&slbf.Config[string]{
		AISize: slbf.DefaultAISize, Threshold: 0.85, BackupBits: lbfBits, BackupHashesK: 4, Model: model,
	})

	aiCatches := 0
	for i, item := range posData {
		if i%(posLen/20+1) == 0 {
			fmt.Printf("\r[*] [3/4] Inferencing AI & Populating Learned BF: %d%%    ", (i*100)/posLen)
		}
		if model.Predict(item) >= 0.85 {
			aiCatches++
		}
		lbf.Add(item)
	}
	fmt.Print("\r[+] [3/4] Inferencing AI & Populating Learned BF: 100% (Done)    \n")

	runtime.ReadMemStats(&m3)
	lbfRam := m3.Alloc - m2.Alloc

	lbfFalsePositives := 0
	startLBF := time.Now()
	for l := 0; l < loops; l++ {
		fmt.Printf("\r[*] [4/4] Stress Testing Learned BF: Loop %d/%d...    ", l+1, loops)
		for _, item := range negData {
			if lbf.MayContain(item) {
				lbfFalsePositives++
			}
		}
	}
	durLBF := time.Since(startLBF)
	fmt.Printf("\r[+] [4/4] Stress Testing Learned BF: Done (%v)        \n", durLBF)

	// ---------------------------------------------------
	// REPORT GENERATION
	// ---------------------------------------------------
	fmt.Println("\n================= BENCHMARK REPORT =================")
	fmt.Printf("Dataset (Positives): %d items\n", len(posData))
	fmt.Printf("Queries (Negatives): %d items x %d loops = %d requests\n", len(negData), loops, testQueryCount)
	fmt.Println("----------------------------------------------------")

	tbfFPR := (float64(tbfFalsePositives) / float64(testQueryCount)) * 100
	lbfFPR := (float64(lbfFalsePositives) / float64(testQueryCount)) * 100
	aiCatchRate := (float64(aiCatches) / float64(len(posData))) * 100

	fmt.Println("1. ACCURACY (False Positive Rate)")
	fmt.Printf("   - Traditional BF : %.4f%% (%d errors)\n", tbfFPR, tbfFalsePositives)
	fmt.Printf("   - Learned BF     : %.4f%% (%d errors)\n", lbfFPR, lbfFalsePositives)
	fmt.Printf("   >> AI Filtered   : %.2f%% of malicious items caught without Bitset.\n\n", aiCatchRate)

	fmt.Println("2. MEMORY FOOTPRINT (Static Allocation)")
	fmt.Printf("   - Traditional BF : %.2f KB (Array of %d bits)\n", float64(tbfRam)/1024, tbfBits)
	fmt.Printf("   - Learned BF     : %.2f KB (Array of %d bits + AI Weights)\n", float64(lbfRam)/1024, lbfBits)
	fmt.Printf("   >> RAM Saved     : %.2f%%\n\n", 100.0-(float64(lbfRam)/float64(tbfRam)*100))

	fmt.Println("3. THROUGHPUT (Processing Speed)")
	fmt.Printf("   - Traditional BF : %v (~%.2f Req/sec)\n", durTBF, float64(testQueryCount)/durTBF.Seconds())
	fmt.Printf("   - Learned BF     : %v (~%.2f Req/sec)\n", durLBF, float64(testQueryCount)/durLBF.Seconds())
	fmt.Printf("   >> Speedup       : %.2fx faster\n", float64(durTBF)/float64(durLBF))
	fmt.Println("====================================================")
}

// ---------------------------------------------------------------------
// UTILITIES
// ---------------------------------------------------------------------

// Direct call to Import() - No runtime type assertion needed!
func loadWeights(model slbf.LearnedModel[string], path string) {
	data, err := os.ReadFile(path)
	if err == nil {
		var w []float64
		if err := json.Unmarshal(data, &w); err == nil {
			model.Import(w)
		}
	} else {
		fmt.Printf("[-] Warning: '%s' not found. Proceeding with uninitialized weights.\n", path)
	}
}

func readLines(path string) []string {
	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("\n[-] Error opening file %s: %v", path, err)
	}
	defer file.Close()

	var data []string
	scanner := bufio.NewScanner(file)
	count := 0

	fmt.Printf("\r[*] Reading dataset: %s... ", path)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text != "" {
			data = append(data, text)
			count++
			if count%50000 == 0 {
				fmt.Printf("\r[*] Reading dataset: %s... (%d lines processed)    ", path, count)
			}
		}
	}
	fmt.Printf("\r[+] Loaded %d items from %s                                \n", count, path)
	return data
}
