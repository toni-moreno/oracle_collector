package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// WaitAlignForNextCycle waiths untile a next cycle begins aligned with second 00 of each minute
func WaitAlignForNextCycle(SecPeriod int, l *logrus.Logger) {
	i := int64(time.Duration(SecPeriod) * time.Second)
	remain := i - (time.Now().UnixNano() % i)
	l.Infof("Waiting %s to round until nearest interval... (Cycle = %d seconds)", time.Duration(remain).String(), SecPeriod)
	time.Sleep(time.Duration(remain))
}

// RemoveDuplicatesUnordered removes duplicated elements in the array string
func RemoveDuplicatesUnordered(elements []string) []string {
	encountered := map[string]bool{}

	// Create a map of all unique elements.
	for v := range elements {
		encountered[elements[v]] = true
	}

	// Place all keys from the map into a slice.
	result := []string{}
	for key := range encountered {
		result = append(result, key)
	}
	return result
}

// DiffSlice return de Difference between two Slices
func SliceDiff(X, Y []string) []string {
	diff := []string{}
	vals := map[string]struct{}{}

	for _, x := range Y {
		vals[x] = struct{}{}
	}

	for _, x := range X {
		if _, ok := vals[x]; !ok {
			diff = append(diff, x)
		}
	}

	return diff
}

// https://stackoverflow.com/questions/44956031/how-to-get-intersection-of-two-slice-in-golang
func SliceIntersect(a []string, b []string) (inter []string) {
	// interacting on the smallest list first can potentailly be faster...but not by much, worse case is the same
	low, high := a, b
	if len(a) > len(b) {
		low = b
		high = a
	}

	done := false
	for i, l := range low {
		for j, h := range high {
			// get future index values
			f1 := i + 1
			f2 := j + 1
			if l == h {
				inter = append(inter, h)
				if f1 < len(low) && f2 < len(high) {
					// if the future values aren't the same then that's the end of the intersection
					if low[f1] != high[f2] {
						done = true
					}
				}
				// we don't want to interate on the entire list everytime, so remove the parts we already looped on will make it faster each pass
				high = high[:j+copy(high[j:], high[j+1:])]
				break
			}
		}
		// nothing in the future so we are done
		if done {
			break
		}
	}
	return
}

func diffKeysInMap(X, Y map[string]string) map[string]string {
	diff := map[string]string{}

	for k, vK := range X {
		if _, ok := Y[k]; !ok {
			diff[k] = vK
		}
	}

	return diff
}

// DiffKeyValuesInMap does a diff key and values from 2 strings maps
func DiffKeyValuesInMap(X, Y map[string]string) map[string]string {
	diff := map[string]string{}

	for kX, vX := range X {
		if vY, ok := Y[kX]; !ok {
			// not exist
			diff[kX] = vX
		} else {
			// exist
			if vX != vY {
				// but value is different
				diff[kX] = vX
			}
		}
	}
	return diff
}

// CSV2IntArray CSV intenger array conversion
func CSV2IntArray(csv string) ([]int64, error) {
	var iarray []int64
	result := Splitter(csv, ",;|")
	for i, v := range result {
		vc, err := strconv.Atoi(v)
		if err != nil {
			return iarray, fmt.Errorf("Bad Format in CSV array item %d | value %s | Error %s", i, v, err)
		}
		iarray = append(iarray, int64(vc))
	}
	return iarray, nil
}

// Splitter multiple value split
func Splitter(s string, splits string) []string {
	m := make(map[rune]int)
	for _, r := range splits {
		m[r] = 1
	}

	splitter := func(r rune) bool {
		return m[r] == 1
	}

	return strings.FieldsFunc(s, splitter)
}

func TrimLeftChar(s string) string {
	for i := range s {
		if i > 0 {
			return s[i:]
		}
	}
	return s[:0]
}
