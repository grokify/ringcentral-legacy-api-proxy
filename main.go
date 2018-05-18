package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	cfg "github.com/grokify/gotilla/config"
	hum "github.com/grokify/gotilla/net/httputilmore"
	nhu "github.com/grokify/gotilla/net/nethttputil"

	rc "github.com/grokify/go-ringcentral/client"
	ru "github.com/grokify/go-ringcentral/clientutil"
	ro "github.com/grokify/oauth2more/ringcentral"
)

// Handler is a struct to hold the service handlers.
type Handler struct {
	AppPort        int
	APIClient      *rc.APIClient
	AppCredentials *ro.ApplicationCredentials
}

// RingOut is a net/http handler for performing a RingOut API
// call using the RingCentral legacy ringout.asp API definition.
func (h *Handler) RingOut(res http.ResponseWriter, req *http.Request) {
	reqUtil := nhu.RequestUtil{Request: req}

	cmd := strings.ToLower(reqUtil.QueryParamString("cmd"))

	pwdCredentials := ro.PasswordCredentials{
		Username:        reqUtil.QueryParamString("username"),
		Extension:       reqUtil.QueryParamString("ext"),
		Password:        reqUtil.QueryParamString("password"),
		RefreshTokenTTL: int64(-1)}

	apiClient, err := ru.NewApiClientPassword(*h.AppCredentials, pwdCredentials)
	if err != nil {
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	switch cmd {
	case "call":
		prompt := reqUtil.QueryParamString("prompt")
		playPrompt := false
		if prompt == "1" {
			playPrompt = true
		}

		ringOut := ru.RingOutRequest{
			To:         reqUtil.QueryParamString("to"),
			From:       reqUtil.QueryParamString("from"),
			CallerId:   reqUtil.QueryParamString("clid"),
			PlayPrompt: playPrompt}

		log.Printf("%v\n", ringOut)
		ringoutCall(res, apiClient, ringOut)
	}
}

func ringoutCall(res http.ResponseWriter, apiClient *rc.APIClient, ringOut ru.RingOutRequest) {
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

	res.Header().Set(hum.HeaderContentType, hum.HeaderContentTypeValueJSONUTF8)
	res.Write(bytes)
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
