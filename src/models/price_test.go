package models

import (
	"encoding/json"
	"testing"
	"time"
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
		p := NewPrice(tt.input)
		b, err := json.Marshal(p)
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", tt.input, err)
		}

		var result struct {
			Value string `json:"value"`
		}
		if err := json.Unmarshal(b, &result); err != nil {
			t.Fatalf("failed to unmarshal result for %s: %v", tt.input, err)
		}

		if result.Value != tt.expected {
			t.Errorf("For %s: Expected value %s, got %s", tt.input, tt.expected, result.Value)
		}
	}
}

func TestPriceIsPositive(t *testing.T) {
	if !NewPrice("10.00").IsPositive() {
		t.Error("Expected 10.00 to be positive")
	}
	if !NewPrice("0.01").IsPositive() {
		t.Error("Expected 0.01 to be positive")
	}
	if NewPrice("0").IsPositive() {
		t.Error("Expected 0 to not be positive")
	}
	if NewPrice("-1").IsPositive() {
		t.Error("Expected -1 to not be positive")
	}
}

func TestPriceIsAtLeast(t *testing.T) {
	if !NewPrice("0.01").IsAtLeast("0.01") {
		t.Error("Expected 0.01 to be at least 0.01")
	}
	if !NewPrice("10.00").IsAtLeast("0.01") {
		t.Error("Expected 10.00 to be at least 0.01")
	}
	if NewPrice("0.009").IsAtLeast("0.01") {
		t.Error("Expected 0.009 to NOT be at least 0.01")
	}
}

func TestPriceAdd(t *testing.T) {
	sum := NewPrice("9.99").Add(NewPrice("10.01"))
	if sum.Cmp(NewPrice("20.00")) != 0 {
		t.Errorf("Expected 9.99 + 10.01 = 20.00, got %v", sum.Value)
	}
}

func TestPriceMultiply(t *testing.T) {
	p := NewPrice("9.99")
	prod := p.Multiply(3)
	if prod.Cmp(NewPrice("29.97")) != 0 {
		t.Errorf("Expected 9.99 * 3 = 29.97, got %v", prod.Value)
	}

	prodZero := p.Multiply(0)
	if prodZero.Cmp(NewPrice("0.00")) != 0 {
		t.Errorf("Expected 9.99 * 0 = 0.00, got %v", prodZero.Value)
	}
}

func TestIsPriceStale(t *testing.T) {
	p := NewPrice("10.00")
	if p.IsPriceStale() {
		t.Error("Expected 10.00 to not be stale")
	}

	pStale := NewPrice("10.00")
	pStale.UpdatedAt = time.Now().Add(-25 * time.Hour)
	if !pStale.IsPriceStale() {
		t.Error("Expected 10.00 to be stale")
	}
}
