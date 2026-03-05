package update

import (
	"bytes"
	"testing"
)

func TestParseSemver(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		ok    bool
		major int
		minor int
		patch int
	}{
		{name: "with v", in: "v1.2.3", ok: true, major: 1, minor: 2, patch: 3},
		{name: "without v", in: "2.4.6", ok: true, major: 2, minor: 4, patch: 6},
		{name: "invalid", in: "dev", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseSemver(tt.in)
			if ok != tt.ok {
				t.Fatalf("ok mismatch: got %v want %v", ok, tt.ok)
			}
			if !ok {
				return
			}
			if got.major != tt.major || got.minor != tt.minor || got.patch != tt.patch {
				t.Fatalf("got %v", got)
			}
		})
	}
}

func TestCompareSemver(t *testing.T) {
	if compareSemver(semver{1, 2, 3}, semver{1, 2, 3}) != 0 {
		t.Fatal("expected equal")
	}
	if compareSemver(semver{1, 2, 4}, semver{1, 2, 3}) <= 0 {
		t.Fatal("expected greater")
	}
	if compareSemver(semver{1, 1, 9}, semver{1, 2, 0}) >= 0 {
		t.Fatal("expected smaller")
	}
}

func TestConfirmUpdate(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "empty yes", in: "\n", want: true},
		{name: "y", in: "y\n", want: true},
		{name: "yes", in: "yes\n", want: true},
		{name: "n", in: "n\n", want: false},
		{name: "other", in: "later\n", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := bytes.NewBufferString(tt.in)
			out := bytes.NewBuffer(nil)
			got := confirmUpdate(in, out)
			if got != tt.want {
				t.Fatalf("got %v want %v", got, tt.want)
			}
		})
	}
}
