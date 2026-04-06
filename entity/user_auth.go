package entity

type UserAuthenticateRequest struct {
	Token string
}

type UserAuthenticateResponse struct {
	Authenticated bool
	ErrorReason   string
	AuthData      *UserAuthData
}

type UserAuthData struct {
	Identity       string
	IdentityHeader string
	ExtraHeaders   map[string][]string
}
