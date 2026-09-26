package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ttasc/slbf"
	"github.com/ttasc/slbf/models"
)

func main() {
	binFile := flag.String("db", "slbf.dat", "Path to SLBF binary state file")
	target := flag.String("target", "", "URL to check")
	flag.Parse()

	if *target == "" {
		log.Fatal("Usage: slbf -bin <file> -target <url>")
	}

	// 1. Initialize empty filter with URL Model
	model := models.NewURLClassifier(slbf.DefaultAISize)
	filter, _ := slbf.New(&slbf.Config[string]{Model: model})

	// 2. Load State from Binary
	data, err := os.ReadFile(*binFile)
	if err != nil {
		log.Fatalf("[-] Error reading binary file: %v", err)
	}
	if err := filter.UnmarshalBinary(data); err != nil {
		log.Fatalf("[-] Error unmarshalling state: %v", err)
	}

	// 3. Execution & Timing
	_ = filter.MayContain(*target) // Cache warmup

	start := time.Now()
	result := filter.MayContain(*target)
	latency := time.Since(start)

	// 4. Output logic evaluation
	fmt.Println("--- SLBF LOOKUP RESULT ---")
	fmt.Printf("Target  : %s\n", *target)
	fmt.Printf("Latency : %v\n", latency)

	if result {
		fmt.Println("Status  : [BLOCKED] Malicious/Exists")
		// Determine WHY it was blocked
		if model.Predict(*target) >= slbf.DefaultThreshold {
			fmt.Println("Reason  : 🧠 Blocked instantly by AI model (Bypassed Bitset).")
		} else {
			fmt.Println("Reason  : 🛡️ AI passed, but Traditional Bitset caught it.")
		}
	} else {
		fmt.Println("Status  : [PASSED] Benign/Not Found")
		fmt.Println("Reason  : ✅ Both AI and Bitset confirmed safety.")
	}
}
