package generator

import (
	"math/rand/v2"
	"strings"
)

const (
	hexChars      = "0123456789abcdef"
	updateChance  = 0.5
	maxNewPerTick = 3
)

// randomHex генерирует строку из n случайных hex-символов с префиксом
func randomHex(n int, prefix string) string {
	var sb strings.Builder
	sb.Grow(len(prefix) + n)
	sb.WriteString(prefix)
	for i := 0; i < n; i++ {
		sb.WriteByte(hexChars[rand.IntN(len(hexChars))])
	}

	return sb.String()
}

// round2 округляет float64 до 2 знаков после запятой
func round2(v float64) float64 {
	return float64(int64(v*100)) / 100
}
