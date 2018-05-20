package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	hum "github.com/grokify/gotilla/net/httputilmore"

	rc "github.com/grokify/go-ringcentral/client"
	ru "github.com/grokify/go-ringcentral/clientutil"
)

// RingOutRequestParams represents the full list of request
// parameters that can be sent. All parameter names are lower-case.
// Supports both GET and POST.
type RingOutRequestParams struct {
	Cmd       string `schema:"cmd"`
	Username  string `schema:"username"`
	Ext       string `schema:"ext"`
	Password  string `schema:"password"`
	To        string `schema:"to"`
	From      string `schema:"from"`
	Clid      string `schema:"clid"`
	Prompt    string `schema:"prompt"`
	SessionID string `schema:"sessionid"`
	Format    string `schema:"format"`
}

// HasValidCommand returns true if `cmd` is set to a supported value.
func (params *RingOutRequestParams) HasValidCommand() bool {
	cmds := map[string]int{"call": 1, "list": 1, "status": 0, "cancel": 0}
	if val, ok := cmds[strings.ToLower(params.Cmd)]; ok && val == 1 {
		return true
	}
	return false
}

// PlayPrompt returns the prompt parameter converted to a boolean.
func (params *RingOutRequestParams) PlayPrompt() bool {
	if params.Prompt == "1" {
		return true
	}
	return false
}

func RingoutList(res http.ResponseWriter, apiClient *rc.APIClient, responseFormat string) {
	fmt.Println("LIST")
	info, resp, err := apiClient.CallHandlingSettingsApi.ListExtensionForwardingNumbers(
		context.Background(), "~", "~", map[string]interface{}{})
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		eresp := hum.ResponseInfo{StatusCode: 500, Message: err.Error()}
		res.Write(eresp.ToJson())
		return
	} else if resp.StatusCode >= 500 {
		res.WriteHeader(http.StatusInternalServerError)
		eresp := hum.ResponseInfo{StatusCode: resp.StatusCode, Message: fmt.Sprintf("REST API Response: %v", resp.StatusCode)}
		res.Write(eresp.ToJson())
		return
	} else if resp.StatusCode >= 400 {
		res.WriteHeader(http.StatusBadRequest)
		eresp := hum.ResponseInfo{StatusCode: resp.StatusCode, Message: fmt.Sprintf("REST API Response: %v", resp.StatusCode)}
		res.Write(eresp.ToJson())
		return
	} else if resp.StatusCode >= 300 {
		res.WriteHeader(http.StatusInternalServerError)
		eresp := hum.ResponseInfo{StatusCode: resp.StatusCode, Message: fmt.Sprintf("REST API Response: %v", resp.StatusCode)}
		res.Write(eresp.ToJson())
		return
	}
	bytes, err := json.Marshal(info)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	if responseFormat == "json" {
		res.Header().Set(hum.HeaderContentType, hum.ContentTypeAppJsonUtf8)
		res.Write(bytes)
	} else {
		res.Header().Set(hum.HeaderContentType, hum.ContentTypeTextPlainUsAscii)
		res.Write([]byte(ringoutListLegacyResponseBody(info.Records)))
	}
}

func ringoutListLegacyResponseBody(numberInfos []rc.ForwardingNumberInfo) string {
	parts := []string{}
	for _, numberInfo := range numberInfos {
		label := strings.TrimSpace(numberInfo.Label)
		number := strings.TrimSpace(numberInfo.PhoneNumber)
		rx := regexp.MustCompile(`^\+1(\d{10})$`)
		m := rx.FindStringSubmatch(number)
		if len(m) > 1 {
			number = m[1]
		}
		parts = append(parts, number)
		parts = append(parts, label)
	}
	return fmt.Sprintf("OK %s", strings.Join(parts, ";"))
}

func RingoutCall(res http.ResponseWriter, apiClient *rc.APIClient, ringOut ru.RingOutRequest, responseFormat string) {
	info, resp, err := apiClient.RingOutApi.MakeRingOutCallNew(
		context.Background(), "~", "~", *ringOut.Body())
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	} else if resp.StatusCode >= 500 {
		res.WriteHeader(http.StatusInternalServerError)
		return
	} else if resp.StatusCode >= 400 {
		res.WriteHeader(http.StatusBadRequest)
		return
	} else if resp.StatusCode >= 300 {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	bytes, err := json.Marshal(info)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	if responseFormat == "json" {
		res.Header().Set(hum.HeaderContentType, hum.ContentTypeAppJsonUtf8)
		res.Write(bytes)
	} else {
		res.Header().Set(hum.HeaderContentType, hum.ContentTypeTextPlainUsAscii)
		res.Write([]byte(fmt.Sprintf("OK %s", info.Id)))
	}
}
