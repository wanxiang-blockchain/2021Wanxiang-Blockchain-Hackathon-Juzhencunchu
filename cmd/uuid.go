package cmd

import (
	"fmt"
	"math/rand"
)

// GenerateRandomBytes is used to generate random bytes of given size.
func GenerateRandomBytes() ([]byte, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return nil, fmt.Errorf("failed to read random bytes: %v", err)
	}
	return buf, nil
}

// GenerateUUID is used to generate a random UUID
func GenerateUUID() (string, error) {
	buf, err := GenerateRandomBytes()
	if err != nil {
		return "", err
	}
	return FormatUUID(buf), nil
}

func FormatUUID(buf []byte) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		buf[0:4],
		buf[4:6],
		buf[6:8],
		buf[8:10],
		buf[10:16])
}
