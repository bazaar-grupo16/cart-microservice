package models

import (
	"encoding/json"
	"testing"
)

func TestPriceRoundingBehavior(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1.005", "1.01"},
		{"18.605", "18.61"},
		{"20.005", "20.01"},
		{"20.015", "20.02"},
		{"10.000", "10.00"},
		{"9.99", "9.99"},
		{"100000000000.555", "100000000000.56"},
	}

	for _, tt := range tests {
		p := Price(tt.input)
		b, err := json.Marshal(p)
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", tt.input, err)
		}

		if string(b) != tt.expected {
			t.Errorf("For %s: Expected %s, got %s", tt.input, tt.expected, string(b))
		}
	}
}

func TestPriceIsPositive(t *testing.T) {
	if !Price("10.00").IsPositive() {
		t.Error("Expected 10.00 to be positive")
	}
	if !Price("0.01").IsPositive() {
		t.Error("Expected 0.01 to be positive")
	}
	if Price("0").IsPositive() {
		t.Error("Expected 0 to not be positive")
	}
	if Price("-1").IsPositive() {
		t.Error("Expected -1 to not be positive")
	}
}

func TestPriceIsAtLeast(t *testing.T) {
	if !Price("0.01").IsAtLeast("0.01") {
		t.Error("Expected 0.01 to be at least 0.01")
	}
	if !Price("10.00").IsAtLeast("0.01") {
		t.Error("Expected 10.00 to be at least 0.01")
	}
	if Price("0.009").IsAtLeast("0.01") {
		t.Error("Expected 0.009 to NOT be at least 0.01")
	}
}

func TestPriceAdd(t *testing.T) {
	sum := Price("9.99").Add(Price("10.01"))
	if sum.Cmp(Price("20.00")) != 0 {
		t.Errorf("Expected 9.99 + 10.01 = 20.00, got %s", string(sum))
	}
}
