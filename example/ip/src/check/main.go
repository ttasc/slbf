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
	binFile := flag.String("db", "ip_filter.bin", "Path to SLBF binary state file")
	target := flag.String("target", "", "IPv4 address to check (e.g. 192.168.1.1)")
	flag.Parse()

	if *target == "" {
		log.Fatal("Usage: check -db <file> -target <ipv4>")
	}

	targetIP, ok := parseIPv4(*target)
	if !ok {
		log.Fatalf("[-] Invalid IPv4 address format: %s", *target)
	}

	// 1. Initialize empty filter with IP Model
	model := models.NewIPClassifier(slbf.DefaultAISize)
	filter, _ := slbf.New(&slbf.Config[uint32]{Model: model})

	// 2. Load State from Binary
	data, err := os.ReadFile(*binFile)
	if err != nil {
		log.Fatalf("[-] Error reading binary file: %v", err)
	}
	if err := filter.UnmarshalBinary(data); err != nil {
		log.Fatalf("[-] Error unmarshalling state: %v", err)
	}

	// 3. Execution & Timing
	_ = filter.MayContain(targetIP) // Cache warmup

	start := time.Now()
	result := filter.MayContain(targetIP)
	latency := time.Since(start)

	// 4. Output logic evaluation
	fmt.Println("--- SLBF LOOKUP RESULT (IP) ---")
	fmt.Printf("Target  : %s (0x%08X)\n", *target, targetIP)
	fmt.Printf("Latency : %v\n", latency)

	if result {
		fmt.Println("Status  : [BLOCKED] Malicious/Exists")
		if model.Predict(targetIP) >= slbf.DefaultThreshold {
			fmt.Println("Reason  : 🧠 Blocked instantly by AI model (Bypassed Bitset).")
		} else {
			fmt.Println("Reason  : 🛡️ AI passed, but Traditional Bitset caught it.")
		}
	} else {
		fmt.Println("Status  : [PASSED] Benign/Not Found")
		fmt.Println("Reason  : ✅ Both AI and Bitset confirmed safety.")
	}
}

func parseIPv4(s string) (uint32, bool) {
	var ip uint32
	var octet uint32
	var dots int
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= '0' && c <= '9' {
			octet = octet*10 + uint32(c-'0')
			if octet > 255 {
				return 0, false
			}
		} else if c == '.' {
			dots++
			if dots > 3 {
				return 0, false
			}
			ip = (ip << 8) | octet
			octet = 0
		} else {
			return 0, false
		}
	}
	if dots != 3 {
		return 0, false
	}
	ip = (ip << 8) | octet
	return ip, true
}
