# slbf - Simple Learned Bloom Filter

**slbf** is a highly optimized, generic, and zero-allocation implementation of a **Learned Bloom Filter** in Go.

By combining lightweight, embedded Machine Learning models with a traditional bitset fallback, `slbf` significantly reduces memory footprints (up to 80% RAM savings) while maintaining high throughput for large-scale filtering systems.

It is tailor-made for high-performance systems like API Gateways (JWT/Token revocation), EDRs (Malware hash scanning), Anti-Spam engines, and Web Application Firewalls (IP/URL blocking).

## Features

* **Zero-Allocation AI Inference:** The prediction models run directly on RAM without triggering garbage collection, ensuring extreme low latency.
* **Massive Memory Savings:** By letting the AI catch the majority of true positives, the backup traditional bitset can be scaled down exponentially.
* **Generic Typed:** Powered by Go Generics (`Filter[T]`), allowing you to filter `string`, `uint32`, `[32]byte`, or any custom data type seamlessly.
* **Hot-Swapping:** Update AI weights dynamically at runtime without downtime via `SwapModel()`.
* **Thread-Safe:** Built with `sync.RWMutex` for highly concurrent read-heavy environments.

## Installation

```bash
go get github.com/ttasc/slbf
```

## Quick Start

Here is a practical example of using `slbf` in an API Gateway to check if a JWT token has been revoked.

```go
package main

import (
	"fmt"
	"log"

	"github.com/ttasc/slbf"
	"github.com/ttasc/slbf/models"
)

func main() {
	// 1. Initialize the built-in Token Classifier model
	model := models.NewTokenClassifier(slbf.DefaultAISize)

	// For this example, we quickly train the model in-memory
	revokedTokens := []string{"eyJhb...revoked_1", "eyJhb...revoked_2"}
	validTokens := []string{"eyJhb...valid_1", "eyJhb...valid_2"}
	model.Train(revokedTokens, validTokens, 5, 0.05) // 5 epochs, 0.05 learning rate

	// 2. Configure and instantiate the Learned Bloom Filter
	filter, err := slbf.New(&slbf.Config[string]{
		AISize:        slbf.DefaultAISize,         // Size of the AI model's weight array
		Threshold:     slbf.DefaultThreshold,      // e.g., 0.85 AI confidence threshold
		BackupBits:    slbf.DefaultBackupBits,     // Size of the traditional fallback bitset
		BackupHashesK: slbf.DefaultBackupHashesK,  // Number of hash functions for the bitset
		Model:         model,
	})
	if err != nil {
		log.Fatalf("Failed to initialize SLBF: %v", err)
	}

	// 3. Populate the filter with known positives (Revoked Tokens)
	// If the AI predicts < Threshold, it automatically falls back to the bitset.
	for _, token := range revokedTokens {
		filter.Add(token)
	}

	// 4. Query the filter at extremely high speeds
	isRevoked := filter.MayContain("eyJhb...revoked_1")
	fmt.Printf("Is revoked_token_1 revoked? %v\n", isRevoked) // Output: true

	isRevoked = filter.MayContain("eyJhb...valid_1")
	fmt.Printf("Is valid_token_1 revoked? %v\n", isRevoked)   // Output: false (Subject to False Positive Rate)
}
```

## Custom Models

The library provides several highly optimized models out of the box in the `models/` package. Each is mathematically tuned for its specific data type.

If you want to plug in your own Machine Learning model, you simply need to implement the `slbf.LearnedModel[T]` interface:

```go
type LearnedModel[T any] interface {
	Predict(value T) float64
	Train(positives, negatives []T, epochs int, lr float64)
	Hash(value T) (uint32, uint32)
}
```

> [!WARNING]
> **Important Rule:** The `Predict(value T)` and `Hash(value T)` methods **must be zero-allocation**. Do not use standard library functions that allocate memory on the heap (like `strings.Split`, regex, or standard JSON marshallers) inside these methods, as they are invoked on the hot path millions of times per second.

> [!INFO]
> Additionally, you need to implement the `encoding.BinaryMarshaler` and `encoding.BinaryUnmarshaler` interfaces if you want the model to support import/export.

## Benchmarking

The repository includes a comprehensive CLI tool for training and benchmarking the `URLClassifier`. You can use this tool to observe the exact RAM savings and throughput speedups against a traditional Bloom Filter.

```bash
# Build the tool
go build -o bench ./example/url/bench/main.go

./bench \
    -train-pos train_pos.txt \
    -train-neg train_neg.txt \
    -epochs 100 \
    -lr 0.01 \
    -test-pos test_pos.txt \
    -test-neg test_neg.txt \
    -loops 100 \
```

## Contributing

We welcome pull requests! To contribute to this project:
1. Fork the repository.
2. Create a new branch for your feature (`git checkout -b feature/amazing-feature`).
3. Ensure your code satisfies standard Go formatting (`go fmt`) and passes linting.
4. If you are adding a new classifier to `models/`, mathematically prove that `Predict` and `Hash` involve **zero heap allocations**.
5. Submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
