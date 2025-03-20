package benchmark

import (
	"testing"

	"github.com/cccarv82/compressvideo/pkg/util"
)

// BenchmarkFormatSize benchmark para formatação de tamanhos
func BenchmarkFormatSize(b *testing.B) {
	// Tamanhos para testar
	sizes := []int64{
		1024,       // 1 KB
		1048576,    // 1 MB
		52428800,   // 50 MB
		1073741824, // 1 GB
	}
	
	// Benchmark da formatação de tamanhos
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, size := range sizes {
			result := util.FormatSize(size)
			if result == "" {
				b.Fatal("Resultado da formatação vazio")
			}
		}
	}
} 