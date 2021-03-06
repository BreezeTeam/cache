package bloom

import (
	"fmt"
	"testing"
)

func TestMemoryBloomFilter(t *testing.T) {
	filter := NewBloomFilter(uint(16<<20), 5)
	RandTest(t, filter, 50000)

}

func RandTest(t *testing.T, filter *BloomFilter, n int) {

	for i := 0; i < n; i++ {
		filter.PutString(fmt.Sprintf("r%d", i))
	}

	var miss_numbers int

	for i := 0; i < n; i++ {
		exists_record := fmt.Sprintf("r%d", i)
		not_exists_record := fmt.Sprintf("rr%d", i)
		if !filter.HasString(exists_record) {
			miss_numbers++
		}

		if filter.HasString(not_exists_record) {
			miss_numbers++
		}
	}
	hit_rate := float64(n-miss_numbers) / float64(n)
	fmt.Printf("hit rate: %f\n", hit_rate)

	if hit_rate < 0.9 {
		t.Fatalf("hit rate is %f, too low", hit_rate)
	}
}
