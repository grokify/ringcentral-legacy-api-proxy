package handlers

import (
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	hum "github.com/grokify/gotilla/net/httputilmore"

	ru "github.com/grokify/go-ringcentral/clientutil"
	tu "github.com/grokify/gotilla/time/timeutil"
	ro "github.com/grokify/oauth2more/ringcentral"
)

const (
	Attachment = "attachment"
)

type LegacyMultipartFormParser struct {
	form *multipart.Form
}

func NewLegacyMultipartFormParser(form *multipart.Form) LegacyMultipartFormParser {
	return LegacyMultipartFormParser{form: form}
}

func (parser *LegacyMultipartFormParser) PasswordCredentials() ro.PasswordCredentials {
	return NewPasswordCredentialsLegacyMultipartForm(parser.form)
}

func (parser *LegacyMultipartFormParser) FaxRequest() ru.FaxRequest {
	return NewFaxRequestLegacyMultipartForm(parser.form)
}

func (parser *LegacyMultipartFormParser) Format() string {
	if vals, ok := parser.form.Value["Format"]; ok && len(vals) > 0 {
		return strings.ToLower(strings.TrimSpace(vals[0]))
	}
	if vals, ok := parser.form.Value["format"]; ok && len(vals) > 0 {
		return strings.ToLower(strings.TrimSpace(vals[0]))
	}
	return ""
}

func NewPasswordCredentialsLegacyMultipartForm(form *multipart.Form) ro.PasswordCredentials {
	var pwdCreds ro.PasswordCredentials
	if vals, ok := form.Value["Username"]; ok && len(vals) > 0 {
		pwdCreds.Username = strings.TrimSpace(vals[0])
	}
	if vals, ok := form.Value["Extension"]; ok && len(vals) > 0 {
		pwdCreds.Extension = strings.TrimSpace(vals[0])
	}
	if vals, ok := form.Value["Password"]; ok && len(vals) > 0 {
		pwdCreds.Password = strings.TrimSpace(vals[0])
	}
	return pwdCreds
}

// NewFaxRequestLegacyMultipartForm returns a REST API clientutil.FaxRequest
// given a Legacy RPC API `*multipart.Form`
// https://github.com/golang/go/blob/master/src/net/http/request.go#L237
// http://sanatgersappa.blogspot.com/2013/03/handling-multiple-file-uploads-in-go.html
func NewFaxRequestLegacyMultipartForm(form *multipart.Form) ru.FaxRequest {
	fax := ru.NewFaxRequest()
	if vals, ok := form.Value["Recipient"]; ok && len(vals) > 0 {
		for _, val := range vals {
			val = strings.TrimSpace(val)
			parts := strings.Split(val, "|")
			if len(parts) > 0 {
				fax.To = append(fax.To, strings.TrimSpace(parts[0]))
			}
		}
	}
	if vals, ok := form.Value["Coverpage"]; ok && len(vals) > 0 {
		for _, val := range vals {
			if coverPage, err := ru.FaxCoverPageNameToIndex(val); err == nil {
				fax.CoverIndex = int(coverPage)
			}
		}
	}
	if vals, ok := form.Value["Coverpagetext"]; ok && len(vals) > 0 {
		for _, val := range vals {
			if len(val) > 0 {
				fax.CoverPageText = val
			}
		}
	}
	if vals, ok := form.Value["Resolution"]; ok && len(vals) > 0 {
		for _, val := range vals {
			val = strings.ToLower(strings.TrimSpace(val))
			if val == "high" || val == "low" {
				fax.Resolution = strings.Title(val)
			}
		}
	}
	// GMT time in format dd:mm:yy hh:mm
	if vals, ok := form.Value["Sendtime"]; ok && len(vals) > 0 {
		for _, val := range vals {
			dt, err := tu.ParseFirst([]string{tu.DMYHM2, time.RFC3339}, val)
			if err == nil {
				fax.SendTime = &dt
			}
		}
	}
	if fileHeaders, ok := form.File["Attachment"]; ok {
		fax.FileHeaders = fileHeaders
	}
	return fax
}

func WriteFaxResponse(res http.ResponseWriter, resp *http.Response, err error, format string) {
	legacyResponseCode := Successful
	httpStatusCode := resp.StatusCode
	message := "Successful"
	if err != nil || resp.StatusCode >= 500 {
		legacyResponseCode = GenericError
		httpStatusCode = http.StatusInternalServerError
		message = "Generic error"
	} else if resp.StatusCode == 401 {
		legacyResponseCode = AuthorizationFailed
		httpStatusCode = http.StatusUnauthorized
		message = "Authorization failed"
	} else if resp.StatusCode >= 300 {
		legacyResponseCode = GenericError
		httpStatusCode = http.StatusInternalServerError
		message = "Generic error"
	} else {
		legacyResponseCode = Successful
		httpStatusCode = http.StatusOK
		message = "Successful"
	}
	if format == "json" {
		res.Header().Set(hum.HeaderContentType, hum.ContentTypeAppJsonUtf8)
		if httpStatusCode != http.StatusOK {
			resInfo := hum.ResponseInfo{
				StatusCode: httpStatusCode,
				Message:    message}
			res.Write(resInfo.ToJson())
		} else {
			bytes, err := hum.ResponseBody(resp)
			if err == nil {
				res.Write(bytes)
			} else {
				resInfo := hum.ResponseInfo{
					StatusCode: http.StatusInternalServerError,
					Message:    err.Error()}
				res.Write(resInfo.ToJson())
			}
		}
	} else {
		res.Header().Set(hum.HeaderContentType, hum.ContentTypeTextPlainUsAscii)
		res.Write([]byte(strconv.Itoa(int(legacyResponseCode))))
	}
}

type FaxResponseCode int

const (
	Successful          FaxResponseCode = iota // 0
	AuthorizationFailed                        // 1
	FaxingProhibited                           // 2
	NoFaxRecipients                            // 3
	NoFaxData                                  // 4
	GenericError                               // 5
)

// FaxResponseCodeToResponseInfo
/*
0 - Successful
1 - Authorization failed
2 - Faxing is prohibited for the account
3 - No recipients specified
4 - No fax data specified
5 - Generic error
*/
func FaxResponseCodeToResponseInfo(code FaxResponseCode) hum.ResponseInfo {
	switch code {
	case Successful:
		return hum.ResponseInfo{StatusCode: http.StatusOK, Message: "Successful"}
	case AuthorizationFailed:
		return hum.ResponseInfo{StatusCode: http.StatusUnauthorized, Message: "Authorization failed"}
	case FaxingProhibited:
		return hum.ResponseInfo{StatusCode: http.StatusForbidden, Message: "Faxing is prohibited for the account"}
	case NoFaxRecipients:
		return hum.ResponseInfo{StatusCode: http.StatusBadRequest, Message: "No recipients specified"}
	case NoFaxData:
		return hum.ResponseInfo{StatusCode: http.StatusBadRequest, Message: "No fax data specified"}
	default:
		return hum.ResponseInfo{StatusCode: http.StatusBadRequest, Message: "No fax data specified"}
	}
}
