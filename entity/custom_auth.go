package entity

type CustomAuthenticateRequest struct {
	Token string
}

type CustomAuthenticateResponse struct {
	Authenticated  bool
	ErrorReason    string
	Identity       string
	IdentityHeader string
	ExtraHeaders   map[string][]string
}
