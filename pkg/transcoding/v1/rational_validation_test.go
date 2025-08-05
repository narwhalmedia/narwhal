package transcodingv1

import (
	"testing"
)

func TestValidateRational(t *testing.T) {
	tests := []struct {
		name    string
		r       *Rational
		wantErr bool
	}{
		{
			name:    "valid rational",
			r:       &Rational{Numerator: 24000, Denominator: 1001},
			wantErr: false,
		},
		{
			name:    "zero numerator is valid",
			r:       &Rational{Numerator: 0, Denominator: 1},
			wantErr: false,
		},
		{
			name:    "zero denominator is invalid",
			r:       &Rational{Numerator: 24, Denominator: 0},
			wantErr: true,
		},
		{
			name:    "negative values are valid",
			r:       &Rational{Numerator: -24, Denominator: -1},
			wantErr: false,
		},
		{
			name:    "nil rational is invalid",
			r:       nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRational(tt.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRational() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRational_ToFloat64(t *testing.T) {
	tests := []struct {
		name    string
		r       *Rational
		want    float64
		wantErr bool
	}{
		{
			name:    "23.976 fps",
			r:       &Rational{Numerator: 24000, Denominator: 1001},
			want:    23.976023976023978, // 24000/1001
			wantErr: false,
		},
		{
			name:    "integer frame rate",
			r:       &Rational{Numerator: 30, Denominator: 1},
			want:    30.0,
			wantErr: false,
		},
		{
			name:    "zero denominator returns error",
			r:       &Rational{Numerator: 30, Denominator: 0},
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.ToFloat64()
			if (err != nil) != tt.wantErr {
				t.Errorf("Rational.ToFloat64() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Rational.ToFloat64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewRational(t *testing.T) {
	tests := []struct {
		name        string
		numerator   int32
		denominator int32
		wantErr     bool
	}{
		{
			name:        "valid rational",
			numerator:   24000,
			denominator: 1001,
			wantErr:     false,
		},
		{
			name:        "zero denominator rejected",
			numerator:   30,
			denominator: 0,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewRational(tt.numerator, tt.denominator)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRational() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Errorf("NewRational() returned nil without error")
			}
		})
	}
}

func TestCommonFrameRates(t *testing.T) {
	// Test that all predefined frame rates are valid
	frameRates := map[string]*Rational{
		"23.976": FrameRate23_976,
		"29.97":  FrameRate29_97,
		"59.94":  FrameRate59_94,
		"24":     FrameRate24,
		"25":     FrameRate25,
		"30":     FrameRate30,
		"50":     FrameRate50,
		"60":     FrameRate60,
	}

	for name, fr := range frameRates {
		t.Run(name+" fps", func(t *testing.T) {
			if err := ValidateRational(fr); err != nil {
				t.Errorf("Common frame rate %s is invalid: %v", name, err)
			}

			_, err := fr.ToFloat64()
			if err != nil {
				t.Errorf("Failed to convert %s fps to float64: %v", name, err)
			}
		})
	}
}
