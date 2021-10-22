package cmd

import (
	"context"
	"github.com/gorilla/mux"
	"net/http"
	"net/url"
	"path"
	"s3Gateway/internal/auth"
	xhttp "s3Gateway/internal/http"
	"s3Gateway/internal/logger"
	"strings"
	"time"
)

// guessIsBrowserReq - returns true if the request is browser.
// This implementation just validates user-agent and
// looks for "Mozilla" string. This is no way certifiable
// way to know if the request really came from a browser
// since User-Agent's can be arbitrary. But this is just
// a best effort function.
func guessIsBrowserReq(req *http.Request) bool {
	if req == nil {
		return false
	}
	aType := getRequestAuthType(req)
	return strings.Contains(req.Header.Get("User-Agent"), "Mozilla") && aType == authTypeAnonymous
}
func trimLeadingSlash(ep string) string {
	if len(ep) > 0 && ep[0] == '/' {
		// Path ends with '/' preserve it
		if ep[len(ep)-1] == '/' && len(ep) > 1 {
			ep = path.Clean(ep)
			ep += SlashSeparator
		} else {
			ep = path.Clean(ep)
		}
		ep = ep[1:]
	}
	return ep
}

// unescapeGeneric is similar to url.PathUnescape or url.QueryUnescape
// depending on input, additionally also handles situations such as
// `//` are normalized as `/`, also removes any `/` prefix before
// returning.
func unescapeGeneric(p string, escapeFn func(string) (string, error)) (string, error) {
	ep, err := escapeFn(p)
	if err != nil {
		return "", err
	}
	return trimLeadingSlash(ep), nil
}

// unescapePath is similar to unescapeGeneric but for specifically
// path unescaping.
func unescapePath(p string) (string, error) {
	return unescapeGeneric(p, url.PathUnescape)
}

// similar to unescapeGeneric but never returns any error if the unescaping
// fails, returns the input as is in such occasion, not meant to be
// used where strict validation is expected.
func likelyUnescapeGeneric(p string, escapeFn func(string) (string, error)) string {
	ep, err := unescapeGeneric(p, escapeFn)
	if err != nil {
		return p
	}
	return ep
}

type ReqInfo struct {
	Writer     http.ResponseWriter
	Request    *http.Request
	RequestID  string // x-amz-request-id
	API        string // API name - GetObject PutObject NewMultipartUpload etc.
	BucketName string // Bucket name
	ObjectName string // Object name
	AccessKey  string // Access Key
	SecretKey  string // secret Key
}

// Returns context with ReqInfo details set in the context.
func newContext(r *http.Request, w http.ResponseWriter, api string) context.Context {
	vars := mux.Vars(r)
	bucket := vars["bucket"]
	object := likelyUnescapeGeneric(vars["object"], url.PathUnescape)
	prefix := likelyUnescapeGeneric(vars["prefix"], url.QueryUnescape)
	if prefix != "" {
		object = prefix
	}
	reqInfo := &ReqInfo{
		Writer:     w,
		Request:    r,
		RequestID:  w.Header().Get(xhttp.AmzRequestID),
		API:        api,
		BucketName: bucket,
		ObjectName: object,
	}
	return context.WithValue(r.Context(), requestInfo, reqInfo)
}
func GetReqInfo(ctx context.Context) (*ReqInfo, bool) {
	v, ok := ctx.Value(requestInfo).(*ReqInfo)
	return v, ok
}
func SetKey(ctx context.Context, cred auth.Credentials) APIErrorCode {
	q, ok := GetReqInfo(ctx)
	if !ok {
		return ErrAuthHeaderEmpty
	}
	q.AccessKey = cred.AccessKey
	q.SecretKey = cred.SecretKey
	return ErrNone
}

type ResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *ResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

//AccessLog 记录访问日志
func AccessLog(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wc := &ResponseWriter{
			statusCode:     http.StatusOK,
			ResponseWriter: w,
		}
		for key, values := range r.Header {
			logger.Debug("header key=>%s,value=>%s", key, strings.Join(values, ";"))
		}

		defer func() {
			logger.Info("url path %s,time %s,response status code %d", r.URL.Path, time.Since(start), wc.statusCode)
			if err := recover(); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}()
		h.ServeHTTP(wc, r)
	})
}
