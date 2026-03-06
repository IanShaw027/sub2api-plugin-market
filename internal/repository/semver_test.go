package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnsureVPrefix(t *testing.T) {
	assert.Equal(t, "v1.0.0", ensureVPrefix("1.0.0"))
	assert.Equal(t, "v1.0.0", ensureVPrefix("v1.0.0"))
	assert.Equal(t, "", ensureVPrefix(""))
}

func TestSemverLTE(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"1.0.0", "1.0.0", true},
		{"1.0.0", "1.1.0", true},
		{"1.0.0", "2.0.0", true},
		{"1.9.0", "1.10.0", true},  // key case: string comparison gives wrong result
		{"2.0.0", "1.0.0", false},
		{"1.10.0", "1.9.0", false}, // key case
		{"1.0.0-alpha", "1.0.0", true},
		{"v1.0.0", "v2.0.0", true}, // with v prefix
	}

	for _, tt := range tests {
		t.Run(tt.a+"<="+tt.b, func(t *testing.T) {
			assert.Equal(t, tt.want, semverLTE(tt.a, tt.b))
		})
	}
}

func TestSemverGTE(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"1.0.0", "1.0.0", true},
		{"2.0.0", "1.0.0", true},
		{"1.10.0", "1.9.0", true}, // key case
		{"1.0.0", "2.0.0", false},
		{"1.9.0", "1.10.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.a+">="+tt.b, func(t *testing.T) {
			assert.Equal(t, tt.want, semverGTE(tt.a, tt.b))
		})
	}
}

func TestIsVersionCompatible(t *testing.T) {
	tests := []struct {
		name       string
		min, max   string
		host       string
		compatible bool
	}{
		{"exact match", "1.0.0", "", "1.0.0", true},
		{"host above min, no max", "1.0.0", "", "2.0.0", true},
		{"host below min", "2.0.0", "", "1.0.0", false},
		{"host within range", "1.0.0", "2.0.0", "1.5.0", true},
		{"host above max", "1.0.0", "2.0.0", "3.0.0", false},
		{"host at max", "1.0.0", "2.0.0", "2.0.0", true},
		{"host at min", "1.0.0", "2.0.0", "1.0.0", true},
		{"1.9 vs 1.10 boundary", "1.0.0", "1.10.0", "1.9.0", true},
		{"1.10 vs 1.9 max boundary", "1.0.0", "1.9.0", "1.10.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.compatible, isVersionCompatible(tt.min, tt.max, tt.host))
		})
	}
}
