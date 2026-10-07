package fingerprint

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"strings"
	"unicode"
)

type SimHash struct{}

func (SimHash) Kind() Kind { return KindSimHash }

func (SimHash) Compute(ctx context.Context, root string) (Signal, error) {
	_ = ctx
	values, err := fileHashes(root)
	if err != nil {
		return Signal{}, err
	}
	var tokens []string
	for _, v := range values {
		tokens = append(tokens, hex.EncodeToString(v[:8]))
	}
	bits := simhash64(tokens)
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, bits)
	return Signal{
		Kind:   KindSimHash,
		Value:  hex.EncodeToString(buf),
		Weight: 8,
		Ready:  true,
		Note:   "64-bit SimHash of file digest tokens. Hamming distance is the comparison, not a verdict.",
		Details: map[string]any{"tokens": len(tokens)},
	}, nil
}

func simhash64(tokens []string) uint64 {
	var acc [64]int
	for _, t := range tokens {
		h := fnv.New64a()
		_, _ = h.Write([]byte(t))
		v := h.Sum64()
		for i := 0; i < 64; i++ {
			if v&(1<<i) != 0 {
				acc[i]++
			} else {
				acc[i]--
			}
		}
	}
	var out uint64
	for i := 0; i < 64; i++ {
		if acc[i] >= 0 {
			out |= 1 << i
		}
	}
	return out
}

func Hamming(a, b string) (int, error) {
	da, err := hex.DecodeString(a)
	if err != nil {
		return 0, err
	}
	db, err := hex.DecodeString(b)
	if err != nil {
		return 0, err
	}
	if len(da) != len(db) {
		return 0, fmt.Errorf("simhash width mismatch")
	}
	n := 0
	for i := range da {
		x := da[i] ^ db[i]
		for x > 0 {
			n += int(x & 1)
			x >>= 1
		}
	}
	return n, nil
}

func tokenizeText(s string) []string {
	s = strings.ToLower(s)
	var b strings.Builder
	var out []string
	flush := func() {
		if b.Len() >= 3 {
			out = append(out, b.String())
		}
		b.Reset()
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			b.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	return out
}
