package domain

type AdminAuthenticateResponse struct {
	Authenticated bool
	ErrorReason   string
	AdminId       int
}
