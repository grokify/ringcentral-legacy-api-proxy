package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	cfg "github.com/grokify/gotilla/config"
	log "github.com/sirupsen/logrus"

	rc "github.com/grokify/go-ringcentral/client"
	ru "github.com/grokify/go-ringcentral/clientutil"
	ro "github.com/grokify/oauth2more/ringcentral"

	"github.com/apex/gateway"
	"github.com/buaazp/fasthttprouter"
	"github.com/grokify/gotilla/net/anyhttp"
	"github.com/grokify/ringcentral-legacy-api-proxy/handlers"
	"github.com/valyala/fasthttp"
)

// Handler is a struct to hold the service handlers.
type Handler struct {
	AppPort        int
	APIClient      *rc.APIClient
	AppCredentials *ro.ApplicationCredentials
}

func (h *Handler) FaxOutNetHttp(res http.ResponseWriter, req *http.Request) {
	log.Info("START_HANDLE_FAXOUT_NET_HTTP")
	h.handleAnyRequestFaxOut(anyhttp.NewResReqNetHttp(res, req))
}

func (h *Handler) FaxOutFastHttp(ctx *fasthttp.RequestCtx) {
	log.Info("START_HANDLE_FAXOUT_FAST_HTTP")
	h.handleAnyRequestFaxOut(anyhttp.NewResReqFastHttp(ctx))
}

func (h *Handler) RingOutNetHttp(res http.ResponseWriter, req *http.Request) {
	log.Info("START_HANDLE_RINGOUT_NET_HTTP")
	h.handleAnyRequestRingOut(anyhttp.NewResReqNetHttp(res, req))
}

func (h *Handler) RingOutFastHttp(ctx *fasthttp.RequestCtx) {
	log.Info("START_HANDLE_RINGOUT_FAST_HTTP")
	h.handleAnyRequestRingOut(anyhttp.NewResReqFastHttp(ctx))
}

func (h *Handler) handleAnyRequestFaxOut(aRes anyhttp.Response, aReq anyhttp.Request) {
	log.Info("START_HANDLE_FAXOUT_ANY_REQUEST")
	if strings.ToUpper(string(aReq.Method())) != http.MethodPost {
		anyhttp.WriteSimpleJson(aRes,
			http.StatusMethodNotAllowed,
			fmt.Sprintf("Method [%v] not allowed", string(aReq.Method())))
		return
	}

	form, err := aReq.MultipartForm()
	if err != nil {
		anyhttp.WriteSimpleJson(aRes, http.StatusBadRequest, err.Error())
		return
	}
	formParser := handlers.NewLegacyMultipartFormParser(form)

	pwdCreds := formParser.PasswordCredentials()
	pwdCreds.RefreshTokenTTL = int64(-1)

	// Authorize
	apiClient, err := ru.NewApiClientPassword(*h.AppCredentials, pwdCreds)
	if err != nil {
		anyhttp.WriteSimpleJson(aRes, http.StatusUnauthorized, err.Error())
		return
	}

	restFaxReq := formParser.FaxRequest()

	resp, err := restFaxReq.Post(
		apiClient.HTTPClient(),
		ru.BuildFaxApiUrl(os.Getenv("RINGCENTRAL_SERVER_URL")))

	handlers.WriteFaxAnyResponse(aRes, resp, err, formParser.Format())
}

// RingOut is a net/http handler for performing a RingOut API
// call using the RingCentral legacy ringout.asp API definition.
func (h *Handler) handleAnyRequestRingOut(aRes anyhttp.Response, aReq anyhttp.Request) {
	err := aReq.ParseForm()

	aReq.Method()
	if err != nil {
		anyhttp.WriteSimpleJson(aRes, http.StatusBadRequest, err.Error())
		return
	}
	reqParams := handlers.NewRingOutRequestParamsFromAnyArgs(aReq.AllArgs())
	if !reqParams.HasValidCommand() {
		anyhttp.WriteSimpleJson(aRes, http.StatusBadRequest, fmt.Sprintf("Invalid Command cmd[%v]", reqParams.Cmd))
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
		aRes.SetStatusCode(http.StatusUnauthorized)
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
		handlers.RingoutCallAnyResponse(aRes, apiClient, ringOut, reqParams.Format)
	case "list":
		handlers.RingoutListAnyResponse(aRes, apiClient, reqParams.Format)
	}
}

func serveAwsLambda(handler Handler) {
	log.Info("STARTING_AWS_LAMBDA")
	log.Fatal(gateway.ListenAndServe(fmt.Sprintf(":%v", handler.AppPort), getHttpServeMux(handler)))
}

func serveNetHttp(handler Handler) {
	log.Info("STARTING_NET_HTTP")
	done := make(chan bool)
	go http.ListenAndServe(fmt.Sprintf(":%v", handler.AppPort), getHttpServeMux(handler))
	log.Printf("Server listening on port %v", handler.AppPort)
	<-done
}

func getHttpServeMux(handler Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/ringout.asp", http.HandlerFunc(handler.RingOutNetHttp))
	mux.HandleFunc("/ringout.asp/", http.HandlerFunc(handler.RingOutNetHttp))
	mux.HandleFunc("/faxout.asp", http.HandlerFunc(handler.FaxOutNetHttp))
	mux.HandleFunc("/faxout.asp/", http.HandlerFunc(handler.FaxOutNetHttp))
	return mux
}

func serveFastHttp(handler Handler) {
	log.Info("STARTING_FAST_HTTP")
	router := fasthttprouter.New()
	router.POST("/faxout.asp", handler.FaxOutFastHttp)
	router.POST("/faxout.asp/", handler.FaxOutFastHttp)
	router.POST("/ringout.asp", handler.RingOutFastHttp)
	router.POST("/ringout.asp/", handler.RingOutFastHttp)
	router.GET("/ringout.asp", handler.RingOutFastHttp)
	router.GET("/ringout.asp/", handler.RingOutFastHttp)

	done := make(chan bool)
	go fasthttp.ListenAndServe(fmt.Sprintf(":%v", handler.AppPort), router.Handler)
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
		port = 3000
	}
	port = 3000

	handler := Handler{
		AppPort: port,
		AppCredentials: &ro.ApplicationCredentials{
			ServerURL:    os.Getenv("RINGCENTRAL_SERVER_URL"),
			ClientID:     os.Getenv("RINGCENTRAL_CLIENT_ID"),
			ClientSecret: os.Getenv("RINGCENTRAL_CLIENT_SECRET")}}

	engine := strings.ToLower(strings.TrimSpace(os.Getenv("HTTP_ENGINE")))
	if len(engine) == 0 {
		engine = "nethttp"
	}

	switch engine {
	case "awslambda":
		serveAwsLambda(handler)
	case "fasthttp":
		serveFastHttp(handler)
	default:
		serveNetHttp(handler)
	}
}
