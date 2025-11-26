package request

import (
	"context"
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

var (
	ErrNotAuthenticated = errors.New("not authenticated")
)

type AuthData struct {
	AppName       string
	SystemId      int
	DomainId      int
	ServiceId     int
	ApplicationId int
}

type Context struct {
	request        *http.Request
	responseWriter http.ResponseWriter

	endpoint string

	authenticated bool
	authData      *AuthData

	adminAuthenticated bool
	adminId            int
	adminToken         string

	queryParams map[string]string
}

func NewContext(request *http.Request, response http.ResponseWriter, endpoint string) *Context {
	return &Context{
		request:        request,
		responseWriter: response,
		endpoint:       endpoint,
	}
}

func (c *Context) Request() *http.Request {
	return c.request
}

func (c *Context) ResponseWriter() http.ResponseWriter {
	return c.responseWriter
}

func (c *Context) SetResponseWriter(writer http.ResponseWriter) {
	c.responseWriter = writer
}

func (c *Context) Endpoint() string {
	return c.endpoint
}

func (c *Context) Authenticate(authData AuthData) {
	c.authenticated = true
	c.authData = &authData
}

func (c *Context) GetAuthData() (AuthData, error) {
	if !c.authenticated {
		return AuthData{}, ErrNotAuthenticated
	}
	return *c.authData, nil
}

func (c *Context) IsAdminAuthenticated() bool {
	return c.adminAuthenticated
}

func (c *Context) AdminId() int {
	return c.adminId
}

func (c *Context) AdminToken() string {
	return c.adminToken
}

func (c *Context) AuthenticateAdmin(adminId int, adminToken string) {
	c.adminAuthenticated = true
	c.adminId = adminId
	c.adminToken = adminToken
}

func (c *Context) Context() context.Context {
	return c.request.Context()
}

func (c *Context) SetContext(ctx context.Context) {
	c.request = c.request.WithContext(ctx)
}

func (c *Context) Param(name string) string {
	value := c.request.Header.Get(name)
	if value != "" {
		return strings.TrimSpace(value)
	}

	if c.queryParams == nil {
		query := c.request.URL.Query()
		c.queryParams = map[string]string{}
		for key, values := range query {
			if len(values) == 0 {
				continue
			}
			c.queryParams[strings.ToLower(key)] = values[0]
		}
	}
	value = c.queryParams[strings.ToLower(name)]

	return strings.TrimSpace(value)
}
