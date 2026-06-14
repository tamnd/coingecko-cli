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

func TestClassifyCoin(t *testing.T) {
	cases := []struct {
		in  string
		typ string
		id  string
	}{
		{"bitcoin", "coin", "bitcoin"},
		{"ethereum", "coin", "ethereum"},
		{"Bitcoin", "coin", "bitcoin"},
		{"SOLANA", "coin", "solana"},
	}
	for _, tc := range cases {
		typ, id, err := Domain{}.Classify(tc.in)
		if err != nil {
			t.Errorf("Classify(%q) unexpected error: %v", tc.in, err)
			continue
		}
		if typ != tc.typ {
			t.Errorf("Classify(%q) type = %q, want %q", tc.in, typ, tc.typ)
		}
		if id != tc.id {
			t.Errorf("Classify(%q) id = %q, want %q", tc.in, id, tc.id)
		}
	}
}

func TestClassifyIDs(t *testing.T) {
	cases := []string{
		"bitcoin,ethereum",
		"bitcoin,ethereum,solana",
	}
	for _, tc := range cases {
		typ, id, err := Domain{}.Classify(tc)
		if err != nil {
			t.Errorf("Classify(%q) unexpected error: %v", tc, err)
			continue
		}
		if typ != "ids" {
			t.Errorf("Classify(%q) type = %q, want ids", tc, typ)
		}
		if id != tc {
			t.Errorf("Classify(%q) id = %q, want %q", tc, id, tc)
		}
	}
}

func TestClassifyEmpty(t *testing.T) {
	_, _, err := Domain{}.Classify("")
	if err == nil {
		t.Error("Classify(\"\") should return an error")
	}
}

func TestLocateCoin(t *testing.T) {
	got, err := Domain{}.Locate("coin", "bitcoin")
	want := "https://www.coingecko.com/en/coins/bitcoin"
	if err != nil || got != want {
		t.Errorf("Locate(coin, bitcoin) = (%q, %v), want (%q, nil)", got, err, want)
	}
}

func TestLocateIDs(t *testing.T) {
	got, err := Domain{}.Locate("ids", "bitcoin,ethereum")
	if err != nil {
		t.Errorf("Locate(ids, ...) unexpected error: %v", err)
	}
	if got != "https://www.coingecko.com/en/coins/bitcoin" {
		t.Errorf("Locate(ids, bitcoin,ethereum) = %q, want URL for bitcoin", got)
	}
}

func TestLocateUnknownType(t *testing.T) {
	_, err := Domain{}.Locate("nft", "bored-ape")
	if err == nil {
		t.Error("Locate with unknown type should return an error")
	}
}
