package cmd

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/url"
	mjson "s3Gateway/model/json"
	"sort"
	"strconv"
	"strings"
	"time"
)

//请求openApi的接口

func SignatureOpenApi(nonce, secret, timestamp string) string {
	params := []string{nonce, secret, timestamp}
	sort.Strings(params)
	s := strings.Join(params, "")
	o := sha1.New()
	o.Write([]byte(s))
	return hex.EncodeToString(o.Sum(nil))
}
func RandomNonce() string {
	b := bytes.NewBuffer(nil)
	for i := 0; i < 10; i++ {
		b.WriteByte(byte(rand.Intn(9) + 48))
	}
	return b.String()
}

type MetaData struct {
	//url
	Url string
	//header
	Header map[string][]string
	//body
	Body   io.Reader
	secret string
}

func NewMetAData(targetUrl, appId, secret string) *MetaData {
	d := &MetaData{
		Url:    targetUrl,
		Header: make(map[string][]string),
		secret: secret,
	}
	d.Header["AppId"] = append(d.Header["AppId"], appId)
	d.Header["AppVersion"] = append(d.Header["AppVersion"], "1.0.0")
	d.Header["Content-Type"] = append(d.Header["Content-Type"], "application/x-www-form-urlencoded")
	d.Header["User-Agent"] = append(d.Header["User-Agent"], "chrome")
	return d
}
func DoRequest(method string, metaData *MetaData) (*http.Response, error) {
	var req *http.Request
	var err error
	if method == http.MethodGet {
		req, err = http.NewRequest(method, metaData.Url, nil)
	} else {
		req, err = http.NewRequest(method, metaData.Url, metaData.Body)
	}
	if err != nil {
		return nil, err
	}
	for key, value := range metaData.Header {
		for _, v := range value {
			req.Header.Add(key, v)
		}
	}
	//签名
	nonce := RandomNonce()
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	sing := SignatureOpenApi(nonce, metaData.secret, timestamp)
	signature := url.Values{}
	signature.Add("signature", sing)
	signature.Add("nonce", nonce)
	signature.Add("timestamp", timestamp)
	req.Header.Add("Signature", signature.Encode())
	http.DefaultClient.Timeout = 10 * time.Second
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return response, nil
}

//StatusOk 判断是否为200
func StatusOk(response *http.Response) APIErrorCode {
	if response.StatusCode != http.StatusOK {
		return ErrBusy
	}
	return ErrNone
}

//UserSecret s3 查询用户 ak
func UserSecret(ak string) (*mjson.CompanySecret, error) {
	x := fmt.Sprintf("%s/%s", GlobalConfig.OpenApi.Host, GlobalConfig.OpenApi.Paths.SecretByAppid)
	meta := NewMetAData(x, "bingheyun2", "896f669fc2ba4f31961fb76157a54d9f")
	data := url.Values{}
	data.Set("app_id", ak)
	meta.Body = strings.NewReader(data.Encode())
	rep, err := DoRequest(http.MethodPost, meta)
	if err != nil {
		return nil, err
	}
	if errCode := StatusOk(rep); errCode != ErrNone {
		return nil, errors.New(rep.Status)
	}
	defer rep.Body.Close()
	csDecoder := json.NewDecoder(rep.Body)
	cs := mjson.CompanySecret{}
	if err := csDecoder.Decode(&cs); err != nil {
		return nil, err
	}
	if cs.Code != 0 {
		return nil, errors.New(cs.Msg)
	}
	return &cs, nil
}

//ListFile 文件列表
func ListFile(ctx context.Context) (*mjson.ListFile, *APIError) {
	reqInfo, ok := GetReqInfo(ctx)
	if !ok {
		return nil, errorCodes.ToAPIErr(ErrAuthHeaderEmpty)
	}
	meta := NewMetAData(fmt.Sprintf("%s/%s", GlobalConfig.OpenApi.Host, GlobalConfig.OpenApi.Paths.ListFile), reqInfo.AccessKey, reqInfo.SecretKey)

	v := url.Values{}
	v.Add("bucket_name", reqInfo.BucketName)
	meta.Body = strings.NewReader(v.Encode())
	rep, err := DoRequest(http.MethodPost, meta)
	if err != nil {
		return nil, errorCodes.ToAPIErr(ErrBusy)
	}
	if errCode := StatusOk(rep); errCode != ErrNone {
		return nil, errorCodes.ToAPIErr(errCode)
	}
	defer rep.Body.Close()
	var files mjson.ListFile
	decoder := json.NewDecoder(rep.Body)
	if err := decoder.Decode(&files); err != nil {
		return nil, errorCodes.ToAPIErr(ErrBusy)
	}
	if files.Code == 0 && files.Data.HttpStatus == http.StatusOK {
		return &files, nil
	}
	return nil, &APIError{
		Code:           files.Data.Error.Code,
		Description:    files.Data.Error.Message,
		HTTPStatusCode: files.Data.HttpStatus,
	}
}

//DownloadFile 文件下载
func DownloadFile(ctx context.Context, storeHost, cid string) (*http.Response, *APIError) {
	reqInfo, ok := GetReqInfo(ctx)
	if !ok {
		return nil, errorCodes.ToAPIErr(ErrAuthHeaderEmpty)
	}
	meta := NewMetAData(fmt.Sprintf("%s/%s", storeHost, GlobalConfig.OpenApi.Paths.DownloadFile), reqInfo.AccessKey, reqInfo.SecretKey)
	meta.Header["from"] = append(meta.Header["from"], "s3")
	v := url.Values{}
	v.Add("cid", cid)
	v.Add("bucket_name", reqInfo.BucketName)
	v.Add("key", reqInfo.ObjectName)
	meta.Url = fmt.Sprintf("%s?%s", meta.Url, v.Encode())

	rep, err := DoRequest(http.MethodGet, meta)
	if err != nil {
		return nil, errorCodes.ToAPIErr(ErrBusy)
	}
	if errCode := StatusOk(rep); errCode != ErrNone {
		return nil, errorCodes.ToAPIErr(errCode)
	}
	return rep, nil
}

//FileUpdateCredential 文件上传
func FileUpdateCredential(ctx context.Context) (*mjson.UploadCredential, *APIError) {
	reqInfo, ok := GetReqInfo(ctx)
	if !ok {
		return nil, errorCodes.ToAPIErr(ErrAuthHeaderEmpty)
	}
	x := fmt.Sprintf("%s/%s", GlobalConfig.OpenApi.Host, GlobalConfig.OpenApi.Paths.UploadCredential)
	meta := NewMetAData(x, reqInfo.AccessKey, reqInfo.SecretKey)
	meta.Header["form"] = append(meta.Header["from"], "s3")
	data := url.Values{}
	data.Set("bucket_name", reqInfo.BucketName)
	data.Set("key", reqInfo.ObjectName)

	meta.Body = strings.NewReader(data.Encode())
	rep, err := DoRequest(http.MethodPost, meta)
	if err != nil {
		return nil, errorCodes.ToAPIErr(ErrBusy)
	}
	if errCode := StatusOk(rep); errCode != ErrNone {
		return nil, errorCodes.ToAPIErr(ErrBusy)
	}
	defer rep.Body.Close()
	credentialDecoder := json.NewDecoder(rep.Body)
	credential := mjson.UploadCredential{}
	if err := credentialDecoder.Decode(&credential); err != nil {
		return nil, errorCodes.ToAPIErr(ErrBusy)
	}
	//没有返回httpStatus
	if credential.Code == 0 && credential.Data.HttpStatus == 0 {
		return &credential, nil

	}
	return nil, &APIError{
		Code:           credential.Data.Error.Code,
		Description:    credential.Data.Error.Message,
		HTTPStatusCode: credential.Data.HttpStatus,
	}
}
func FileUpdate(ctx context.Context, file io.Reader, credential *mjson.UploadCredential, fileSize int64) (*mjson.UploadFile, *APIError) {
	reqInfo, ok := GetReqInfo(ctx)
	if !ok {
		return nil, errorCodes.ToAPIErr(ErrAuthHeaderEmpty)
	}
	x := fmt.Sprintf("%s/%s", credential.Data.StoreHost, GlobalConfig.OpenApi.Paths.UploadFile)
	meta := NewMetAData(x, reqInfo.AccessKey, reqInfo.SecretKey)
	meta.Header["Credential"] = []string{credential.Data.Credential}
	meta.Header["from"] = []string{"s3"}
	meta.Header["EventId"] = []string{credential.Data.EventId}
	meta.Header["BucketName"] = []string{reqInfo.BucketName}
	meta.Header["FileName"] = []string{reqInfo.ObjectName}
	meta.Header["key"] = []string{reqInfo.ObjectName}
	meta.Header["FileSize"] = []string{strconv.FormatInt(fileSize, 10)}
	writer := bytes.NewBuffer(nil)
	meta.Body = writer
	body := multipart.NewWriter(writer)
	meta.Header["Content-Type"] = []string{body.FormDataContentType()}
	fp, err := body.CreateFormFile("file", reqInfo.ObjectName)
	if err != nil {
		return nil, errorCodes.ToAPIErr(ErrBusy)
	}
	if i, err := io.Copy(fp, file); err != nil {
		return nil, errorCodes.ToAPIErr(ErrBusy)
	} else {
		log.Println(i)
	}
	body.Close()

	rep, err := DoRequest(http.MethodPost, meta)
	if err != nil {
		return nil, errorCodes.ToAPIErr(ErrBusy)
	}
	if rep.StatusCode != http.StatusCreated {
		return nil, errorCodes.ToAPIErr(ErrNoSuchBucket)
	}
	defer rep.Body.Close()
	uploadDecoder := json.NewDecoder(rep.Body)
	uf := mjson.UploadFile{}
	if err := uploadDecoder.Decode(&uf); err != nil {
		return nil, errorCodes.ToAPIErr(ErrBusy)
	}
	if uf.Code != 0 || uf.Data.HttpStatus != http.StatusOK {
		return nil, &APIError{
			Code:           credential.Data.Error.Code,
			Description:    credential.Data.Error.Message,
			HTTPStatusCode: credential.Data.HttpStatus,
		}
	}
	return &uf, nil
}

//DelFile 删除文件
func DelFile(ctx context.Context) *APIError {
	reqInfo, ok := GetReqInfo(ctx)
	if !ok {
		return errorCodes.ToAPIErr(ErrAuthHeaderEmpty)
	}
	meta := NewMetAData(fmt.Sprintf("%s/%s", GlobalConfig.OpenApi.Host, GlobalConfig.OpenApi.Paths.DeleteFile), reqInfo.AccessKey, reqInfo.SecretKey)

	val := url.Values{}
	val.Add("key", reqInfo.ObjectName)
	val.Add("bucket_name", reqInfo.BucketName)
	meta.Body = strings.NewReader(val.Encode())
	rep, err := DoRequest(http.MethodPost, meta)
	if err != nil {
		return errorCodes.ToAPIErr(ErrBusy)
	}
	if errCode := StatusOk(rep); errCode != ErrNone {
		return errorCodes.ToAPIErr(errCode)
	}
	defer rep.Body.Close()
	decoder := json.NewDecoder(rep.Body)
	m := mjson.DeleteFile{}
	if err := decoder.Decode(&m); err != nil {
		return errorCodes.ToAPIErr(ErrBusy)
	}
	if m.Code == 0 && m.Data.HttpStatus == http.StatusNoContent {
		return nil
	}
	return &APIError{
		Code:           m.Data.Error.Code,
		Description:    m.Data.Error.Message,
		HTTPStatusCode: m.Data.HttpStatus,
	}
}
func HeadFile(ctx context.Context) (*mjson.HeadFile, *APIError) {
	reqInfo, ok := GetReqInfo(ctx)
	if !ok {
		return nil, errorCodes.ToAPIErr(ErrAuthHeaderEmpty)
	}
	meta := NewMetAData(fmt.Sprintf("%s/%s", GlobalConfig.OpenApi.Host, GlobalConfig.OpenApi.Paths.HeadFile), reqInfo.AccessKey, reqInfo.SecretKey)
	meta.Header["form"] = append(meta.Header["from"], "s3")
	payload := url.Values{}
	payload.Add("bucket_name", reqInfo.BucketName)
	payload.Add("key", reqInfo.ObjectName)
	meta.Body = strings.NewReader(payload.Encode())
	rep, err := DoRequest(http.MethodPost, meta)
	if err != nil {
		return nil, errorCodes.ToAPIErr(ErrBusy)
	}
	if errCode := StatusOk(rep); errCode != ErrNone {
		return nil, errorCodes.ToAPIErr(errCode)
	}
	m := mjson.HeadFile{}
	defer rep.Body.Close()
	decoder := json.NewDecoder(rep.Body)
	if err := decoder.Decode(&m); err != nil {
		return nil, errorCodes.ToAPIErr(ErrBusy)
	}
	if m.Code == 0 && m.Data.HttpStatus == http.StatusOK {
		return &m, nil
	}
	return nil, &APIError{
		Code:           m.Data.Error.Code,
		Description:    m.Data.Error.Message,
		HTTPStatusCode: m.Data.HttpStatus,
	}
}

//GetCid 换取cid
func GetCid(ctx context.Context) (*mjson.GetCid, *APIError) {
	reqInfo, ok := GetReqInfo(ctx)
	if !ok {
		return nil, errorCodes.ToAPIErr(ErrAuthHeaderEmpty)
	}
	meta := NewMetAData(fmt.Sprintf("%s/%s", GlobalConfig.OpenApi.Host, GlobalConfig.OpenApi.Paths.GetCid), reqInfo.AccessKey, reqInfo.SecretKey)
	payload := url.Values{}
	payload.Add("bucket_name", reqInfo.BucketName)
	payload.Add("key", reqInfo.ObjectName)
	meta.Body = strings.NewReader(payload.Encode())
	rep, err := DoRequest(http.MethodPost, meta)
	if err != nil {
		return nil, errorCodes.ToAPIErr(ErrBusy)
	}
	if errCode := StatusOk(rep); errCode != ErrNone {
		return nil, errorCodes.ToAPIErr(errCode)
	}
	m := mjson.GetCid{}
	defer rep.Body.Close()
	decoder := json.NewDecoder(rep.Body)
	if err := decoder.Decode(&m); err != nil {
		return nil, errorCodes.ToAPIErr(ErrBusy)
	}
	if m.Code != 0 {
		return nil, errorCodes.ToAPIErr(ErrBusy)
	}
	return &m, nil
}

//CreateBucket 创建桶
func CreateBucket(ctx context.Context) *APIError {
	reqInfo, ok := GetReqInfo(ctx)
	if !ok {
		return errorCodes.ToAPIErr(ErrAuthHeaderEmpty)
	}
	meta := NewMetAData(fmt.Sprintf("%s/%s", GlobalConfig.OpenApi.Host, GlobalConfig.OpenApi.Paths.CreateBucket), reqInfo.AccessKey, reqInfo.SecretKey)
	payload := url.Values{}
	payload.Add("bucket_name", reqInfo.BucketName)
	meta.Body = strings.NewReader(payload.Encode())
	rep, err := DoRequest(http.MethodPost, meta)
	if err != nil {
		return errorCodes.ToAPIErr(ErrBusy)
	}
	if errCode := StatusOk(rep); errCode != ErrNone {
		return errorCodes.ToAPIErr(ErrBusy)
	}
	m := mjson.CreateBucket{}
	defer rep.Body.Close()
	decoder := json.NewDecoder(rep.Body)
	if err := decoder.Decode(&m); err != nil {
		return errorCodes.ToAPIErr(ErrBusy)
	}
	if m.Code == 0 && m.Data.HttpStatus == http.StatusOK {
		return nil
	}
	return &APIError{
		Code:           m.Data.Error.Code,
		Description:    m.Data.Error.Message,
		HTTPStatusCode: m.Data.HttpStatus,
	}
}

//DelBucket 删除桶
func DelBucket(ctx context.Context) *APIError {
	reqInfo, ok := GetReqInfo(ctx)
	if !ok {
		return errorCodes.ToAPIErr(ErrAuthHeaderEmpty)
	}
	meta := NewMetAData(fmt.Sprintf("%s/%s", GlobalConfig.OpenApi.Host, GlobalConfig.OpenApi.Paths.DeleteBucket), reqInfo.AccessKey, reqInfo.SecretKey)
	payload := url.Values{}
	payload.Add("bucket_name", reqInfo.BucketName)
	meta.Body = strings.NewReader(payload.Encode())
	rep, err := DoRequest(http.MethodPost, meta)
	if err != nil {
		return errorCodes.ToAPIErr(ErrBusy)
	}
	if errCode := StatusOk(rep); errCode != ErrNone {
		return errorCodes.ToAPIErr(errCode)
	}
	m := mjson.DeleteBucket{}
	defer rep.Body.Close()
	decoder := json.NewDecoder(rep.Body)
	if err := decoder.Decode(&m); err != nil {
		return errorCodes.ToAPIErr(ErrBusy)
	}
	if m.Code == 0 && m.Data.HttpStatus == http.StatusNoContent {
		return nil
	}

	return &APIError{
		Code:           m.Data.Error.Code,
		Description:    m.Data.Error.Message,
		HTTPStatusCode: m.Data.HttpStatus,
	}
}
func ListBucket(ctx context.Context) (*mjson.ListBucket, *APIError) {
	reqInfo, ok := GetReqInfo(ctx)
	if !ok {
		return nil, errorCodes.ToAPIErr(ErrAuthHeaderEmpty)
	}
	meta := NewMetAData(fmt.Sprintf("%s/%s", GlobalConfig.OpenApi.Host, GlobalConfig.OpenApi.Paths.ListBucket), reqInfo.AccessKey, reqInfo.SecretKey)
	payload := url.Values{}
	payload.Add("bucket_name", reqInfo.BucketName)
	meta.Body = strings.NewReader(payload.Encode())
	rep, err := DoRequest(http.MethodPost, meta)
	if err != nil {
		return nil, errorCodes.ToAPIErr(ErrBusy)
	}
	if errCode := StatusOk(rep); errCode != ErrNone {
		return nil, errorCodes.ToAPIErr(errCode)
	}
	m := mjson.ListBucket{}
	defer rep.Body.Close()
	decoder := json.NewDecoder(rep.Body)
	if err := decoder.Decode(&m); err != nil {
		return nil, errorCodes.ToAPIErr(ErrBusy)
	}
	if m.Code == 0 && m.Data.HttpStatus == http.StatusOK {
		return &m, nil
	}
	return nil, &APIError{
		Code:           m.Data.Error.Code,
		Description:    m.Data.Error.Message,
		HTTPStatusCode: m.Data.HttpStatus,
	}
}

func HeadBucket(ctx context.Context) *APIError {
	reqInfo, ok := GetReqInfo(ctx)
	if !ok {
		return errorCodes.ToAPIErr(ErrAuthHeaderEmpty)
	}
	meta := NewMetAData(fmt.Sprintf("%s/%s", GlobalConfig.OpenApi.Host, GlobalConfig.OpenApi.Paths.HeadBucket), reqInfo.AccessKey, reqInfo.SecretKey)
	payload := url.Values{}
	payload.Add("bucket_name", reqInfo.BucketName)
	meta.Body = strings.NewReader(payload.Encode())
	rep, err := DoRequest(http.MethodPost, meta)
	if err != nil {
		return errorCodes.ToAPIErr(ErrBusy)
	}
	if errCode := StatusOk(rep); errCode != ErrNone {
		return errorCodes.ToAPIErr(errCode)
	}
	m := mjson.CreateBucket{}
	defer rep.Body.Close()
	decoder := json.NewDecoder(rep.Body)
	if err := decoder.Decode(&m); err != nil {
		return errorCodes.ToAPIErr(ErrBusy)
	}
	if m.Code == 0 && m.Data.HttpStatus == http.StatusOK {
		return nil
	}
	return &APIError{
		Code:           m.Data.Error.Code,
		Description:    m.Data.Error.Message,
		HTTPStatusCode: m.Data.HttpStatus,
	}
}
