// Package jwts provides various different JWT tokens.
package jwts

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// AudienceX40API is the audience field required for the X40 API.
	AudienceX40API = "https://api.x40.link"

	// ClaimPermissions is how auth0 returns the roles that are requested (via scopes).
	//
	// See:
	// 1. https://auth0.com/docs/get-started/apis/enable-role-based-access-control-for-apis
	ClaimPermissions = "permissions"
)

// Err* are sentinel errors
var (
	ErrMissingPermission = errors.New("missing permission")
	ErrMissingSubject    = errors.New("missing subject")
)

// NeedsPermission allows ensuring the validator guarantees a permission exists.
func NeedsPermission(needs string) func(X40) error {
	return func(x X40) error {
		// Shortcut, in the user (for some reason) tried to supply a zero length permission.
		if needs == "" {
			return nil
		}

		for _, has := range x.Permissions {
			if has == needs {
				return nil
			}
		}

		return fmt.Errorf("%w: %s", ErrMissingPermission, needs)
	}
}

// X40 is a token extended with claims specific to this application
type X40 struct {
	// val is the extension function that allows custom validating these claims
	Needs func(X40) error

	// The standard claims (based on the golang-jwt/jwt package)
	jwt.RegisteredClaims

	// See jwts.ClaimPermissions
	Permissions []string `json:"permissions"`
}

// Validate allows extending the claims validation.
func (x X40) Validate() error {
	// By default, we always also want the subject claim.
	if x.Subject == "" {
		return ErrMissingSubject
	}

	if x.Needs != nil {
		return x.Needs(x)
	}

	return nil
}
