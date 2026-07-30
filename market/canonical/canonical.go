// Package canonical provides deterministic ordering and checksum primitives for
// replayable market-calculation inputs.
package canonical

import (
	"crypto/sha256"
	"encoding/binary"
	"sort"
)

// SortedStrings returns a sorted copy of values. It never mutates caller input.
func SortedStrings(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}

// Checksum hashes an unambiguous length-prefixed sequence of canonical parts.
// Callers must supply values in their documented canonical order.
func Checksum(parts ...string) [sha256.Size]byte {
	hash := sha256.New()
	var length [8]byte
	for _, part := range parts {
		binary.BigEndian.PutUint64(length[:], uint64(len(part)))
		_, _ = hash.Write(length[:])
		_, _ = hash.Write([]byte(part))
	}
	var result [sha256.Size]byte
	copy(result[:], hash.Sum(nil))
	return result
}
