package common

import (
	"crypto/rand"
	"fmt"
)

const NilUUID = "00000000-0000-0000-0000-000000000000"

func IsNilUUID(s string) bool {
	return s == NilUUID
}

func GenerateUUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
