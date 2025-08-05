package transcodingv1

import (
	"fmt"
)

// ValidateRational validates that a Rational number is valid (non-zero denominator)
func ValidateRational(r *Rational) error {
	if r == nil {
		return fmt.Errorf("rational cannot be nil")
	}

	if r.Denominator == 0 {
		return fmt.Errorf("denominator must not be zero")
	}

	return nil
}

// ToFloat64 safely converts a Rational to float64, checking for zero denominator
func (r *Rational) ToFloat64() (float64, error) {
	if err := ValidateRational(r); err != nil {
		return 0, err
	}

	return float64(r.Numerator) / float64(r.Denominator), nil
}

// NewRational creates a new Rational with validation
func NewRational(numerator, denominator int32) (*Rational, error) {
	r := &Rational{
		Numerator:   numerator,
		Denominator: denominator,
	}

	if err := ValidateRational(r); err != nil {
		return nil, err
	}

	return r, nil
}

// Common frame rates as pre-validated Rational values
var (
	// NTSC frame rates
	FrameRate23_976 = &Rational{Numerator: 24000, Denominator: 1001} // 23.976 fps
	FrameRate29_97  = &Rational{Numerator: 30000, Denominator: 1001} // 29.97 fps
	FrameRate59_94  = &Rational{Numerator: 60000, Denominator: 1001} // 59.94 fps

	// Standard frame rates
	FrameRate24 = &Rational{Numerator: 24, Denominator: 1} // 24 fps
	FrameRate25 = &Rational{Numerator: 25, Denominator: 1} // 25 fps (PAL)
	FrameRate30 = &Rational{Numerator: 30, Denominator: 1} // 30 fps
	FrameRate50 = &Rational{Numerator: 50, Denominator: 1} // 50 fps
	FrameRate60 = &Rational{Numerator: 60, Denominator: 1} // 60 fps
)
