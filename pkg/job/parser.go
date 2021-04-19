package job

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

var isNotDigit = func(c rune) bool { return c < '0' || c > '9' }

// NewSchedule constructs schedule from spec
func parseYearSpec(spec string) (result []int, err error) {
	ss := strings.Split(spec, ",")
	for _, entry := range ss {
		entry = strings.Trim(entry, " ")
		switch {
		case entry == "*":
			// return empty list, which means any year, see job.Next for year handling
			return
		case strings.ContainsAny(entry, "-"):
			r, err := convertRange(entry)
			if err != nil {
				return []int{}, err
			}
			result = append(result, r...)
		case strings.IndexFunc(entry, isNotDigit) == -1:
			n, err := convertString(entry)
			if err != nil {
				return []int{}, err
			}
			result = append(result, n)
		default:
			err = fmt.Errorf("Unsupported schedule config time entry for Year: ' %s'", entry)
			return
		}
	}
	// Sanitize - remove dups and sort
	result = sliceRemoveDuplicates(result)
	sort.Ints(result)
	return
}

func parseTimeSpec(s string, min, max int) (result []int, err error) {
	ss := strings.Split(s, ",")
	for _, entry := range ss {
		entry = strings.Trim(entry, " ")
		switch {
		case entry == "*":
			result, err = makeRange(min, max)
			if err != nil {
				return []int{}, err
			}
		case strings.ContainsAny(entry, "-"):
			r, err := convertRange(entry)
			if err != nil {
				return []int{}, err
			}
			result = append(result, r...)
		case strings.ContainsAny(entry, "/"):
			s, err := convertSlashed(entry, min, max)
			if err != nil {
				return []int{}, err
			}
			result = append(result, s...)
		case strings.IndexFunc(entry, isNotDigit) == -1:
			n, err := convertString(entry)
			if err != nil {
				return []int{}, err
			}
			result = append(result, n)
		default:
			return []int{}, fmt.Errorf("unsupported schedule spec time entry: ' %s'", entry)
		}
	}
	// Sanitize - remove dups and sort
	result = sliceRemoveDuplicates(result)
	sort.Ints(result)
	if result[0] < min {
		return []int{}, fmt.Errorf("entry %d cannot be less that min: %d", result[0], min)
	}
	if result[len(result)-1] > max {
		return []int{}, fmt.Errorf("entry %d cannot be more than max: %d", result[len(result)-1], max)
	}
	return

}
func sliceRemoveDuplicates(s []int) []int {
	seen := make(map[int]struct{}, len(s))
	j := 0
	for _, v := range s {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		s[j] = v
		j++
	}
	return s[:j]
}

func makeRange(min, max int) (result []int, err error) {
	if min > max {
		err = fmt.Errorf("range %d - %d is incorrect", min, max)
		return
	}
	result = make([]int, max-min+1)
	for i := range result {
		result[i] = min + i
	}
	return
}
func convertRange(s string) (result []int, err error) {
	split := strings.Split(s, "-")
	f0, err := strconv.ParseInt(split[0], 10, 32)
	if err != nil {
		return
	}
	f1, err := strconv.ParseInt(split[1], 10, 32)
	if err != nil {
		return
	}
	return makeRange(int(f0), int(f1))
}
func convertEnum(s string) (result []int, err error) {
	split := strings.Split(s, ",")
	result = make([]int, len(split))
	for key, val := range split {
		valNum, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			return []int{}, err
		}
		result[key] = int(valNum)
	}
	return
}
func convertString(s string) (result int, err error) {
	c, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return
	}
	result = int(c)
	return
}
func convertSlashed(s string, min, max int) (result []int, err error) {
	split := strings.Split(s, "/")
	var first int
	if split[0] == "*" || split[0] == "" {
		first = min
	} else {
		num, err := strconv.ParseInt(split[0], 10, 32)
		if err != nil {
			err = fmt.Errorf("parse error for ?/num spec: %w", err)
			return result, err
		}
		first = int(num)
	}
	if first > max {
		err = fmt.Errorf("first member %d is bigger than max %d", first, max)
		return
	}

	num, err := strconv.ParseInt(split[1], 10, 32)
	if err != nil {
		return []int{}, err
	}
	step := int(num)
	// /5 means min, min+5,..
	for i := first; i <= max; i += step {
		result = append(result, i)
	}
	return
}
