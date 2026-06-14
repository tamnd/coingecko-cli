package coingecko

import (
	"testing"
)

// These tests are offline: they exercise the URI driver's pure string functions,
// which need no network. The client's HTTP behaviour is covered in coingecko_test.go.

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "coingecko" {
		t.Errorf("Scheme = %q, want coingecko", info.Scheme)
	}
	if len(info.Hosts) == 0 {
		t.Error("Hosts is empty")
	}
	if info.Identity.Binary != "coingecko" {
		t.Errorf("Identity.Binary = %q, want coingecko", info.Identity.Binary)
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		in  string
		typ string
		id  string
	}{
		{"bitcoin", "coin", "bitcoin"},
		{"ethereum", "coin", "ethereum"},
	}
	for _, tc := range cases {
		typ, id, err := Domain{}.Classify(tc.in)
		if err != nil || typ != tc.typ || id != tc.id {
			t.Errorf("Classify(%q) = (%q, %q, %v), want (%q, %q, nil)",
				tc.in, typ, id, err, tc.typ, tc.id)
		}
	}
}

func TestClassifyEmpty(t *testing.T) {
	_, _, err := Domain{}.Classify("")
	if err == nil {
		t.Error("Classify(\"\") should return an error")
	}
}

func TestLocate(t *testing.T) {
	got, err := Domain{}.Locate("coin", "bitcoin")
	want := "https://www.coingecko.com/en/coins/bitcoin"
	if err != nil || got != want {
		t.Errorf("Locate = (%q, %v), want (%q, nil)", got, err, want)
	}
}

func TestLocateUnknownType(t *testing.T) {
	_, err := Domain{}.Locate("nft", "bored-ape")
	if err == nil {
		t.Error("Locate with unknown type should return an error")
	}
}
