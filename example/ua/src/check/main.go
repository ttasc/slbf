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
	binFile := flag.String("db", "ua_filter.bin", "Path to SLBF binary state file")
	target := flag.String("target", "", "User-Agent string to check")
	flag.Parse()

	if *target == "" {
		log.Fatal("Usage: check -db <file> -target <user_agent_string>")
	}

	// 1. Khởi tạo filter rỗng với UserAgentClassifier
	model := models.NewUserAgentClassifier(slbf.DefaultAISize)
	filter, _ := slbf.New(&slbf.Config[string]{Model: model})

	// 2. Nạp snapshot từ file nhị phân
	data, err := os.ReadFile(*binFile)
	if err != nil {
		log.Fatalf("[-] Error reading binary file: %v", err)
	}
	if err := filter.UnmarshalBinary(data); err != nil {
		log.Fatalf("[-] Error unmarshalling state: %v", err)
	}

	// 3. Thực thi tra cứu & đo độ trễ
	_ = filter.MayContain(*target) // Cache warmup

	start := time.Now()
	result := filter.MayContain(*target)
	latency := time.Since(start)

	// 4. Đánh giá và xuất kết quả
	fmt.Println("--- SLBF LOOKUP RESULT (USER-AGENT) ---")
	fmt.Printf("Target  : %s\n", *target)
	fmt.Printf("Latency : %v\n", latency)

	if result {
		fmt.Println("Status  : [BLOCKED] Malicious Bot/Scraper/Exists")
		if model.Predict(*target) >= slbf.DefaultThreshold {
			fmt.Println("Reason  : 🧠 Blocked instantly by AI model (Bypassed Bitset).")
		} else {
			fmt.Println("Reason  : 🛡️ AI passed, but Traditional Bitset caught it.")
		}
	} else {
		fmt.Println("Status  : [PASSED] Legitimate Browser/Not Found")
		fmt.Println("Reason  : ✅ Both AI and Bitset confirmed safety.")
	}
}
