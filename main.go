package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/gorilla/schema"
	cfg "github.com/grokify/gotilla/config"
	hum "github.com/grokify/gotilla/net/httputilmore"

	rc "github.com/grokify/go-ringcentral/client"
	ru "github.com/grokify/go-ringcentral/clientutil"
	ro "github.com/grokify/oauth2more/ringcentral"
)

var decoder = schema.NewDecoder()

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

// Handler is a struct to hold the service handlers.
type Handler struct {
	AppPort        int
	APIClient      *rc.APIClient
	AppCredentials *ro.ApplicationCredentials
}

// RingOut is a net/http handler for performing a RingOut API
// call using the RingCentral legacy ringout.asp API definition.
func (h *Handler) RingOut(res http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	var reqParams RingOutRequestParams
	err = decoder.Decode(&reqParams, req.Form)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	} else if !reqParams.HasValidCommand() {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	pwdCredentials := ro.PasswordCredentials{
		Username:        reqParams.Username,
		Extension:       reqParams.Ext,
		Password:        reqParams.Password,
		RefreshTokenTTL: int64(-1)}

	apiClient, err := ru.NewApiClientPassword(*h.AppCredentials, pwdCredentials)
	if err != nil {
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	switch strings.ToLower(reqParams.Cmd) {
	case "call":
		ringOut := ru.RingOutRequest{
			To:         reqParams.To,
			From:       reqParams.From,
			CallerId:   reqParams.Clid,
			PlayPrompt: reqParams.PlayPrompt()}

		log.Printf("%v\n", ringOut)
		ringoutCall(res, apiClient, ringOut, reqParams.Format)
	case "list":
		ringoutList(res, apiClient, reqParams.Format)
	}
}

type ErrorResponse struct {
	StatusCode int
	Message    string
}

func (eresp *ErrorResponse) ToJSON() []byte {
	bytes, err := json.Marshal(eresp)
	if err != nil {
		eresp2 := ErrorResponse{StatusCode: 500, Message: err.Error()}
		bytes, _ := json.Marshal(eresp2)
		return bytes
	}
	return bytes
}

func ringoutList(res http.ResponseWriter, apiClient *rc.APIClient, format string) {
	fmt.Println("LIST")
	info, resp, err := apiClient.CallHandlingSettingsApi.ListExtensionForwardingNumbers(
		context.Background(), "~", "~", map[string]interface{}{})
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		eresp := ErrorResponse{StatusCode: 500, Message: err.Error()}
		res.Write(eresp.ToJSON())
		return
	} else if resp.StatusCode >= 500 {
		res.WriteHeader(http.StatusInternalServerError)
		eresp := ErrorResponse{StatusCode: resp.StatusCode, Message: fmt.Sprintf("REST API Response: %v", resp.StatusCode)}
		res.Write(eresp.ToJSON())
		return
	} else if resp.StatusCode >= 400 {
		res.WriteHeader(http.StatusBadRequest)
		eresp := ErrorResponse{StatusCode: resp.StatusCode, Message: fmt.Sprintf("REST API Response: %v", resp.StatusCode)}
		res.Write(eresp.ToJSON())
		return
	} else if resp.StatusCode >= 300 {
		res.WriteHeader(http.StatusInternalServerError)
		eresp := ErrorResponse{StatusCode: resp.StatusCode, Message: fmt.Sprintf("REST API Response: %v", resp.StatusCode)}
		res.Write(eresp.ToJSON())
		return
	}
	bytes, err := json.Marshal(info)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	if format == "json" {
		res.Header().Set(hum.HeaderContentType, hum.HeaderContentTypeValueJSONUTF8)
		res.Write(bytes)
	} else {
		res.Header().Set(hum.HeaderContentType, "text/plain; charset=us-ascii")
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

func ringoutCall(res http.ResponseWriter, apiClient *rc.APIClient, ringOut ru.RingOutRequest, format string) {
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

	if format == "json" {
		res.Header().Set(hum.HeaderContentType, hum.HeaderContentTypeValueJSONUTF8)
		res.Write(bytes)
	} else {
		res.Header().Set(hum.HeaderContentType, "text/plain; charset=us-ascii")
		res.Write([]byte(fmt.Sprintf("OK %s", info.Id)))
	}
}

func serveNetHttp(handler Handler) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ringout.asp", http.HandlerFunc(handler.RingOut))
	mux.HandleFunc("/ringout.asp/", http.HandlerFunc(handler.RingOut))

	done := make(chan bool)
	go http.ListenAndServe(fmt.Sprintf(":%v", handler.AppPort), mux)
	log.Printf("Server listening on port %v", handler.AppPort)
	<-done
}

func main() {
	err := cfg.LoadDotEnvSkipEmpty(os.Getenv("ENV_PATH"), "./.env")
	if err != nil {
		panic(err)
	}

	// PORT environment variable is automatically set for Heroku.
	portRaw := os.Getenv("PORT")
	port, err := strconv.Atoi(portRaw)
	if err != nil {
		port = 8080
	}

	handler := Handler{
		AppPort: port,
		AppCredentials: &ro.ApplicationCredentials{
			ServerURL:    os.Getenv("RINGCENTRAL_SERVER_URL"),
			ClientID:     os.Getenv("RINGCENTRAL_CLIENT_ID"),
			ClientSecret: os.Getenv("RINGCENTRAL_CLIENT_SECRET")}}

	serveNetHttp(handler)
}
