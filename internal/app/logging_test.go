package app

import "testing"

func TestParseAppLogLevel(t *testing.T) {
	cases := []struct {
		input string
		want  appLogLevel
	}{
		{"DEBUG", levelDebug}, {"info", levelInfo}, {" WARN ", levelWarn},
		{"WARNING", levelWarn}, {"ERROR", levelError}, {"", levelInfo}, {"invalid", levelInfo},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			if got := parseAppLogLevel(tc.input); got != tc.want {
				t.Fatalf("parseAppLogLevel(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}
