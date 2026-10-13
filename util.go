package main

import (
	"errors"
	"strconv"
)

func itoa(i int) string   { return strconv.Itoa(i) }
func errf(s string) error { return errors.New(s) }
