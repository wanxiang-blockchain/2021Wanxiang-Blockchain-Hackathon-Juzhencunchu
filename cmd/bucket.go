package cmd

import (
	"encoding/xml"
	"github.com/gorilla/mux"
	"github.com/minio/pkg/bucket/policy"
	"log"
	"net/http"
	mxml "s3Gateway/model/xml"
	"strings"
)

type Bucket struct{}

func (*Bucket) Create(w http.ResponseWriter, r *http.Request) {
	ctx := newContext(r, w, "create-Bucket")
	params := mux.Vars(r)
	if cred, s3Error := checkRequestAuthType(ctx, r, policy.CreateBucketAction, params["bucket"], ""); s3Error != ErrNone {
		WriteErrorResponse(ctx, w, errorCodes.ToAPIErr(s3Error), r.URL, guessIsBrowserReq(r))
		return
	} else {
		if err := SetKey(ctx, cred); err != ErrNone {
			WriteErrorResponse(ctx, w, errorCodes.ToAPIErr(err), r.URL, guessIsBrowserReq(r))
			return
		}
	}
	s3Err := CreateBucket(ctx)
	if s3Err != nil {
		WriteErrorResponse(ctx, w, s3Err, r.URL, guessIsBrowserReq(r))
		return
	}
	WriteSuccessResponseHeadersOnly(w)
}

func (*Bucket) Head(w http.ResponseWriter, r *http.Request) {
	ctx := newContext(r, w, "head-Bucket")
	params := mux.Vars(r)
	log.Println(params)
	if cred, s3Error := checkRequestAuthType(ctx, r, policy.CreateBucketAction, params["bucket"], ""); s3Error != ErrNone {
		WriteErrorResponse(ctx, w, errorCodes.ToAPIErr(s3Error), r.URL, guessIsBrowserReq(r))
		return
	} else {
		if err := SetKey(ctx, cred); err != ErrNone {
			WriteErrorResponse(ctx, w, errorCodes.ToAPIErr(err), r.URL, guessIsBrowserReq(r))
			return
		}
	}
	if s3err := HeadBucket(ctx); s3err != nil {
		WriteErrorResponse(ctx, w, s3err, r.URL, guessIsBrowserReq(r))
		return
	}
	WriteSuccessResponseHeadersOnly(w)
}

func (*Bucket) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := newContext(r, w, "delete-Bucket")
	params := mux.Vars(r)
	log.Println(params)
	if cred, s3Error := checkRequestAuthType(ctx, r, policy.CreateBucketAction, params["bucket"], ""); s3Error != ErrNone {
		WriteErrorResponse(ctx, w, errorCodes.ToAPIErr(s3Error), r.URL, guessIsBrowserReq(r))
		return
	} else {
		if err := SetKey(ctx, cred); err != ErrNone {
			WriteErrorResponse(ctx, w, errorCodes.ToAPIErr(err), r.URL, guessIsBrowserReq(r))
			return
		}
	}
	if s3err := DelBucket(ctx); s3err != nil {
		WriteErrorResponse(ctx, w, s3err, r.URL, guessIsBrowserReq(r))
		return
	}
	WriteSuccessNoContent(w)
}
func (*Bucket) List(w http.ResponseWriter, r *http.Request) {
	ctx := newContext(r, w, "list-Bucket")
	params := mux.Vars(r)
	log.Println(params)
	if cred, s3Error := checkRequestAuthType(ctx, r, policy.CreateBucketAction, params["bucket"], ""); s3Error != ErrNone {
		WriteErrorResponse(ctx, w, errorCodes.ToAPIErr(s3Error), r.URL, guessIsBrowserReq(r))
		return
	} else {
		if err := SetKey(ctx, cred); err != ErrNone {
			WriteErrorResponse(ctx, w, errorCodes.ToAPIErr(err), r.URL, guessIsBrowserReq(r))
			return
		}
	}
	out, s3err := ListBucket(ctx)
	if s3err != nil {
		WriteErrorResponse(ctx, w, s3err, r.URL, guessIsBrowserReq(r))
		return
	}
	buckets := mxml.ListBucket{}
	decoder := xml.NewDecoder(strings.NewReader(out.Data.Data))
	if err := decoder.Decode(&buckets); err != nil {
		WriteErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrMalformedXML), r.URL, guessIsBrowserReq(r))
		return
	}
	WriteSuccessResponseXML(w, EncodeResponse(buckets))
}
func (*Bucket) Location(w http.ResponseWriter, r *http.Request) {
	ctx := newContext(r, w, "location-Bucket")
	params := mux.Vars(r)
	log.Println(params)
	if cred, s3Error := checkRequestAuthType(ctx, r, policy.CreateBucketAction, params["bucket"], ""); s3Error != ErrNone {
		WriteErrorResponse(ctx, w, errorCodes.ToAPIErr(s3Error), r.URL, guessIsBrowserReq(r))
		return
	} else {
		if err := SetKey(ctx, cred); err != ErrNone {
			WriteErrorResponse(ctx, w, errorCodes.ToAPIErr(err), r.URL, guessIsBrowserReq(r))
			return
		}
	}
	body := mxml.CreateBucket{
		XMLName:  xml.Name{Local: "LocationConstraint"},
		Location: "ap-east-1",
	}
	WriteSuccessResponseXML(w, EncodeResponse(body))
}
