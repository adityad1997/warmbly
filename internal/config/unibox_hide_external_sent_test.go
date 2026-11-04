package config

import "testing"

func TestUniboxHideExternalSentEnabled(t *testing.T) {
	t.Setenv("UNIBOX_HIDE_EXTERNAL_SENT_ENABLED", "")
	if !UniboxHideExternalSentEnabled() {
		t.Fatal("unset must default on")
	}
	for _, tc := range []struct {
		raw  string
		want bool
	}{
		{"true", true},
		{"1", true},
		{"false", false},
		{"0", false},
		{"nope", true},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			t.Setenv("UNIBOX_HIDE_EXTERNAL_SENT_ENABLED", tc.raw)
			if got := UniboxHideExternalSentEnabled(); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
