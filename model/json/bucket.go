package json

type ListBucket struct {
	Base
	Data struct {
		DataBase
		Data string `json:"data"` //payload
	} `json:"data"`
}
type CreateBucket struct {
	Base
	Data struct{ DataBase } `json:"data"`
}
type DeleteBucket struct {
	Base
	Data struct{ DataBase } `json:"data"`
}
