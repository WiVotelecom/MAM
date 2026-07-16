package uaparse

import "testing"

func TestParse(t *testing.T) {
	cases := []struct {
		name    string
		ua      string
		browser string
		os      string
		mobile  bool
	}{
		{
			name:    "chrome on windows",
			ua:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			browser: "Chrome",
			os:      "Windows 10/11",
		},
		{
			name:    "edge",
			ua:      "Mozilla/5.0 (Windows NT 10.0) AppleWebKit/537.36 Chrome/120 Safari/537.36 Edg/120.0.0.0",
			browser: "Microsoft Edge",
			os:      "Windows 10/11",
		},
		{
			name:    "firefox on linux",
			ua:      "Mozilla/5.0 (X11; Linux x86_64; rv:121.0) Gecko/20100101 Firefox/121.0",
			browser: "Firefox",
			os:      "Linux",
		},
		{
			name:    "safari on ios mobile",
			ua:      "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
			browser: "Safari",
			os:      "iOS",
			mobile:  true,
		},
		{
			name:    "curl",
			ua:      "curl/8.4.0",
			browser: "curl",
			os:      "Unknown",
		},
		{
			name:    "empty",
			ua:      "",
			browser: "Unknown",
			os:      "Unknown",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Parse(tc.ua)
			if got.Browser != tc.browser {
				t.Errorf("browser = %q, want %q", got.Browser, tc.browser)
			}
			if got.OS != tc.os {
				t.Errorf("os = %q, want %q", got.OS, tc.os)
			}
			if got.Mobile != tc.mobile {
				t.Errorf("mobile = %v, want %v", got.Mobile, tc.mobile)
			}
		})
	}
}
