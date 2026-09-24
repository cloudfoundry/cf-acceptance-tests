package cfcmdtrace

import "testing"

func TestSignatureNormalizesCatsNames(t *testing.T) {
	a := []string{"push", "CATS-3-APP-a1b2c3d4e5f6a7b8"}
	b := []string{"push", "CATS-7-APP-9f8e7d6c5b4a3210"}
	if signature(a, "CATS") != signature(b, "CATS") {
		t.Fatalf("random names not collapsed:\n%q\n%q", signature(a, "CATS"), signature(b, "CATS"))
	}
}

func TestSignatureNormalizesGuidHexAndSuffix(t *testing.T) {
	cases := [][]string{
		{"delete", "app", "12345678-1234-1234-1234-123456789abc"},
		{"delete", "app", "deadbeefdeadbeefdeadbeef"},
		{"delete", "app-4821"},
	}
	want := []string{
		"delete app <GUID>",
		"delete app <HEX>",
		"delete app-<N>",
	}
	for i, c := range cases {
		if got := signature(c, "CATS"); got != want[i] {
			t.Fatalf("case %d: got %q want %q", i, got, want[i])
		}
	}
}

func TestSignatureRespectsConfiguredPrefix(t *testing.T) {
	a := []string{"push", "ACME-2-APP-a1b2c3d4e5f6a7b8"}
	if got := signature(a, "ACME"); got != "push <PREFIX>-N-<RESOURCE>-<RAND>" {
		t.Fatalf("configured prefix not applied: %q", got)
	}
}
