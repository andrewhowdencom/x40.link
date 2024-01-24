package cfg

import (
	"fmt"
	"sync"
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestAddFlagTo(t *testing.T) {
	for _, tc := range []struct {
		name string
		v    V

		panic interface{}
	}{
		{
			name: "string",
			v: V{
				Path:    "example.path",
				Short:   "",
				Default: "foo",
				Usage:   "configures the example path",
				mu:      &sync.Mutex{},
			},
		},
		{
			name: "string+short",
			v: V{
				Path:    "example.path",
				Short:   "e",
				Default: "bar",
				Usage:   "configures the example path",
				mu:      &sync.Mutex{},
			},
		},
		{
			name: "unsupported value",
			v: V{
				Path:    "example.path",
				Default: struct{}{},
				Usage:   "this should fail",
				mu:      &sync.Mutex{},
			},
			panic: "unsupported conversion to flag: example.path",
		},
		{
			name: "bool",
			v: V{
				Path:    "example.path",
				Default: false,
				Usage:   "enables the example path",
				mu:      &sync.Mutex{},
			},
		},
	} {
		tc := tc

		// This work is not concurrency safe.
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				rPanic := recover()
				assert.Equal(t, tc.panic, rPanic)
			}()

			fs := pflag.NewFlagSet("TEST-FLAG-SET", pflag.ExitOnError)

			tc.v.AddFlagTo(fs)

			flag := fs.Lookup(tc.v.Path)

			assert.NotNil(t, flag)

			assert.Equal(t, tc.v.Path, flag.Name)
			assert.Equal(t, tc.v.Short, flag.Shorthand)

			switch tc.v.Default.(type) {
			case string:
				assert.Equal(t, tc.v.Default, flag.DefValue)
			case bool:
				assert.Equal(t, fmt.Sprintf("%t", tc.v.Default), flag.DefValue)
			}

			assert.Equal(t, tc.v.Usage, flag.Usage)

		})
	}
}

func TestFlagToConfiguration(t *testing.T) {
	t.Parallel()

	fs := pflag.NewFlagSet("TEST-FLAG-SET", pflag.ExitOnError)
	v := &V{Path: "example.path", Default: "foo", mu: &sync.Mutex{}}

	v.AddFlagTo(fs)

	// Bind the flags
	assert.Nil(t, viper.GetViper().BindPFlags(fs))

	// Look it up in Viper
	assert.Equal(t, "foo", viper.GetString(v.Path))

	// Change the flag value
	if err := fs.Lookup(v.Path).Value.Set("bar"); err != nil {
		panic(err)
	}

	// Look it up in Viper
	assert.Equal(t, "bar", viper.GetString(v.Path))

}
