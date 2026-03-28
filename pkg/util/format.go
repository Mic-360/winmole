package util

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"time"
)

func Clamp(value, low, high int) int {
	if low > high {
		low, high = high, low
	}
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func ClampFloat(value, low, high float64) float64 {
	if low > high {
		low, high = high, low
	}
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func FormatBytes(value int64) string {
	if value < 0 {
		value = 0
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	size := float64(value)
	index := 0
	for size >= 1024 && index < len(units)-1 {
		size /= 1024
		index++
	}
	if index == 0 {
		return fmt.Sprintf("%d %s", value, units[index])
	}
	return fmt.Sprintf("%.1f %s", size, units[index])
}

func FormatRate(bytesPerSecond float64) string {
	return fmt.Sprintf("%s/s", FormatBytes(int64(bytesPerSecond)))
}

func FormatPercent(value float64) string {
	return fmt.Sprintf("%.1f%%", value)
}

func FormatUptime(duration time.Duration) string {
	if duration < 0 {
		duration = 0
	}
	days := duration / (24 * time.Hour)
	duration -= days * 24 * time.Hour
	hours := duration / time.Hour
	duration -= hours * time.Hour
	minutes := duration / time.Minute
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	return fmt.Sprintf("%dh %dm", hours, minutes)
}

func ShortenPath(path string, max int) string {
	clean := filepath.Clean(path)
	if len(clean) <= max || max <= 4 {
		return clean
	}
	parts := strings.Split(clean, string(filepath.Separator))
	if len(parts) < 3 {
		return "..." + clean[len(clean)-max+3:]
	}
	short := parts[0] + string(filepath.Separator) + "..." + string(filepath.Separator) + strings.Join(parts[len(parts)-2:], string(filepath.Separator))
	if len(short) <= max {
		return short
	}
	return "..." + clean[len(clean)-max+3:]
}

func NormalizeWhitespace(input string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(input)), " ")
}

func TitleCase(input string) string {
	parts := strings.Fields(strings.TrimSpace(input))
	for index, part := range parts {
		if part == "" {
			continue
		}
		parts[index] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	return strings.Join(parts, " ")
}

func Ratio(value, total float64) float64 {
	if total <= 0 {
		return 0
	}
	return value / total
}

func Round(value float64, places int) float64 {
	power := math.Pow10(places)
	return math.Round(value*power) / power
}
