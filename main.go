package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/schema"
	cfg "github.com/grokify/gotilla/config"

	rc "github.com/grokify/go-ringcentral/client"
	ru "github.com/grokify/go-ringcentral/clientutil"
	ro "github.com/grokify/oauth2more/ringcentral"

	"github.com/grokify/ringcentral-legacy-api-proxy/handlers"
)

var decoder = schema.NewDecoder()

// Handler is a struct to hold the service handlers.
type Handler struct {
	AppPort        int
	APIClient      *rc.APIClient
	AppCredentials *ro.ApplicationCredentials
}

func (h *Handler) FaxOut(res http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	err := req.ParseMultipartForm(100000)
	if err != nil {
		panic(err)
	}
	form := req.MultipartForm
	formParser := handlers.NewLegacyMultipartFormParser(form)

	pwdCreds := formParser.PasswordCredentials()
	pwdCreds.RefreshTokenTTL = int64(-1)

	// Authorize
	apiClient, err := ru.NewApiClientPassword(*h.AppCredentials, pwdCreds)
	if err != nil {
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	restFaxReq := formParser.FaxRequest()

	url := ru.BuildFaxApiUrl(os.Getenv("RINGCENTRAL_SERVER_URL"))

	resp, err := restFaxReq.Post(apiClient.HTTPClient(), url)

	handlers.WriteFaxResponse(res, resp, err, formParser.Format())
}

// RingOut is a net/http handler for performing a RingOut API
// call using the RingCentral legacy ringout.asp API definition.
func (h *Handler) RingOut(res http.ResponseWriter, req *http.Request) {
	// Parse Request Data
	err := req.ParseForm()
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	var reqParams handlers.RingOutRequestParams
	err = decoder.Decode(&reqParams, req.Form)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		return
	} else if !reqParams.HasValidCommand() {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	// Authorize
	apiClient, err := ru.NewApiClientPassword(
		*h.AppCredentials,
		ro.PasswordCredentials{
			Username:        reqParams.Username,
			Extension:       reqParams.Ext,
			Password:        reqParams.Password,
			RefreshTokenTTL: int64(-1)})
	if err != nil {
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Process Request
	switch strings.ToLower(reqParams.Cmd) {
	case "call":
		ringOut := ru.RingOutRequest{
			To:         reqParams.To,
			From:       reqParams.From,
			CallerId:   reqParams.Clid,
			PlayPrompt: reqParams.PlayPrompt()}

		log.Printf("%v\n", ringOut)
		handlers.RingoutCall(res, apiClient, ringOut, reqParams.Format)
	case "list":
		handlers.RingoutList(res, apiClient, reqParams.Format)
	}
}

func serveNetHttp(handler Handler) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ringout.asp", http.HandlerFunc(handler.RingOut))
	mux.HandleFunc("/ringout.asp/", http.HandlerFunc(handler.RingOut))
	mux.HandleFunc("/faxout.asp", http.HandlerFunc(handler.FaxOut))
	mux.HandleFunc("/faxout.asp/", http.HandlerFunc(handler.FaxOut))

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
