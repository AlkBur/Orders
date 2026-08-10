package app

import "testing"

func TestStatusTextColor(t *testing.T) {
	tests := []struct {
		color string
		want  string
	}{
		{"#ff0000", "#000000"}, // красный — тёмный текст
		{"#000000", "#ffffff"}, // чёрный — белый текст
		{"#ffffff", "#000000"}, // белый — чёрный текст
		{"#336699", "#ffffff"}, // синий — белый текст
		{"#ffff00", "#000000"}, // жёлтый — чёрный текст
	}
	for _, tt := range tests {
		if got := statusTextColor(tt.color); got != tt.want {
			t.Errorf("statusTextColor(%s) = %s, want %s", tt.color, got, tt.want)
		}
	}
}

func TestStatusTextColor_Invalid(t *testing.T) {
	tests := []string{"", "ffffff", "#ff00", "#fffffff", "red", "rgb(0,0,0)"}
	for _, c := range tests {
		if got := statusTextColor(c); got != "" {
			t.Errorf("statusTextColor(%q) = %q, want \"\"", c, got)
		}
	}
}

func TestStatusColorIfValid(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"#123456", "#123456"},
		{"#ABCDEF", "#ABCDEF"},
		{"#abcdef", "#abcdef"},
		{"", ""},
		{"123456", ""},
		{"#12345", ""},
		{"#1234567", ""},
		{"red", ""},
		{"rgb(0,0,0)", ""},
	}
	for _, tt := range tests {
		if got := statusColorIfValid(tt.in); got != tt.want {
			t.Errorf("statusColorIfValid(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestValidStatusColor_FormatBounds(t *testing.T) {
	valid := []string{"#123456", "#ABCDEF", "#abcdef", "#000000", "#FFFFFF"}
	for _, c := range valid {
		if !validStatusColor(c) {
			t.Errorf("validStatusColor(%q) = false, want true", c)
		}
	}
	invalid := []string{"", "123456", "#12345", "#1234567", "red", "rgb(0,0,0)", "#12 345", "##12345"}
	for _, c := range invalid {
		if validStatusColor(c) {
			t.Errorf("validStatusColor(%q) = true, want false", c)
		}
	}
}
