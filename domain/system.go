package domain

type AuthenticateRequest struct {
	Token string
}

type AuthenticateResponse struct {
	Authenticated bool
	ErrorReason   string
	AuthData      *AuthData
}

type AuthorizeRequest struct {
	ApplicationId int
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
