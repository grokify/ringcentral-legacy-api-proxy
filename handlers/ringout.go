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

	"github.com/grokify/gotilla/net/anyhttp"
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

func NewRingOutRequestParamsFromAnyArgs(args anyhttp.Args) RingOutRequestParams {
	return RingOutRequestParams{
		Cmd:       args.GetString("cmd"),
		Username:  args.GetString("username"),
		Ext:       args.GetString("ext"),
		Password:  args.GetString("password"),
		To:        args.GetString("to"),
		From:      args.GetString("from"),
		Clid:      args.GetString("clid"),
		Prompt:    args.GetString("prompt"),
		SessionID: args.GetString("sessionid"),
		Format:    args.GetString("format"),
	}
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

func RingoutListAnyResponse(aRes anyhttp.Response, apiClient *rc.APIClient, responseFormat string) {
	info, resp, err := apiClient.CallHandlingSettingsApi.ListExtensionForwardingNumbers(
		context.Background(), "~", "~", map[string]interface{}{})
	if err != nil {
		anyhttp.WriteSimpleJson(aRes, http.StatusInternalServerError, err.Error())
	} else {
		aRes.SetStatusCode(resp.StatusCode)
		if responseFormat == "json" {
			bytes, err := json.Marshal(info)
			if err != nil {
				anyhttp.WriteSimpleJson(aRes, http.StatusInternalServerError, err.Error())
			}
			aRes.SetContentType(hum.ContentTypeAppJsonUtf8)
			aRes.SetBodyBytes(bytes)
		} else {
			aRes.SetContentType(hum.ContentTypeTextPlainUsAscii)
			aRes.SetBodyBytes([]byte(ringoutListLegacyResponseBody(info.Records)))
		}
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

func RingoutCallAnyResponse(aRes anyhttp.Response, apiClient *rc.APIClient, ringOut ru.RingOutRequest, responseFormat string) {
	info, resp, err := apiClient.RingOutApi.MakeRingOutCallNew(
		context.Background(), "~", "~", *ringOut.Body())
	if err != nil {
		aRes.SetStatusCode(http.StatusInternalServerError)
		anyhttp.WriteSimpleJson(aRes, http.StatusInternalServerError, err.Error())
	} else {
		aRes.SetStatusCode(resp.StatusCode)
		if responseFormat == "json" {
			bytes, err := json.Marshal(info)
			if err != nil {
				anyhttp.WriteSimpleJson(aRes, http.StatusInternalServerError, err.Error())
			}
			aRes.SetContentType(hum.ContentTypeAppJsonUtf8)
			aRes.SetBodyBytes(bytes)
		} else {
			aRes.SetContentType(hum.ContentTypeTextPlainUsAscii)
			aRes.SetBodyBytes([]byte(fmt.Sprintf("OK %s", info.Id)))
		}
	}
}
