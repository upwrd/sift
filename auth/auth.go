// Package auth manages the authorization of SIFT users
package auth

import (
	"code.google.com/p/go-uuid/uuid"
	log "gopkg.in/inconshreveable/log15.v2"
	logext "gopkg.in/inconshreveable/log15.v2/ext"
	"sift/logging"
)

// Log is used to log messages for the auth package. Logs are disabled by
// default; use sift/logging.SetLevel() to set log levels for all packages, or
// Log.SetHandler() to set a custom handler for this package (see:
// https://godoc.org/gopkg.in/inconshreveable/log15.v2)
var Log = logging.Log.New("pkg", "auth")

// A Token uniquely identifies a user. To get a Token, call Authorizor.Login().
type Token string

// An Authorizor allows users to log in to a system, and authorizes them for
// specific actions
type Authorizor interface {
	Login() Token
	Authorize(Token, interface{}) bool
}

// A SiftAuthorizor authorizes users to log in to a system, and authorizes them for
// specific actions. SiftAuthorizors could be instantiated by a call to auth.New()
type SiftAuthorizor struct {
	log log.Logger
}

// New creates a new SiftAuthorizor
func New() *SiftAuthorizor {
	return &SiftAuthorizor{
		log: Log.New("obj", "authorizor", "id", logext.RandId(8)),
	}
}

// Login registers a new user and returns a unique Token that they may use to
// authorize further actions.
func (a *SiftAuthorizor) Login() Token {
	return Token(uuid.New())
}

// Authorize confirms whether or not a particular user (represented by their
// token) has access to perform a particular action.
func (a *SiftAuthorizor) Authorize(t Token, path interface{}) bool {
	// TODO: properly implement Authorize
	a.log.Warn("STUB: Authorize always returns true (authorized)")
	return true
}
