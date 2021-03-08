package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_RootArgs(t *testing.T) {
	assert.NotNil(t, rootArgs(rootCmd, nil))
	assert.Nil(t, rootArgs(rootCmd, []string{"url"}))
}

func Test_RootRun(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	rootRun(rootCmd, []string{"ftp://url"})

	assert.Equal(t, "unsupported protocol \"ftp\". http and https are supported\n", buf.String())
}
