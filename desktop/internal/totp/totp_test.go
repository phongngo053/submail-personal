package totp

import (
	"encoding/base32"
	"testing"
	"time"
)

// TestCodeRFC6238Vectors checks the standard SHA-1 test vectors from RFC 6238
// Appendix B. The shared secret is the ASCII string "12345678901234567890".
func TestCodeRFC6238Vectors(t *testing.T) {
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte("12345678901234567890"))

	cases := []struct {
		name string
		unix int64
		want string
	}{
		{"T=59", 59, "287082"},
		{"T=1111111109", 1111111109, "081804"},
		{"T=1111111111", 1111111111, "050471"},
		{"T=1234567890", 1234567890, "005924"},
		{"T=2000000000", 2000000000, "279037"},
		{"T=20000000000", 20000000000, "353130"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Code(secret, time.Unix(tc.unix, 0).UTC())
			if err != nil {
				t.Fatalf("Code() error = %v", err)
			}
			if got != tc.want {
				t.Errorf("Code() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestDecodeBase32 checks decoding against known Google Authenticator secrets.
func TestDecodeBase32(t *testing.T) {
	cases := []struct {
		secret string
		want   string
	}{
		{"JBSWY3DPEHPK3PXP", "48656c6c6f21deadbeef"},
		{"KRSXG5CTMVRXEZLU", "54657374536563726574"},
		{"MZXW6YTBOI5GM2LE", "666f6f6261723a666964"},
		{"NBSWY3DPO5XXE3DE", "68656c6c6f776f726c64"},
	}

	for _, tc := range cases {
		t.Run(tc.secret, func(t *testing.T) {
			got, err := DecodeBase32(tc.secret)
			if err != nil {
				t.Fatalf("DecodeBase32() error = %v", err)
			}
			if hexEncode(got) != tc.want {
				t.Errorf("DecodeBase32() = %s, want %s", hexEncode(got), tc.want)
			}
		})
	}
}

// TestDecodeBase32IgnoresFormatting checks that spaces, dashes, padding and
// lowercase are tolerated, since users paste secrets by hand.
func TestDecodeBase32IgnoresFormatting(t *testing.T) {
	want := "48656c6c6f21deadbeef"
	for _, secret := range []string{
		"jbsw y3dp ehpk 3pxp",
		"JBSW-Y3DP-EHPK-3PXP",
		"JBSWY3DPEHPK3PXP====",
		" JBSWY3DPEHPK3PXP ",
	} {
		got, err := DecodeBase32(secret)
		if err != nil {
			t.Fatalf("DecodeBase32(%q) error = %v", secret, err)
		}
		if hexEncode(got) != want {
			t.Errorf("DecodeBase32(%q) = %s, want %s", secret, hexEncode(got), want)
		}
	}
}

// TestDecodeBase32RejectsGarbage checks that a secret with no decodable
// characters is an error rather than a silent empty key.
func TestDecodeBase32RejectsGarbage(t *testing.T) {
	if _, err := DecodeBase32("!!!???"); err == nil {
		t.Fatal("DecodeBase32() expected error for invalid secret")
	}
}

// TestCodeRejectsInvalidSecret checks the error path for an unusable secret.
func TestCodeRejectsInvalidSecret(t *testing.T) {
	if _, err := Code("!!!", time.Now()); err == nil {
		t.Fatal("Code() expected error for invalid secret")
	}
}

// TestRemaining checks the countdown helper at a time-step boundary.
func TestRemaining(t *testing.T) {
	if got := Remaining(time.Unix(59, 0)); got != 1 {
		t.Errorf("Remaining(59) = %d, want 1", got)
	}
	if got := Remaining(time.Unix(60, 0)); got != 30 {
		t.Errorf("Remaining(60) = %d, want 30", got)
	}
}

// hexEncode formats bytes as lowercase hex without pulling in encoding/hex
// at every call site.
func hexEncode(b []byte) string {
	const hexDigits = "0123456789abcdef"
	out := make([]byte, 0, len(b)*2)
	for _, c := range b {
		out = append(out, hexDigits[c>>4], hexDigits[c&0x0F])
	}
	return string(out)
}
