package bldr

import (
	"fmt"
	"net/http"

	"github.com/distribution/distribution/v3/registry/auth"
)

type OCIBasicAuthAccess struct {
	user     string
	password string
}

var _ auth.AccessController = &OCIBasicAuthAccess{}

func NewOCIBasicAuthAccess(options map[string]interface{}) (auth.AccessController, error) {
	user, ok := options["username"]
	if !ok {
		return nil, fmt.Errorf("user not defined for basic auth access")
	}

	password, ok := options["password"]
	if !ok {
		return nil, fmt.Errorf("password not defined for basic auth access")
	}

	return &OCIBasicAuthAccess{user: user.(string), password: password.(string)}, nil
}

func (ac *OCIBasicAuthAccess) Authorized(req *http.Request, _ ...auth.Access) (*auth.Grant, error) {
	// Fetching does not require authentication
	if req.Method == http.MethodGet || req.Method == http.MethodHead {
		return &auth.Grant{User: auth.UserInfo{}, Resources: nil}, nil
	}

	username, password, ok := req.BasicAuth()
	if !ok {
		return nil, &basicAuthChallenge{}
	}

	if username == ac.user && password == ac.password {
		return &auth.Grant{User: auth.UserInfo{}, Resources: nil}, nil
	}

	return nil, auth.ErrAuthenticationFailure
}

type basicAuthChallenge struct {
}

var _ auth.Challenge = basicAuthChallenge{}

func (ch basicAuthChallenge) SetHeaders(_ *http.Request, w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Basic")
}

func (ch basicAuthChallenge) Error() string {
	return fmt.Sprintf("basic auth challenge: %s", auth.ErrInvalidCredential.Error())
}
