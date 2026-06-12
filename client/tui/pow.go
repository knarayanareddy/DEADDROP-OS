package tui

import (
	"crypto/sha256"
	"fmt"
	"strconv"
	"strings"
)

func solvePow(endpoint, challenge string) (string, bool) {
	parts := strings.Split(challenge, ":")
	if len(parts) != 4 {
		return "", false
	}
	difficulty, err := strconv.Atoi(parts[2])
	if err != nil || difficulty < 1 || difficulty > 32 {
		return "", false
	}
	prefix := endpoint + ":" + parts[0] + ":" + parts[1] + ":" + parts[2] + ":"
	for counter := uint64(0); ; counter++ {
		c := strconv.FormatUint(counter, 10)
		sum := sha256.Sum256([]byte(prefix + c))
		if leadingZeroBits(sum[:]) >= difficulty {
			return fmt.Sprintf("%s:%s", challenge, c), true
		}
	}
}

func leadingZeroBits(b []byte) int {
	count := 0
	for _, v := range b {
		if v == 0 {
			count += 8
			continue
		}
		for i := 7; i >= 0; i-- {
			if v&(1<<uint(i)) == 0 {
				count++
			} else {
				return count
			}
		}
	}
	return count
}
