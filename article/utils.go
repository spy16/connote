package article

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseTime parses the given time-specification and returns.
// ds can be 'today', 'yesterday', 'tomorrow', an int value
// representing offset in days (-1 for yesterday, +1 for tomorrow etc.) or a date
// in 02/01/2006 format.
func ParseTime(ds string) (time.Time, error) {
	var t time.Time

	switch ds {
	case "today", "now", "0":
		t = time.Now()

	case "yesterday", "yday", "-1":
		t = time.Now().AddDate(0, 0, -1)

	case "tomorrow", "tom", "1", "+1":
		t = time.Now().AddDate(0, 0, 1)

	default:
		daysOffset, err := strconv.ParseInt(ds, 10, 32)
		if err == nil {
			t = time.Now().AddDate(0, 0, int(daysOffset))
		} else {
			ds = strings.Replace(ds, "/", "-", -1)

			ts, err := time.Parse("02-01-2006", ds)
			if err != nil {
				return t, fmt.Errorf("unknown time-string: %s", ds)
			}
			t = ts
		}

	}

	return t, nil
}
