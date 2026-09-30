package db

import (
	"fmt"
	"math"
)

func floatValue(value *float64) float64 {
	if value == nil {
		return 0
	}

	return *value
}

func sqliteInteger(name string, value uint64) (int64, error) {
	if value > math.MaxInt64 {
		return 0, fmt.Errorf("%s exceeds SQLite integer range", name)
	}

	return int64(value), nil
}
