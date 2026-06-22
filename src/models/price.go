package models

import (
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Price is a struct that ensures decimal precision for currency values.
// In MongoDB and JSON it contains the string 'value' and 'updatedAt' timestamp.
type Price struct {
	Value     json.Number `bson:"value"     json:"value"`
	UpdatedAt time.Time   `bson:"updatedAt" json:"updatedAt"`
}

// NewPrice creates a new Price with the given value string and current UTC timestamp.
func NewPrice(val string) Price {
	if val == "" {
		val = "0.00"
	}
	return Price{
		Value:     json.Number(val),
		UpdatedAt: time.Now().UTC(),
	}
}

// Rat returns the exact rational representation of the price value.
func (p Price) Rat() *big.Rat {
	r, ok := new(big.Rat).SetString(string(p.Value))
	if !ok {
		return new(big.Rat)
	}
	return r
}

// Float64 returns the float64 representation of the price value.
func (p Price) Float64() float64 {
	f, _ := p.Value.Float64()
	return f
}

// MarshalJSON customizes the JSON representation to ensure Value has strictly 2 decimal places.
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

	bigHundred := big.NewInt(100)
	dollars := new(big.Int)
	remainder := new(big.Int)
	dollars.DivMod(cents, bigHundred, remainder)
	remainder.Abs(remainder)

	formattedValue := fmt.Sprintf("%s.%02d", dollars.String(), remainder)

	// We return the full struct JSON
	type alias Price
	return json.Marshal(&struct {
		Value string `json:"value"`
		alias
	}{
		Value: formattedValue,
		alias: alias(p),
	})
}

// UnmarshalJSON customizes the JSON unmarshaling of Price.
func (p *Price) UnmarshalJSON(data []byte) error {
	// Support both: a naked number/string (old format) or the new struct format
	var simple json.Number
	if err := json.Unmarshal(data, &simple); err == nil {
		*p = NewPrice(string(simple))
		return nil
	}

	type alias Price
	var aux struct {
		Value json.Number `json:"value"`
		alias
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	p.Value = aux.Value
	p.UpdatedAt = aux.UpdatedAt
	return nil
}

// MarshalBSONValue implements bson.ValueMarshaler.
func (p Price) MarshalBSONValue() (bson.Type, []byte, error) {
	// Canonical "X.YY" formatting for the string value
	r := p.Rat()
	hundred := new(big.Rat).SetInt64(100)
	r.Mul(r, hundred)
	half := new(big.Rat).SetFrac64(1, 2)
	r.Add(r, half)
	cents := new(big.Int).Div(r.Num(), r.Denom())
	bigHundred := big.NewInt(100)
	dollars := new(big.Int)
	remainder := new(big.Int)
	dollars.DivMod(cents, bigHundred, remainder)
	remainder.Abs(remainder)
	formattedValue := fmt.Sprintf("%s.%02d", dollars.String(), remainder)

	b, err := bson.Marshal(struct {
		Value     string    `bson:"value"`
		UpdatedAt time.Time `bson:"updatedAt"`
	}{
		Value:     formattedValue,
		UpdatedAt: p.UpdatedAt.UTC(),
	})
	if err != nil {
		return bson.TypeEmbeddedDocument, nil, err
	}
	return bson.TypeEmbeddedDocument, b, nil
}

// UnmarshalBSONValue implements bson.ValueUnmarshaler.
func (p *Price) UnmarshalBSONValue(t bson.Type, data []byte) error {
	// Support reading the old string format ("9.99")
	if t == bson.TypeString {
		var s string
		if err := bson.UnmarshalValue(t, data, &s); err != nil {
			return err
		}
		*p = NewPrice(s)
		return nil
	}

	// Support the new struct format
	var aux struct {
		Value     string    `bson:"value"`
		UpdatedAt time.Time `bson:"updatedAt"`
	}
	if err := bson.Unmarshal(data, &aux); err != nil {
		return err
	}
	p.Value = json.Number(aux.Value)
	p.UpdatedAt = aux.UpdatedAt
	return nil
}

// Cmp compares two Price values using exact arithmetic.
func (p Price) Cmp(other Price) int {
	return p.Rat().Cmp(other.Rat())
}

// IsPositive returns true if the price value is strictly greater than zero.
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

// Add returns a new Price that is the exact sum of p and other, with current timestamp.
func (p Price) Add(other Price) Price {
	sum := new(big.Rat).Add(p.Rat(), other.Rat())
	return NewPrice(sum.FloatString(2))
}

// Multiply returns a new Price that is the exact product of p and a factor, with current timestamp.
func (p Price) Multiply(factor int) Price {
	f := new(big.Rat).SetInt64(int64(factor))
	prod := new(big.Rat).Mul(p.Rat(), f)
	return NewPrice(prod.FloatString(2))
}

func (p Price) IsPriceStale() bool {
	return time.Since(p.UpdatedAt) > 24*time.Hour
}
