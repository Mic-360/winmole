package util

import "strings"

var sparkChars = []rune("▁▂▃▄▅▆▇█")

func Sparkline(values []float64, width int) string {
	if width <= 0 {
		return ""
	}
	if len(values) == 0 {
		return strings.Repeat("·", width)
	}
	trimmed := values
	if len(trimmed) > width {
		trimmed = trimmed[len(trimmed)-width:]
	}
	if len(trimmed) < width {
		padding := make([]float64, width-len(trimmed))
		trimmed = append(padding, trimmed...)
	}
	maxValue := 0.0
	for _, value := range trimmed {
		if value > maxValue {
			maxValue = value
		}
	}
	if maxValue <= 0 {
		return strings.Repeat("▁", width)
	}
	var builder strings.Builder
	for _, value := range trimmed {
		index := int((value / maxValue) * float64(len(sparkChars)-1))
		if index < 0 {
			index = 0
		}
		if index >= len(sparkChars) {
			index = len(sparkChars) - 1
		}
		builder.WriteRune(sparkChars[index])
	}
	return builder.String()
}
