package jwts_test

import (
	"testing"

	"github.com/andrewhowdencom/x40.link/api/auth/jwts"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestNeedsPermission(t *testing.T) {
	for _, tc := range []struct {
		name  string
		needs string

		tok jwts.X40
		err error
	}{
		{
			name:  "empty needs, pass",
			needs: "",

			tok: jwts.X40{},
			err: nil,
		},
		{
			name:  "need, does't have",
			needs: "TEST-PERMISSION",
			tok:   jwts.X40{},

			err: jwts.ErrMissingPermission,
		},
		{
			name:  "empty, has spares",
			needs: "",
			tok: jwts.X40{
				Permissions: []string{
					"TEST-PERMISSION",
				},
			},
			err: nil,
		},
		{
			name:  "needs, has",
			needs: "TEST-PERMISSION",
			tok: jwts.X40{
				Permissions: []string{
					"NOT-RELEVANT-PERMISSION",
					"TEST-PERMISSION",
				},
			},
			err: nil,
		},
	} {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.ErrorIs(t, jwts.NeedsPermission(tc.needs)(tc.tok), tc.err)
		})
	}
}

func TestValidate(t *testing.T) {
	for _, tc := range []struct {
		name string

		tok jwts.X40
		err error
	}{
		{
			name: "no subject",
			tok:  jwts.X40{},

			err: jwts.ErrMissingSubject,
		},
		{
			name: "has needs, doesn't meet them",
			tok: jwts.X40{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "TEST-SUBJECT",
				},
				Needs: jwts.NeedsPermission("TEST-PERMISSION"),
			},

			err: jwts.ErrMissingPermission,
		},
		{
			name: "has needs, meets them",
			tok: jwts.X40{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "TEST-SUBJECT",
				},
				Needs: jwts.NeedsPermission("TEST-PERMISSION"),
				Permissions: []string{
					"TEST-PERMISSION",
				},
			},

			err: nil,
		},
		{
			name: "has no needs",
			tok: jwts.X40{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "TEST-SUBJECT",
				},
			},

			err: nil,
		},
	} {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.ErrorIs(t, tc.tok.Validate(), tc.err)
		})
	}
}
