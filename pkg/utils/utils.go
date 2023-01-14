package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/exp/constraints"
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
func SliceDiff(slice1 []string, slice2 []string) []string {
	var diff []string

	for _, s1 := range slice1 {
		found := false
		for _, s2 := range slice2 {
			if s1 == s2 {
				found = true
				break
			}
		}
		// String not found. We add it to return slice
		if !found {
			diff = append(diff, s1)
		}
	}

	return diff
}

func SliceIntersect[T constraints.Ordered](pS ...[]T) []T {
	hash := make(map[T]*int) // value, counter
	result := make([]T, 0)
	for _, slice := range pS {
		duplicationHash := make(map[T]bool) // duplication checking for individual slice
		for _, value := range slice {
			if _, isDup := duplicationHash[value]; !isDup { // is not duplicated in slice
				if counter := hash[value]; counter != nil { // is found in hash counter map
					if *counter++; *counter >= len(pS) { // is found in every slice
						result = append(result, value)
					}
				} else { // not found in hash counter map
					i := 1
					hash[value] = &i
				}
				duplicationHash[value] = true
			}
		}
	}
	return result
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
