// pkg/wordpress/auth.go
package wordpress

import (
    "encoding/base64"
    "fmt"
    "net/http"
)

type Auth struct {
    username string
    password string
    baseURL  string
}

func NewAuth(baseURL, username, password string) (*Auth, error) {
    if username == "" || password == "" {
        return nil, fmt.Errorf("username and password are required")
    }
    
    return &Auth{
        username: username,
        password: password,
        baseURL:  baseURL,
    }, nil
}

func (a *Auth) Authenticate(req *http.Request) {
    // Use Application Passwords (Basic Auth)
    auth := base64.StdEncoding.EncodeToString([]byte(a.username + ":" + a.password))
    req.Header.Set("Authorization", "Basic "+auth)
}

func (a *Auth) GetApplicationPasswordURL() string {
    return a.baseURL + "/wp-admin/application-passwords/"
}

// For JWT authentication if plugin is installed
type JWTAuth struct {
    token string
}

func (j *JWTAuth) Authenticate(req *http.Request) {
    req.Header.Set("Authorization", "Bearer "+j.token)
}