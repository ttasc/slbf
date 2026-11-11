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
	posFile := flag.String("pos", "", "Positive dataset (malicious IPs)")
	negFile := flag.String("neg", "", "Negative dataset (benign IPs)")
	outFile := flag.String("out", "ip_filter.bin", "Output binary file for SLBF state")
	epochs := flag.Int("epochs", 50, "Training epochs")
	lr := flag.Float64("lr", 0.05, "Learning rate")
	flag.Parse()

	if *posFile == "" || *negFile == "" {
		log.Fatal("Usage: build -pos <file> -neg <file> [-out <file>]")
	}

	fmt.Println("--- SLBF BUILDER (IP EDITION) ---")

	// 1. Read Data
	posData := readIPs(*posFile)
	negData := readIPs(*negFile)

	// 2. Initialize Model & Train
	model := models.NewIPClassifier(slbf.DefaultAISize)
	startTrain := time.Now()
	for e := 1; e <= *epochs; e++ {
		fmt.Printf("\r[*] Training AI model: Epoch %d/%d...    ", e, *epochs)
		model.Train(posData, negData, 1, *lr)
	}
	fmt.Printf("\r[*] Training AI model: Done (%v)           \n", time.Since(startTrain))

	// 3. Build Learned Bloom Filter
	fmt.Print("[*] Building SLBF state (Populating bitset)... ")
	lbfBits := uint32(len(posData) * 2)
	filter, _ := slbf.New(&slbf.Config[uint32]{
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

func readIPs(path string) []uint32 {
	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("[-] Error opening file: %v", err)
	}
	defer file.Close()

	var data []uint32
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if t := strings.TrimSpace(scanner.Text()); t != "" {
			if ip, ok := parseIPv4(t); ok {
				data = append(data, ip)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("[-] Error reading file: %v", err)
	}
	return data
}
