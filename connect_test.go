package main

import (
	"testing"
)

// cgo not allowed in test
func TestSocket(t *testing.T) { testSocket(t) }
