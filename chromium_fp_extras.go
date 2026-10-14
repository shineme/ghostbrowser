package main

import (
	crand "crypto/rand"
	"encoding/binary"
	"fmt"
)

// Fingerprint extras for the open-source kernel: noise seed regeneration.
// (Media device ID aliases live in the page script, see pageJS "mida".)

// newNoiseSeed returns a fresh random noise seed in [1, 2^40), the same range
// GenerateFingerprint uses, so it stays exact as a JS number.
func newNoiseSeed() int64 {
	var b [8]byte
	_, _ = crand.Read(b[:])
	return int64(binary.LittleEndian.Uint64(b[:])%((1<<40)-1)) + 1
}

// seedShortID is the 8-hex label the UI shows for a seed.
func seedShortID(seed int64) string {
	s := fmt.Sprintf("%08x", uint64(seed))
	return s[:8]
}

// regenerateSeed gives a saved profile a new noise seed. Only the seed changes:
// cookies, storage, proxy and the rest of the fingerprint are untouched.
func (s *Store) regenerateSeed(id string) (int64, error) {
	p, ok := s.GetProfile(id)
	if !ok {
		return 0, fmt.Errorf("环境不存在")
	}
	old := p.Fingerprint.NoiseSeed
	seed := newNoiseSeed()
	for seed == old {
		seed = newNoiseSeed()
	}
	p.Fingerprint.NoiseSeed = seed
	s.PutProfile(*p)
	return seed, nil
}
