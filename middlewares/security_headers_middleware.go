package middlewares

import "github.com/gin-gonic/gin"

// SecurityHeadersMiddleware sets a baseline of response security headers that
// a StackHawk DAST scan found missing on every response: no
// Strict-Transport-Security, no X-Content-Type-Options, no Content-Security-Policy.
// default-src 'self' is deliberately not tighter than that, since
// /main/swagger/*any (gin-swagger) serves its own same-origin JS/CSS/HTML and
// would break under a stricter policy like default-src 'none'.
//
// frame-ancestors and form-action are set explicitly because, per the CSP
// spec, neither falls back to default-src when unset - a later StackHawk
// scan correctly flagged their absence as "CSP: Wildcard Directive" even
// with default-src 'self' present. This is a pure JSON API with no reason
// to be framed or to host an HTML form, so both are set to their tightest
// values.
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		c.Writer.Header().Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'; form-action 'self'")
		c.Next()
	}
}
