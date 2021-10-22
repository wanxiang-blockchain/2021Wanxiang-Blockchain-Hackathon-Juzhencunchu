package cmd

import (
	"encoding/xml"
	"github.com/gorilla/mux"
	"github.com/minio/pkg/bucket/policy"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"s3Gateway/internal/etag"
	xhttp "s3Gateway/internal/http"
	mxml "s3Gateway/model/xml"
	"strconv"
	"strings"
)

type Object struct{}

func (*Object) Head(w http.ResponseWriter, r *http.Request) {
	ctx := newContext(r, w, "head-object")
	params := mux.Vars(r)
	log.Println(params)
	if cred, s3Error := checkRequestAuthType(ctx, r, policy.GetObjectAction, params["bucket"], params["object"]); s3Error != ErrNone {
		WriteErrorResponse(ctx, w, errorCodes.ToAPIErr(s3Error), r.URL, guessIsBrowserReq(r))
		return
	} else {
		if err := SetKey(ctx, cred); err != ErrNone {
			WriteErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrAuthHeaderEmpty), r.URL, guessIsBrowserReq(r))
			return
		}
	}
	headFile, err := HeadFile(ctx)
	if err != nil {
		WriteErrorResponse(ctx, w, err, r.URL, guessIsBrowserReq(r))
		return
	}
	for k, v := range headFile.Data.Header {
		w.Header().Set(k, v)
	}
	WriteSuccessResponseHeadersOnly(w)
}

func (*Object) Get(w http.ResponseWriter, r *http.Request) {
	ctx := newContext(r, w, "head-object")
	params := mux.Vars(r)
	log.Println(params)
	if cred, s3Error := checkRequestAuthType(ctx, r, policy.GetObjectAction, params["bucket"], params["object"]); s3Error != ErrNone {
		WriteErrorResponse(ctx, w, errorCodes.ToAPIErr(s3Error), r.URL, guessIsBrowserReq(r))
		return
	} else {
		if err := SetKey(ctx, cred); err != ErrNone {
			WriteErrorResponse(ctx, w, errorCodes.ToAPIErr(err), r.URL, guessIsBrowserReq(r))
			return
		}
	}
	m, err := GetCid(ctx)
	if err != nil {
		WriteErrorResponse(ctx, w, err, r.URL, guessIsBrowserReq(r))
		return
	}
	rep, err := DownloadFile(ctx, m.Data.StoreHost, m.Data.Cid)
	if err != nil {
		WriteErrorResponse(ctx, w, err, r.URL, guessIsBrowserReq(r))
		return
	}
	defer rep.Body.Close()

	writerHeader := w.Header()
	for k, v := range r.URL.Query() {
		if len(v) < 1 {
			continue
		}
		if strings.HasPrefix(k, "response-") {
			headerName := strings.TrimPrefix(k, "response-")
			writerHeader.Add(strings.ToUpper(headerName[:1])+headerName[1:], v[0])
		}
	}
	for k, vs := range rep.Header {
		for _, v := range vs {
			writerHeader.Add(k, v)
		}
	}
	io.Copy(w, rep.Body)
}
func (*Object) Put(w http.ResponseWriter, r *http.Request) {

	ctx := newContext(r, w, "head-object")
	if cred, _, _, _, err := calculateSeedSignature(r); err != ErrNone {
		WriteErrorResponse(ctx, w, errorCodes.ToAPIErr(err), r.URL, guessIsBrowserReq(r))
		return
	} else {
		if err := SetKey(ctx, cred); err != ErrNone {
			WriteErrorResponse(ctx, w, errorCodes.ToAPIErr(err), r.URL, guessIsBrowserReq(r))
			return
		}
	}

	_, err := etag.FromContentMD5(r.Header)
	if err != nil {
		WriteErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrInvalidDigest), r.URL, guessIsBrowserReq(r))
		return
	}
	size := r.ContentLength
	rAuthType := getRequestAuthType(r)
	if rAuthType == authTypeStreamingSigned {
		if sizeStr, ok := r.Header[xhttp.AmzDecodedContentLength]; ok {
			if sizeStr[0] == "" {
				WriteErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrMissingContentLength), r.URL, guessIsBrowserReq(r))
				return
			}
			size, err = strconv.ParseInt(sizeStr[0], 10, 64)
			if err != nil {
				WriteErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrInvalidDigest), r.URL, guessIsBrowserReq(r))
				return
			}
		}
	}
	if size == -1 {
		WriteErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrMissingContentLength), r.URL, guessIsBrowserReq(r))
		return
	}

	chuncek := httputil.NewChunkedReader(r.Body)
	re, errCode := FileUpdateCredential(ctx)
	if errCode != nil {
		WriteErrorResponse(ctx, w, errCode, r.URL, guessIsBrowserReq(r))
		return
	}
	_, errCode = FileUpdate(ctx, chuncek, re, size)
	if errCode != nil {
		WriteErrorResponse(ctx, w, errCode, r.URL, guessIsBrowserReq(r))
		return
	}
	WriteSuccessResponseHeadersOnly(w)
}

func (*Object) ListV1(w http.ResponseWriter, r *http.Request) {
	ctx := newContext(r, w, "head-object")
	params := mux.Vars(r)
	log.Println(params)
	if cred, s3Error := checkRequestAuthType(ctx, r, policy.GetObjectAction, params["bucket"], params["object"]); s3Error != ErrNone {
		WriteErrorResponse(ctx, w, errorCodes.ToAPIErr(s3Error), r.URL, guessIsBrowserReq(r))
		return
	} else {
		if err := SetKey(ctx, cred); err != ErrNone {
			WriteErrorResponse(ctx, w, errorCodes.ToAPIErr(err), r.URL, guessIsBrowserReq(r))
			return
		}
	}
	out, err := ListFile(ctx)
	if err != nil {
		WriteErrorResponse(ctx, w, err, r.URL, guessIsBrowserReq(r))
		return
	}
	response := mxml.ListObjectResult{}
	decoder := xml.NewDecoder(strings.NewReader(out.Data.Data))
	if err := decoder.Decode(&response); err != nil {
		WriteErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrMalformedXML), r.URL, guessIsBrowserReq(r))
	}
	WriteSuccessResponseXML(w, EncodeResponse(response))
}
func (*Object) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := newContext(r, w, "head-object")
	params := mux.Vars(r)
	log.Println(params)
	if cred, s3Error := checkRequestAuthType(ctx, r, policy.GetObjectAction, params["bucket"], params["object"]); s3Error != ErrNone {
		WriteErrorResponse(ctx, w, errorCodes.ToAPIErr(s3Error), r.URL, guessIsBrowserReq(r))
		return
	} else {
		if err := SetKey(ctx, cred); err != ErrNone {
			WriteErrorResponse(ctx, w, errorCodes.ToAPIErr(err), r.URL, guessIsBrowserReq(r))
			return
		}
	}
	if err := DelFile(ctx); err != nil {
		WriteErrorResponse(ctx, w, err, r.URL, guessIsBrowserReq(r))
		return
	}
	WriteSuccessNoContent(w)
}
