package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"math/big"
)

// Price is a custom type that uses json.Number to ensure precision during JSON unmarshaling.
// All arithmetic and comparisons use math/big.Rat for exact decimal operations.
type Price json.Number

// Rat returns the exact rational representation of the price string.
func (p Price) Rat() *big.Rat {
	r, ok := new(big.Rat).SetString(string(p))
	if !ok {
		return new(big.Rat)
	}
	return r
}

// Float64 returns the float64 representation of the price.
// Prefer Rat(), Cmp(), IsPositive(), or IsAtLeast() for precise operations.
func (p Price) Float64() float64 {
	f, _ := json.Number(p).Float64()
	return f
}

// MarshalJSON customizes the JSON representation of Price to ensure strictly 2 decimal places.
// Rounding is performed using exact rational arithmetic to avoid float64 precision issues.
func (p Price) MarshalJSON() ([]byte, error) {
	r := p.Rat()

	// Multiply by 100
	hundred := new(big.Rat).SetInt64(100)
	r.Mul(r, hundred)

	// Round half-up: add 1/2, then truncate toward zero
	half := new(big.Rat).SetFrac64(1, 2)
	r.Add(r, half)

	// Truncate to integer (floor for positive prices)
	cents := new(big.Int).Div(r.Num(), r.Denom())

	// Format as "dollars.cents" using big.Int arithmetic to avoid int64 overflow
	bigHundred := big.NewInt(100)
	dollars := new(big.Int)
	remainder := new(big.Int)
	dollars.DivMod(cents, bigHundred, remainder)
	remainder.Abs(remainder)
	formatted := fmt.Sprintf("%s.%02d", dollars.String(), remainder)
	return []byte(formatted), nil
}

// UnmarshalJSON customizes the JSON unmarshaling of Price to use json.Number.
func (p *Price) UnmarshalJSON(data []byte) error {
	var n json.Number
	if err := json.Unmarshal(data, &n); err != nil {
		return err
	}
	*p = Price(n)
	return nil
}

// Cmp compares two Price values using exact arithmetic.
// Returns -1 if p < other, 0 if p == other, +1 if p > other.
func (p Price) Cmp(other Price) int {
	return p.Rat().Cmp(other.Rat())
}

// IsPositive returns true if the price is strictly greater than zero, using exact arithmetic.
func (p Price) IsPositive() bool {
	return p.Rat().Sign() > 0
}

// IsAtLeast returns true if the price is greater than or equal to the given minimum string value.
func (p Price) IsAtLeast(min string) bool {
	m, ok := new(big.Rat).SetString(min)
	if !ok {
		return false
	}
	return p.Rat().Cmp(m) >= 0
}

// Add returns a new Price that is the exact sum of p and other.
func (p Price) Add(other Price) Price {
	sum := new(big.Rat).Add(p.Rat(), other.Rat())
	return Price(sum.FloatString(2))
}

// Scan implements the sql.Scanner interface for database compatibility.
func (p *Price) Scan(value interface{}) error {
	if value == nil {
		*p = Price("0.00")
		return nil
	}
	switch v := value.(type) {
	case []byte:
		*p = Price(string(v))
	case string:
		*p = Price(v)
	case float64:
		// Use Rat to avoid precision loss from Sprintf
		r := new(big.Rat).SetFloat64(v)
		*p = Price(r.FloatString(2))
	default:
		return fmt.Errorf("unsupported type for Price: %T", value)
	}
	return nil
}

// Value implements the driver.Valuer interface for database compatibility.
func (p Price) Value() (driver.Value, error) {
	return string(p), nil
}
