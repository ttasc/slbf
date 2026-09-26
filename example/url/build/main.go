package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ttasc/slbf"
	"github.com/ttasc/slbf/models"
)

func main() {
	posFile := flag.String("pos", "", "Positive dataset (malicious URLs)")
	negFile := flag.String("neg", "", "Negative dataset (benign URLs)")
	outFile := flag.String("out", "url_filter.bin", "Output binary file for SLBF state")
	epochs := flag.Int("epochs", 50, "Training epochs")
	lr := flag.Float64("lr", 0.05, "Learning rate")
	flag.Parse()

	if *posFile == "" || *negFile == "" {
		log.Fatal("Usage: build -pos <file> -neg <file> [-out <file>]")
	}

	fmt.Println("--- SLBF BUILDER (URL EDITION) ---")

	// 1. Read Data
	posData := readLines(*posFile)
	negData := readLines(*negFile)

	// 2. Initialize Model & Train
	model := models.NewURLClassifier(slbf.DefaultAISize)
	startTrain := time.Now()
	for e := 1; e <= *epochs; e++ {
		fmt.Printf("\r[*] Training AI model: Epoch %d/%d...    ", e, *epochs)
		model.Train(posData, negData, 1, *lr)
	}
	fmt.Printf("\r[*] Training AI model: Done (%v)           \n", time.Since(startTrain))

	// 3. Build the Learned Bloom Filter
	fmt.Print("[*] Building SLBF state (Populating bitset)... ")
	lbfBits := uint32(len(posData) * 2) // LBF uses 80% less memory
	filter, _ := slbf.New(&slbf.Config[string]{
		AISize:        slbf.DefaultAISize,
		Threshold:     slbf.DefaultThreshold,
		BackupBits:    lbfBits,
		BackupHashesK: slbf.DefaultBackupHashesK,
		Model:         model,
	})

	for _, item := range posData {
		filter.Add(item)
	}
	fmt.Println("Done.")

	// 4. Export to Binary
	fmt.Print("[*] Exporting to binary blob... ")
	data, err := filter.MarshalBinary()
	if err != nil {
		log.Fatalf("\n[-] Error marshalling filter: %v", err)
	}

	if err := os.WriteFile(*outFile, data, 0644); err != nil {
		log.Fatalf("\n[-] Error writing to file: %v", err)
	}
	fmt.Printf("Done.\n[+] Successfully generated: %s (Size: %.2f KB)\n", *outFile, float64(len(data))/1024)
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
