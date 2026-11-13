// Package totp implements RFC 6238 time-based one-time passwords using
// HMAC-SHA1 over a Base32-encoded secret, matching Google Authenticator.
package totp

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"strconv"
	"strings"
	"time"
)

// Period is the TOTP time step in seconds.
const Period = 30

// Digits is the number of digits in a generated code.
const Digits = 6

// ErrInvalidSecret is returned when the secret contains no decodable Base32 data.
var ErrInvalidSecret = errors.New("totp: secret is not valid Base32")

// DecodeBase32 converts a Base32 secret (ignoring spaces, dashes and padding)
// into its raw bytes.
func DecodeBase32(secret string) ([]byte, error) {
	cleaned := strings.ToUpper(strings.TrimSpace(secret))
	cleaned = strings.NewReplacer(" ", "", "-", "", "=", "", "\t", "").Replace(cleaned)

	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
	var bits, value uint32
	out := make([]byte, 0, len(cleaned))

	for i := 0; i < len(cleaned); i++ {
		idx := strings.IndexByte(alphabet, cleaned[i])
		if idx < 0 {
			continue
		}
		value = (value << 5) | uint32(idx)
		bits += 5
		if bits >= 8 {
			out = append(out, byte((value>>(bits-8))&0xFF))
			bits -= 8
		}
	}
	if len(out) == 0 {
		return nil, ErrInvalidSecret
	}
	return out, nil
}

// Code computes the TOTP code for the given Base32 secret at time t.
// It returns the zero-padded digit string.
func Code(secret string, t time.Time) (string, error) {
	key, err := DecodeBase32(secret)
	if err != nil {
		return "", err
	}

	counter := make([]byte, 8)
	binary.BigEndian.PutUint64(counter, uint64(t.Unix())/Period)

	mac := hmac.New(sha1.New, key)
	mac.Write(counter)
	sum := mac.Sum(nil)

	offset := sum[len(sum)-1] & 0x0F
	value := (uint32(sum[offset]&0x7F) << 24) |
		(uint32(sum[offset+1]) << 16) |
		(uint32(sum[offset+2]) << 8) |
		uint32(sum[offset+3])

	mod := uint32(1)
	for i := 0; i < Digits; i++ {
		mod *= 10
	}
	return fmtPad(strconv.FormatUint(uint64(value%mod), 10)), nil
}

// Remaining returns the number of seconds left in the current time step.
func Remaining(t time.Time) int {
	return Period - int(t.Unix()%Period)
}

// fmtPad left-pads s with zeros to the configured digit width.
func fmtPad(s string) string {
	for len(s) < Digits {
		s = "0" + s
	}
	return s
}
