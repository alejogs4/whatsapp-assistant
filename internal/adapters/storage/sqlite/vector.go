package sqlite

import (
	"encoding/binary"
	"fmt"
	"math"
)

func encodeVector(v []float32) []byte {
	buf := make([]byte, len(v)*4)
	for i, f := range v {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}

func decodeVector(b []byte) ([]float32, error) {
	if len(b)%4 != 0 {
		return nil, fmt.Errorf("sqlite: invalid embedding blob length %d", len(b))
	}
	out := make([]float32, len(b)/4)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return out, nil
}

// cosineSimilarity computes cosine similarity between a and b safely.
// Returns 0 for mismatched dimensions or zero-norm vectors (never NaN).
func cosineSimilarity(a, b []float32) float32 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		fa, fb := float64(a[i]), float64(b[i])
		dot += fa * fb
		normA += fa * fa
		normB += fb * fb
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	sim := dot / (math.Sqrt(normA) * math.Sqrt(normB))
	if math.IsNaN(sim) || math.IsInf(sim, 0) {
		return 0
	}
	return float32(sim)
}
