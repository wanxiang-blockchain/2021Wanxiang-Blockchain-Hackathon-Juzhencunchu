package main

import (
	"flag"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"
	"log"
	"net/http"
	"os"
	"s3Gateway/cmd"
	"s3Gateway/internal/logger"
	"s3Gateway/model/yml"
)

func init() {
	cmd.GlobalConfig = &yml.Config{}
}

func main() {
	configPath := flag.String("c", "./config.yaml", "load config(yaml)")
	flag.Parse()
	//读取配置
	fp, err := os.Open(*configPath)
	if err != nil {
		logger.Error("config load error:%s", err.Error())
	}
	y := yaml.NewDecoder(fp)
	if err := y.Decode(&cmd.GlobalConfig); err != nil {
		logger.Error("load yaml error:%s", err.Error())
	}

	router := mux.NewRouter()
	router = router.PathPrefix("/").Subrouter()
	router.Use(cmd.SetAuthHandler, cmd.AccessLog)
	domains := []string{}
	var routers []*mux.Router
	for _, domainName := range domains {
		routers = append(routers, router.Host("{bucket:.+}."+domainName).Subrouter())
	}
	routers = append(routers, router.PathPrefix("/{bucket}").Subrouter())

	bucket := cmd.Bucket{}
	object := cmd.Object{}

	for _, router := range routers {
		{
			//PutObjectPart
			router.Methods(http.MethodPut).Path("/{object:.+}").HandlerFunc(object.Put)
			//HeadObject
			router.Methods(http.MethodHead).Path("/{object:.+}").HandlerFunc(object.Head)
			//DeleteObject
			router.Methods(http.MethodDelete).Path("/{object:.+}").HandlerFunc(object.Delete)
			//GetObject
			router.Methods(http.MethodGet).Path("/{object:.+}").HandlerFunc(object.Get)
			//ListObjectsV1
			router.Methods(http.MethodGet).HandlerFunc(object.ListV1)
		}
		{
			//bucket
			router.Methods(http.MethodGet).Queries("location", "").HandlerFunc(bucket.Location)
			router.Methods(http.MethodPut).HandlerFunc(bucket.Create)
			router.Methods(http.MethodHead).HandlerFunc(bucket.Head)
			router.Methods(http.MethodDelete).HandlerFunc(bucket.Delete)
		}
	}
	router.Methods(http.MethodGet).Path(cmd.SlashSeparator).HandlerFunc(bucket.List)

	addr := ":8002"
	logger.Info("http start listen:%s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Println(err.Error())
	}
}
