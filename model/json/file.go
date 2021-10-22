package json

type UploadCredential struct {
	Base
	Data struct {
		DataBase
		Credential string `json:"credential"`
		StoreHost  string `json:"store_host"`
		EventId    string `json:"event_id"`
	} `json:"data"`
}

type UploadFile struct {
	Base
	Data struct {
		DataBase
	} `json:"data"`
}

type DeleteFile struct {
	Base
	Data struct {
		DataBase
	} `json:"data"`
}

type ListFile struct {
	Base
	Data struct {
		DataBase
		Data string `json:"data"`
	} `json:"data"`
}
type HeadFile struct {
	Base
	Data struct {
		DataBase
		Header map[string]string `json:"header"`
	} `json:"data"`
}
type GetCid struct {
	Base
	Data struct {
		Cid       string `json:"cid"`
		StoreHost string `json:"store_host"`
	} `json:"data"`
}
