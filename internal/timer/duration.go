package timer

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseDur accepts "30" (minutes, fractional OK) or "mm:ss".
func ParseDur(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}
	if i := strings.IndexByte(s, ':'); i >= 0 {
		mm, err1 := strconv.Atoi(strings.TrimSpace(s[:i]))
		ss, err2 := strconv.Atoi(strings.TrimSpace(s[i+1:]))
		if err1 != nil || err2 != nil || mm < 0 || ss < 0 || ss > 59 {
			return 0, fmt.Errorf("bad duration %q (use minutes or mm:ss)", s)
		}
		return time.Duration(mm)*time.Minute + time.Duration(ss)*time.Second, nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f <= 0 {
		return 0, fmt.Errorf("bad duration %q (use minutes or mm:ss)", s)
	}
	return time.Duration(f * float64(time.Minute)), nil
}
