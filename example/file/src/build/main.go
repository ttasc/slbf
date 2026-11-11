package main

import (
	"bufio"
	"encoding/hex"
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
	posFile := flag.String("pos", "", "Positive dataset (malware hashes)")
	negFile := flag.String("neg", "", "Negative dataset (benign hashes)")
	outFile := flag.String("out", "file_filter.bin", "Output binary file for SLBF state")
	epochs := flag.Int("epochs", 50, "Training epochs")
	lr := flag.Float64("lr", 0.05, "Learning rate")
	flag.Parse()

	if *posFile == "" || *negFile == "" {
		log.Fatal("Usage: build -pos <file> -neg <file> [-out <file>]")
	}

	fmt.Println("--- SLBF BUILDER (FILE HASH EDITION) ---")

	// 1. Read Data
	posData := readHashes(*posFile)
	negData := readHashes(*negFile)

	// 2. Initialize Model & Train
	model := models.NewFileHashClassifier(slbf.DefaultAISize)
	startTrain := time.Now()
	for e := 1; e <= *epochs; e++ {
		fmt.Printf("\r[*] Training AI model: Epoch %d/%d...    ", e, *epochs)
		model.Train(posData, negData, 1, *lr)
	}
	fmt.Printf("\r[*] Training AI model: Done (%v)           \n", time.Since(startTrain))

	// 3. Build Learned Bloom Filter
	fmt.Print("[*] Building SLBF state (Populating bitset)... ")
	lbfBits := uint32(len(posData) * 2)
	filter, _ := slbf.New(&slbf.Config[[32]byte]{
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

func parseHexHash(s string) ([32]byte, bool) {
	var hash [32]byte
	if len(s) != 64 {
		return hash, false
	}
	n, err := hex.Decode(hash[:], []byte(s))
	return hash, err == nil && n == 32
}

func readHashes(path string) [][32]byte {
	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("[-] Error opening file: %v", err)
	}
	defer file.Close()

	var data [][32]byte
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if t := strings.TrimSpace(scanner.Text()); t != "" {
			if hash, ok := parseHexHash(t); ok {
				data = append(data, hash)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("[-] Error reading file: %v", err)
	}
	return data
}
