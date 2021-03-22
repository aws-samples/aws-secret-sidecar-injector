package main

import "testing"

func TestWriteOutput(t *testing.T) {
	err := writeOutput("super secret secret", "")
	if err != nil {
		t.Errorf("an error occurred: %v", err)
	}
	err = writeOutput("another secret", "/secrets/aaaaa")
	if err != nil {
		t.Errorf("an error occurred: %v", err)
	}
}
