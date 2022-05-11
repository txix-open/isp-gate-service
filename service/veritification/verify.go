package veritification

type Verify interface {
	ApplicationToken(string) (map[string]string, error)
	Identity(map[string]string, string) (map[string]string, error)
}

func NewRuntimeVerify() Verify {
	return &runtimeVerify{}
}
