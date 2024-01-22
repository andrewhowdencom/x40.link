// Package jwts provides various different JWT tokens.
package jwts

import "github.com/golang-jwt/jwt/v5"

// Default is the claims on a Standard JWT
//
// See
// 1. https://www.cerberauth.com/understanding-oauth2-access-token-claims
// 2. https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.4
// 3. https://www.iana.org/assignments/jwt/jwt.xhtml
type Default struct {
	// The issuer of the access token (i.e. the authorization server)
	Issuer string `json:"iss,omitempty"`

	// An identifier for the end-user at the issuer. For example, "f6b2cd98-b608-11ee-ae61-03b5060ca448"
	Subject string `json:"sub,omitempty"`

	// The audience of the token. In principle, whomever is designed to read it. In practice, the Client ID of the
	// oAuth2 client.
	Audience string `json:"aud,omitempty"`

	// Expiration is the time at which the token should no longer be considered valid. Expressed as "Numeric Date",
	// but in practice, is a unix timestamp.
	Expiration *jwt.NumericDate `json:"exp,omitempty"`

	// IssuedAt, or the point in time at which the token was issued.
	IssuedAt *jwt.NumericDate `json:"iat,omitempty"`

	// Not Before, or the point in time which the token _must not_ be used for processing.
	NotBefore *jwt.NumericDate `json:"nbf,omitempty"`
}

// GetExpirationTime allows the token to be used in the token generation package
func (t *Default) GetExpirationTime() (*jwt.NumericDate, error) {
	return t.Expiration, nil
}

// GetIssuedAt allows the token to be used in the token generation package
func (t *Default) GetIssuedAt() (*jwt.NumericDate, error) {
	return t.IssuedAt, nil
}

// GetNotBefore allows the token to be used in the token generation package
func (t *Default) GetNotBefore() (*jwt.NumericDate, error) {
	return t.NotBefore, nil
}

// GetIssuer allows the token to be used in the token generation package
func (t *Default) GetIssuer() (string, error) {
	return t.Issuer, nil
}

// GetAudience allows the token to be used in the token generation package
func (t *Default) GetAudience() (jwt.ClaimStrings, error) {
	return jwt.ClaimStrings{t.Audience}, nil
}

// GetSubject allows the token to be used in the token generation package
func (t *Default) GetSubject() (string, error) {
	return t.Subject, nil
}

// OIDC is a token extended with (some of the) expected OIDC Claims
// See:
// 1. https://openid.net/specs/openid-connect-core-1_0.html#StandardClaims
// 2. https://auth0.com/docs/secure/tokens/id-tokens/id-token-structure
type OIDC struct {
	*Default

	// End-User's full name in displayable form including all name parts, possibly including titles and suffixes,
	// ordered according to the End-User's locale and preferences.
	Name string `json:"name,omitempty"`

	// End-User's preferred e-mail address. Its value MUST conform to the RFC 5322 [RFC5322] addr-spec syntax. The RP
	// MUST NOT rely upon this value being unique, as discussed in Section 5.7.
	Email string `json:"email,omitempty"`

	// End-User's locale, represented as a BCP47 [RFC5646] language tag. This is typically an ISO 639 Alpha-2 [ISO639]
	// language code in lowercase and an ISO 3166-1 Alpha-2 [ISO3166‑1] country code in uppercase, separated by a dash.
	// For example, en-US or fr-CA. As a compatibility note, some implementations have used an underscore as the
	// separator rather than a dash, for example, en_US; Relying Parties MAY choose to accept this locale syntax
	// as well.
	Locale string `json:"locale,omitempty"`
}

// X40 is a token extended with claims specific to this application
type X40 struct {
	*Default
	*OIDC

	// Roles are the roles the application users may have. Long term, this should be deprecated and removed in
	// favor of ReBAC but for now, it
	Roles []string `json:"x40.link/roles"`
}
