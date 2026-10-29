package tools

import (
	"strconv"

	"github.com/tmih06/ompig/internal/jsnumber"
)

func itoa(value int) string { return strconv.Itoa(value) }

func ftoa(value float64) string { return jsnumber.String(value) }
