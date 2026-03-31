package domain

type AuthenticateAppResponse struct {
	Authenticated bool
	ErrorReason   string
	AuthData      *AppAuthData
}

type AppAuthData struct {
	AppName       string
	SystemId      int
	DomainId      int
	ServiceId     int
	ApplicationId int
}

type ApplicationToken struct {
	AppToken string
	AppName  string
}

type UserToken struct {
	Token string
}

type AuthenticateUserResponse struct {
	Authenticated bool
	ShouldSkip    bool
	ErrorReason   string
	AuthData      *UserAuthData
}

type UserAuthData struct {
	SkipAppAuth    bool
	Identity       string
	IdentityHeader string
	ExtraHeaders   map[string][]string
}
