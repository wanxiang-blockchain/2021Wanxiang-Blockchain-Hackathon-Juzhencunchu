// Copyright (c) 2015-2021 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"context"
	"encoding/hex"
	"github.com/minio/pkg/bucket/policy"
	"net/http"
	"s3Gateway/internal/auth"
	"s3Gateway/internal/etag"
	"s3Gateway/internal/hash"
	xhttp "s3Gateway/internal/http"
	"strings"
)

// Verify if request has AWS Signature Version '4'.
func isRequestSignatureV4(r *http.Request) bool {
	return strings.HasPrefix(r.Header.Get(xhttp.Authorization), signV4Algorithm)
}

// Verify if request has AWS Signature Version '2'.
func isRequestSignatureV2(r *http.Request) bool {
	return (!strings.HasPrefix(r.Header.Get(xhttp.Authorization), signV4Algorithm) &&
		strings.HasPrefix(r.Header.Get(xhttp.Authorization), signV2Algorithm))
}

// Verify if request has AWS PreSign Version '4'.
func isRequestPresignedSignatureV4(r *http.Request) bool {
	_, ok := r.URL.Query()[xhttp.AmzCredential]
	return ok
}

// Verify request has AWS PreSign Version '2'.
func isRequestPresignedSignatureV2(r *http.Request) bool {
	_, ok := r.URL.Query()[xhttp.AmzAccessKeyID]
	return ok
}

// Verify if request has AWS Post policy Signature Version '4'.
func isRequestPostPolicySignatureV4(r *http.Request) bool {
	return strings.Contains(r.Header.Get(xhttp.ContentType), "multipart/form-data") &&
		r.Method == http.MethodPost
}

// Verify if the request has AWS Streaming Signature Version '4'. This is only valid for 'PUT' operation.
func isRequestSignStreamingV4(r *http.Request) bool {
	return r.Header.Get(xhttp.AmzContentSha256) == streamingContentSHA256 &&
		r.Method == http.MethodPut
}

// Authorization type.
type authType int

// List of all supported auth types.
const (
	authTypeUnknown authType = iota
	authTypeAnonymous
	authTypePresigned
	authTypePresignedV2
	authTypePostPolicy
	authTypeStreamingSigned
	authTypeSigned
	authTypeSignedV2
	authTypeSTS
)

// Get request authentication type.
func getRequestAuthType(r *http.Request) authType {
	if isRequestSignatureV2(r) {
		return authTypeSignedV2
	} else if isRequestPresignedSignatureV2(r) {
		return authTypePresignedV2
	} else if isRequestSignStreamingV4(r) {
		return authTypeStreamingSigned
	} else if isRequestSignatureV4(r) {
		return authTypeSigned
	} else if isRequestPresignedSignatureV4(r) {
		return authTypePresigned
	} else if isRequestPostPolicySignatureV4(r) {
		return authTypePostPolicy
	} else if _, ok := r.URL.Query()[xhttp.Action]; ok {
		return authTypeSTS
	} else if _, ok := r.Header[xhttp.Authorization]; !ok {
		return authTypeAnonymous
	}
	return authTypeUnknown
}

// Verify if request has valid AWS Signature Version '2'.
func isReqAuthenticatedV2(r *http.Request) (s3Error APIErrorCode) {
	if isRequestSignatureV2(r) {
		return doesSignV2Match(r)
	}
	return doesPresignV2SignatureMatch(r)
}

func reqSignatureV4Verify(r *http.Request, region string, stype serviceType) (s3Error APIErrorCode) {
	sha256sum, err := getContentSha256Cksum(r, stype)
	if err != nil {

	}
	switch {
	case isRequestSignatureV4(r):
		return doesSignatureMatch(sha256sum, r, region, stype)
	case isRequestPresignedSignatureV4(r):
		return doesPresignedSignatureMatch(sha256sum, r, region, stype)
	default:
		return ErrAccessDenied
	}
}

// Verify if request has valid AWS Signature Version '4'.
func isReqAuthenticated(ctx context.Context, r *http.Request, region string, stype serviceType) (s3Error APIErrorCode) {
	if errCode := reqSignatureV4Verify(r, region, stype); errCode != ErrNone {
		return errCode
	}

	clientETag, err := etag.FromContentMD5(r.Header)
	if err != nil {
		return ErrInvalidDigest
	}

	// Extract either 'X-Amz-Content-Sha256' header or 'X-Amz-Content-Sha256' query parameter (if V4 presigned)
	// Do not verify 'X-Amz-Content-Sha256' if skipSHA256.
	var contentSHA256 []byte
	if skipSHA256 := skipContentSha256Cksum(r); !skipSHA256 && isRequestPresignedSignatureV4(r) {
		if sha256Sum, ok := r.URL.Query()[xhttp.AmzContentSha256]; ok && len(sha256Sum) > 0 {
			contentSHA256, err = hex.DecodeString(sha256Sum[0])
			if err != nil {
				return ErrContentSHA256Mismatch
			}
		}
	} else if _, ok := r.Header[xhttp.AmzContentSha256]; !skipSHA256 && ok {
		contentSHA256, err = hex.DecodeString(r.Header.Get(xhttp.AmzContentSha256))
		if err != nil || len(contentSHA256) == 0 {
			return ErrContentSHA256Mismatch
		}
	}

	// Verify 'Content-Md5' and/or 'X-Amz-Content-Sha256' if present.
	// The verification happens implicit during reading.
	reader, err := hash.NewReader(r.Body, -1, clientETag.String(), hex.EncodeToString(contentSHA256), -1)
	if err != nil {
		return toAPIErrorCode(ctx, err)
	}
	r.Body = reader
	return ErrNone
}

// List of all support S3 auth types.
var supportedS3AuthTypes = map[authType]struct{}{
	authTypeAnonymous:       {},
	authTypePresigned:       {},
	authTypePresignedV2:     {},
	authTypeSigned:          {},
	authTypeSignedV2:        {},
	authTypePostPolicy:      {},
	authTypeStreamingSigned: {},
}

// Validate if the authType is valid and supported.
func isSupportedS3AuthType(aType authType) bool {
	_, ok := supportedS3AuthTypes[aType]
	return ok
}

// setAuthHandler to validate authorization header for the incoming request.
func SetAuthHandler(h http.Handler) http.Handler {
	// handler for validating incoming authorization headers.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		aType := getRequestAuthType(r)
		if isSupportedS3AuthType(aType) {
			// Let top level caller validate for anonymous and known signed requests.
			h.ServeHTTP(w, r)
			return
		} else if aType == authTypeSTS {
			h.ServeHTTP(w, r)
			return
		}
		WriteErrorResponse(r.Context(), w, errorCodes.ToAPIErr(ErrSignatureVersionNotSupported), r.URL, guessIsBrowserReq(r))
	})
}

func validateSignature(ctx context.Context, atype authType, r *http.Request) (auth.Credentials, bool, map[string]interface{}, APIErrorCode) {
	var cred auth.Credentials
	var owner bool
	var s3Err APIErrorCode
	switch atype {
	case authTypeUnknown, authTypeStreamingSigned:
		return cred, owner, nil, ErrSignatureVersionNotSupported
	case authTypeSignedV2, authTypePresignedV2:
		if s3Err = isReqAuthenticatedV2(r); s3Err != ErrNone {
			return cred, owner, nil, s3Err
		}
		cred, owner, s3Err = getReqAccessKeyV2(r)
	case authTypePresigned, authTypeSigned:
		region := globalServerRegion
		if s3Err = isReqAuthenticated(ctx, r, region, serviceS3); s3Err != ErrNone {
			return cred, owner, nil, s3Err
		}
		cred, owner, s3Err = getReqAccessKeyV4(r, region, serviceS3)
	}
	if s3Err != ErrNone {
		return cred, owner, nil, s3Err
	}

	return cred, owner, map[string]interface{}{}, ErrNone
}

// Check request auth type verifies the incoming http request
// - validates the request signature
// - validates the policy action if anonymous tests bucket policies if any,
//   for authenticated requests validates IAM policies.
// returns APIErrorCode if any to be replied to the client.
func checkRequestAuthType(ctx context.Context, r *http.Request, action policy.Action, bucketName, objectName string) (cred auth.Credentials, s3Err APIErrorCode) {
	cred, _, s3Err = checkRequestAuthTypeCredential(ctx, r, action, bucketName, objectName)
	return cred, s3Err
}

// Check request auth type verifies the incoming http request
// - validates the request signature
// - validates the policy action if anonymous tests bucket policies if any,
//   for authenticated requests validates IAM policies.
// returns APIErrorCode if any to be replied to the client.
// Additionally returns the accessKey used in the request, and if this request is by an admin.
func checkRequestAuthTypeCredential(ctx context.Context, r *http.Request, action policy.Action, bucketName, objectName string) (cred auth.Credentials, owner bool, s3Err APIErrorCode) {
	switch getRequestAuthType(r) {
	case authTypeUnknown, authTypeStreamingSigned:
		return cred, owner, ErrSignatureVersionNotSupported
	case authTypePresignedV2, authTypeSignedV2:
		if s3Err = isReqAuthenticatedV2(r); s3Err != ErrNone {
			return cred, owner, s3Err
		}
		cred, owner, s3Err = getReqAccessKeyV2(r)
	case authTypeSigned, authTypePresigned:
		region := globalServerRegion
		switch action {
		case policy.GetBucketLocationAction, policy.ListAllMyBucketsAction:
			region = ""
		}
		if s3Err = isReqAuthenticated(ctx, r, region, serviceS3); s3Err != ErrNone {
			return cred, owner, s3Err
		}
		cred, owner, s3Err = getReqAccessKeyV4(r, region, serviceS3)
	}
	if s3Err != ErrNone {
		return cred, owner, s3Err
	}
	return cred, owner, ErrNone
}
