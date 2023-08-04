package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcess(t *testing.T) {
	tests := []struct {
		desc     string
		givePath string
		wantPath string
	}{
		{
			desc:     "erase inferred cover statement",
			givePath: "testdata/minimal.go",
			wantPath: "testdata/want_minimal.go",
		},
		{
			desc:     "code without inferred covered statements",
			givePath: "testdata/nomodify.go",
			wantPath: "testdata/want_nomodify.go",
		},
		{
			desc:     "code without inferred covered statements",
			givePath: "testdata/switch.go",
			wantPath: "testdata/want_switch.go",
		},
	}
	for _, tt := range tests {
		want, err := os.ReadFile(tt.wantPath)
		assert.NoError(t, err)
		actual, err := process(tt.givePath)
		require.NoError(t, err)

		assert.Equal(t, string(want), string(actual))
	}
}
