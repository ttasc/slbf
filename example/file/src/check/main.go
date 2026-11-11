package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ttasc/slbf"
	"github.com/ttasc/slbf/models"
)

func main() {
	binFile := flag.String("db", "file_filter.bin", "Path to SLBF binary state file")
	target := flag.String("target", "", "SHA-256 hash in hexadecimal format (64 characters)")
	flag.Parse()

	if *target == "" {
		log.Fatal("Usage: check -db <file> -target <hex_hash>")
	}

	targetHash, ok := parseHexHash(*target)
	if !ok {
		log.Fatalf("[-] Invalid SHA-256 hexadecimal hash: %s", *target)
	}

	// 1. Initialize empty filter with FileHash Model
	model := models.NewFileHashClassifier(slbf.DefaultAISize)
	filter, _ := slbf.New(&slbf.Config[[32]byte]{Model: model})

	// 2. Load State from Binary
	data, err := os.ReadFile(*binFile)
	if err != nil {
		log.Fatalf("[-] Error reading binary file: %v", err)
	}
	if err := filter.UnmarshalBinary(data); err != nil {
		log.Fatalf("[-] Error unmarshalling state: %v", err)
	}

	// 3. Execution & Timing
	_ = filter.MayContain(targetHash) // Cache warmup

	start := time.Now()
	result := filter.MayContain(targetHash)
	latency := time.Since(start)

	// 4. Output logic evaluation
	fmt.Println("--- SLBF LOOKUP RESULT (FILE HASH) ---")
	fmt.Printf("Target  : %s\n", *target)
	fmt.Printf("Latency : %v\n", latency)

	if result {
		fmt.Println("Status  : [BLOCKED] Malicious/Exists")
		if model.Predict(targetHash) >= slbf.DefaultThreshold {
			fmt.Println("Reason  : 🧠 Blocked instantly by AI model (Bypassed Bitset).")
		} else {
			fmt.Println("Reason  : 🛡️ AI passed, but Traditional Bitset caught it.")
		}
	} else {
		fmt.Println("Status  : [PASSED] Benign/Not Found")
		fmt.Println("Reason  : ✅ Both AI and Bitset confirmed safety.")
	}
}

func parseHexHash(s string) ([32]byte, bool) {
	var hash [32]byte
	if len(s) != 64 {
		return hash, false
	}
	n, err := hex.Decode(hash[:], []byte(s))
	return hash, err == nil && n == 32
}
