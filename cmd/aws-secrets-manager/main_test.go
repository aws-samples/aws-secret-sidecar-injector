package main

import "testing"

func TestWriteOutput(t *testing.T) {
	err := writeOutput("super secret secret", "/tmp", "")
	if err != nil {
		t.Errorf("an error occurred: %v", err)
	}
}
