package display

import "testing"

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		name string
		v    int64
		want string
	}{
		{name: "Zero", v: 0, want: "0"},
		{name: "SingleDigit", v: 5, want: "5"},
		{name: "ThreeDigits", v: 999, want: "999"},
		{name: "Thousands", v: 12547, want: "12 547"},
		{name: "ExactThousand", v: 1000, want: "1 000"},
		{name: "Million", v: 1000000, want: "1 000 000"},
		{name: "Negative", v: -1234, want: "-1 234"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatNumber(tt.v); got != tt.want {
				t.Fatalf("FormatNumber(%d) = %q, want %q", tt.v, got, tt.want)
			}
		})
	}
}

func TestFormatNumberInt(t *testing.T) {
	if got := FormatNumber(12547); got != "12 547" {
		t.Fatalf("FormatNumber(int) = %q, want %q", got, "12 547")
	}
}
