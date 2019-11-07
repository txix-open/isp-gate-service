package domain

type ProxyResponse struct {
	RequestBody  []byte
	ResponseBody []byte
	Error        error
}
