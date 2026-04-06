package entity

type AuthenticateRequest struct {
	Token string
}

type AuthenticateResponse struct {
	Authenticated bool
	ErrorReason   string
	AuthData      *AppAuthData
}

type AuthorizeRequest struct {
	ApplicationId int
	HttpMethod    string
	Endpoint      string
}

type AuthorizeResponse struct {
	Authorized bool
}

type AdminAuthorizeRequest struct {
	AdminId    int
	Permission string
}

type AdminAuthorizeResponse struct {
	Authorized bool
}
