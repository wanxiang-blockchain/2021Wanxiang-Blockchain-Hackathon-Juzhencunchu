package yml

type Config struct {
	Http struct {
		Addr      string `yaml:"addr"`
		InfoLevel string `yaml:"info_level"`
	} `yaml:"http"`
	OpenApi struct {
		Host   string `yaml:"host"`
		AppId  string `yaml:"app_id"`
		Secret string `yaml:"secret"`
		Paths  struct {
			UploadCredential string `yaml:"upload_credential"`
			SecretByAppid    string `yaml:"secret_by_appid"`
			UploadFile       string `yaml:"upload_file"`
			DownloadFile     string `yaml:"download_file"`
			DeleteFile       string `yaml:"delete_file"`
			ListFile         string `yaml:"list_file"`
			HeadFile         string `yaml:"head_file"`
			CreateBucket     string `yaml:"create_bucket"`
			DeleteBucket     string `yaml:"delete_bucket"`
			ListBucket       string `yaml:"list_bucket"`
			HeadBucket       string `yaml:"head_bucket"`
			GetCid           string `yaml:"get_cid"`
		} `yaml:"paths"`
	} `yaml:"open_api"`
}
