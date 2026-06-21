package service

import "testing"

func TestNormalizeUserSecurityEventToken(t *testing.T) {
	got := normalizeUserSecurityEventToken("  OAuth_Register  ")
	if got != "oauth_register" {
		t.Fatalf("normalize token = %q, want oauth_register", got)
	}

	long := normalizeUserSecurityEventToken("abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnop")
	if len(long) != 64 {
		t.Fatalf("normalized token length = %d, want 64", len(long))
	}
}

func TestTrimSecurityEventString(t *testing.T) {
	if got := trimSecurityEventString("  abc  ", 10); got != "abc" {
		t.Fatalf("trim string = %q, want abc", got)
	}
	if got := trimSecurityEventString("abcdef", 3); got != "abc" {
		t.Fatalf("trim string = %q, want abc", got)
	}
}
