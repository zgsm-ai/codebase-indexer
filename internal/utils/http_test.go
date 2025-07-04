package utils

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsAbortRetryError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "NilError",
			err:      nil,
			expected: false,
		},
		{
			name:     "UnauthorizedError",
			err:      errors.New("401 Unauthorized"),
			expected: true,
		},
		{
			name:     "TooManyRequestsError",
			err:      errors.New("429 Too Many Requests"),
			expected: true,
		},
		{
			name:     "ServiceUnavailableError",
			err:      errors.New("503 Service Unavailable"),
			expected: true,
		},
		{
			name:     "OtherError",
			err:      errors.New("some other error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAbortRetryError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
