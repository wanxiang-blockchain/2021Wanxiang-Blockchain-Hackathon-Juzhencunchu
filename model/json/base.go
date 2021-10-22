package json

type Base struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type DataBase struct {
	HttpStatus int `json:"HttpStatus"`
	Error      struct {
		Code      string `json:"Code"`
		Message   string `json:"Message"`
		Resource  string `json:"Resource"`
		RequestId string `json:"RequestId"`
	} `json:"Error"`
}
