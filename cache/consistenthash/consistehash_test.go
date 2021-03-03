package consistenthash

import (
	"strconv"
	"testing"
)

func TestConsistenthash(t *testing.T) {
	chash := New(3, func(key []byte) uint32 {
		i, _ := strconv.Atoi(string(key))
		return uint32(i)
	})

	chash.Add("6","4","2")
	for _,v := range chash.keys {
		println(v)
	}
	testCases := map[string]string{
		"2":  "2",
		"11": "2",
		"23": "4",
		"27": "2",
	}

	for k,v := range testCases {
		if chash.Get(k) !=v{
			t.Errorf("Asking for %s, should have yielded %s", k, v)
		}
	}

	chash.Add("8")

	testCases["27"]="8"

	for k,v := range testCases {
		if chash.Get(k) !=v{
			t.Errorf("Asking for %s, should have yielded %s", k, v)
		}
	}

}
