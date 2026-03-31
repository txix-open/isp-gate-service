package domain

type AuthData struct {
	AppName       string
	SystemId      int
	DomainId      int
	ServiceId     int
	ApplicationId int

	CustomAuthData *ThirdPartyAuthData
}

type ThirdPartyAuthData struct {
	Identity       string
	IdentityHeader string
	ExtraHeaders   map[string][]string
}

type AuthenticateResponse struct {
	Authenticated bool
	ErrorReason   string
	AuthData      *AuthData
}
