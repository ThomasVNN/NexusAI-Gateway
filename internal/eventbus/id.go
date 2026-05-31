package eventbus

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

// ULID-style ID generator for time-sortable unique IDs
// Format: timestamp (10 chars) + random (16 chars) = 26 char string

const (
	ulidAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	ulidLength   = 26
	timeLen      = 10
	randomLen    = 16
)

// Global entropy pool and mutex for thread-safe ID generation
var (
	entropyPool  []byte
	entropyMutex sync.Mutex
)

func init() {
	// Pre-seed the entropy pool
	entropyPool = make([]byte, 32)
	if _, err := rand.Read(entropyPool); err != nil {
		panic("failed to initialize entropy pool: " + err.Error())
	}
}

// generateEventID creates a ULID-style sortable unique identifier
// Format: TTTTTTTTTT + RRRRRRRRRRRRRRRR (time + random, Crockford Base32 encoded)
// This format is lexicographically sortable by creation time while maintaining uniqueness
func generateEventID() string {
	entropyMutex.Lock()
	defer entropyMutex.Unlock()

	now := time.Now().UnixMilli()

	// Encode timestamp (10 characters)
	timestamp := encodeTimestamp(now)

	// Encode random bytes (16 characters)
	random := encodeRandom()

	return timestamp + random
}

// encodeTimestamp encodes milliseconds since Unix epoch into Crockford Base32
func encodeTimestamp(ms int64) string {
	var chars [timeLen]byte
	for i := timeLen - 1; i >= 0; i-- {
		idx := ms & 0x1F // 31 in base32
		chars[i] = ulidAlphabet[idx]
		ms >>= 5
	}
	return string(chars[:])
}

// encodeRandom generates random characters from the ULID alphabet
func encodeRandom() string {
	result := make([]byte, randomLen)

	// Refill pool if needed
	if len(entropyPool) < randomLen {
		newPool := make([]byte, 32)
		if _, err := rand.Read(newPool); err != nil {
			panic("failed to read entropy: " + err.Error())
		}
		entropyPool = newPool
	}

	copy(result, entropyPool[:randomLen])
	entropyPool = entropyPool[randomLen:]

	// Encode to Crockford Base32 alphabet
	for i := range result {
		result[i] = ulidAlphabet[int(result[i])&0x1F]
	}

	return string(result)
}

// ParseEventID extracts the timestamp component from an event ID
func ParseEventID(id string) (time.Time, error) {
	if len(id) < timeLen {
		return time.Time{}, fmt.Errorf("invalid event ID: too short")
	}

	timestampStr := id[:timeLen]

	// Decode timestamp
	var ms int64
	for _, c := range timestampStr {
		idx := indexBase32(byte(c))
		if idx < 0 {
			return time.Time{}, fmt.Errorf("invalid character in event ID: %c", c)
		}
		ms = (ms << 5) | int64(idx)
	}

	return time.UnixMilli(ms), nil
}

// indexBase32 finds the index of a character in the ULID alphabet
func indexBase32(c byte) int {
	c = toUpper(c)
	for i, a := range ulidAlphabet {
		if byte(a) == c {
			return i
		}
	}
	return -1
}

// toUpper converts lowercase to uppercase
func toUpper(c byte) byte {
	if c >= 'a' && c <= 'z' {
		return c - 32
	}
	return c
}

// GenerateClientID creates a unique identifier for clients/subscriptions
func GenerateClientID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("failed to generate client ID: " + err.Error())
	}
	return base64.URLEncoding.EncodeToString(b)[:22]
}

// GenerateSubscriptionID creates a unique subscription identifier
func GenerateSubscriptionID() string {
	return "sub_" + generateEventID()
}
