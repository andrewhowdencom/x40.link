// Package uid (or unique ID) provides mechanisms of generating the slugs for URLs.
package uid

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net/url"
)

// Type* are prefixes that are applied to the ID so that as new IDs are produced from different mechanisms,
// they do not collide.
//
// Exists to "future-proof" generations, in case it turns out one is prone to collisions.
const (
	TypeRandom byte = 1

	// Not intended for production use.
	TypeFails  byte = 90
	TypeStatic byte = 91
)

// Err* are sentinel errors
var (
	ErrFailed = errors.New("failed to generate id")
)

// funcMap calls the appropriate function based on the sentinel byte.
var funcMap = map[byte]func(u *url.URL) ([]byte, error){
	TypeRandom: Rand,

	// Testing
	TypeFails:  Failing,
	TypeStatic: Static([]byte{00, 00, 00, 00}),
}

// Generator is the type that receives a URL and returns an ID. Note: Not all generators derive their values
// from the URL.
type Generator struct {
	t byte
}

// New generates a generator which transforms the input URL to something
func New(t byte) *Generator {
	_, ok := funcMap[t]
	if !ok {
		panic("invalid generator provided: " + string(t))
	}

	return &Generator{t: t}
}

// ID converts the returned byte array to the base62 representation, complete with prefix.
func (g *Generator) ID(u *url.URL) (string, error) {
	id, err := funcMap[g.t](u)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrFailed, err)
	}

	var i big.Int
	m := append([]byte{g.t}, id...)
	i.SetBytes(m)

	return i.Text(62), nil
}

// Rand returns a random, 3 byte value. 3 bytes is ~8M (signed); when the number of URLs for this
// collides, I'll have bigger problems than just the collisions.
func Rand(_ *url.URL) ([]byte, error) {
	tok := make([]byte, 3)
	_, err := rand.Read(tok)

	if err != nil {
		return nil, err
	}

	return tok, nil
}

// Failing is a generator that just fails. Used for testing.
func Failing(_ *url.URL) ([]byte, error) {
	return nil, errors.New("i failed")
}

// Static is a generator that returns a static set of bytes. Used for testing.
func Static(s []byte) func(*url.URL) ([]byte, error) {
	return func(u *url.URL) ([]byte, error) {
		return s, nil
	}
}
