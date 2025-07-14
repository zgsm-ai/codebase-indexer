package parser

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestScmLoad(t *testing.T) {
	assert.NotPanics(t,
		func() {
			err := loadScm()
			assert.NoError(t, err)
		})
}
