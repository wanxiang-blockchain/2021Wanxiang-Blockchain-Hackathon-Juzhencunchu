package cmd

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	xhttp "s3Gateway/internal/http"
	"strconv"
)

// mimeType represents various MIME type used API responses.
type mimeType string

const (
	// Means no response type.
	mimeNone mimeType = ""
	// Means response type is JSON.
	mimeJSON mimeType = "application/json"
	// Means response type is XML.
	mimeXML mimeType = "application/xml"
)

func WriteResponse(w http.ResponseWriter, statusCode int, response []byte, mType mimeType) {
	if mType != mimeNone {
		w.Header().Set(xhttp.ContentType, string(mType))
	}
	w.Header().Set(xhttp.ContentLength, strconv.Itoa(len(response)))
	w.WriteHeader(statusCode)
	if response != nil {
		w.Write(response)
	}
}

// WriteSuccessResponseXML writes success headers and response if any,
// with content-type set to `application/xml`.
func WriteSuccessResponseXML(w http.ResponseWriter, response []byte) {
	WriteResponse(w, http.StatusOK, response, mimeXML)
}

// WriteSuccessNoContent writes success headers with http status 204
func WriteSuccessNoContent(w http.ResponseWriter) {
	WriteResponse(w, http.StatusNoContent, nil, mimeNone)
}

// WriteRedirectSeeOther writes Location header with http status 303
func WriteRedirectSeeOther(w http.ResponseWriter, location string) {
	w.Header().Set(xhttp.Location, location)
	WriteResponse(w, http.StatusSeeOther, nil, mimeNone)
}

func WriteSuccessResponseHeadersOnly(w http.ResponseWriter) {
	WriteResponse(w, http.StatusOK, nil, mimeNone)
}

// WriteErrorResponse writes error headers
func WriteErrorResponse(ctx context.Context, w http.ResponseWriter, err *APIError, reqURL *url.URL, browser bool) {
	switch err.Code {
	case "SlowDown", "XMinioServerNotInitialized", "XMinioReadQuorum", "XMinioWriteQuorum":
		// Set retry-after header to indicate user-agents to retry request after 120secs.
		// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Retry-After
		w.Header().Set(xhttp.RetryAfter, "120")
	case "InvalidRegion":
		err.Description = fmt.Sprintf("Region does not match; expecting '%s'.", globalServerRegion)
	case "AuthorizationHeaderMalformed":
		err.Description = fmt.Sprintf("The authorization header is malformed; the region is wrong; expecting '%s'.", globalServerRegion)
	case "AccessDenied":
		// The request is from browser and also if browser
		// is enabled we need to redirect.
		if browser {
			w.Header().Set(xhttp.Location, reqURL.Path)
			w.WriteHeader(http.StatusTemporaryRedirect)
			return
		}
	}

	bucketName, ok := ctx.Value("bucketName").(string)
	if !ok {
		bucketName = ""
	}
	objectName, ok := ctx.Value("objectName").(string)
	if !ok {
		objectName = ""
	}
	region, ok := ctx.Value("region").(string)
	if !ok {
		region = ""
	}
	// Generate error response.
	errorResponse := getAPIErrorResponse(bucketName, objectName, region, err, reqURL.Path, w.Header().Get(xhttp.AmzRequestID), "-")
	encodedErrorResponse := EncodeResponse(errorResponse)
	WriteResponse(w, err.HTTPStatusCode, encodedErrorResponse, mimeXML)
}

func WriteErrorResponseHeadersOnly(w http.ResponseWriter, err APIError) {
	WriteResponse(w, err.HTTPStatusCode, nil, mimeNone)
}
func EncodeResponse(response interface{}) []byte {
	var bytesBuffer bytes.Buffer
	bytesBuffer.WriteString(xml.Header)
	e := xml.NewEncoder(&bytesBuffer)
	e.Encode(response)
	return bytesBuffer.Bytes()
}
func WriteErrorResponseString(ctx context.Context, w http.ResponseWriter, err APIError, reqURL *url.URL) {
	// Generate string error response.
	WriteResponse(w, err.HTTPStatusCode, []byte(err.Description), mimeNone)
}
