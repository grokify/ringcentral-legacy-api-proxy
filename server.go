package ringcentrallegacyapiproxy

import (
	"fmt"
	"log"
	"os"

	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"

	"github.com/grokify/ringcentral-sdk-go"
)

const (
	ROUTE_RINGOUT       = "/ringout.asp"
	ROUTE_RINGOUT_SLASH = "/ringout.asp/"
)

func StartServer(config Configuration) {
	router := fasthttprouter.New()

	router.GET("/", HomeHandler)

	ringOutHandler, err := NewRingOutHandler(config)
	if err != nil {
		panic("Bad RingOut Webhook Client Configuration")
	}
	router.GET(ROUTE_RINGOUT, ringOutHandler.HandleFastHTTP)
	router.GET(ROUTE_RINGOUT_SLASH, ringOutHandler.HandleFastHTTP)

	log.Fatal(fasthttp.ListenAndServe(config.Address(), router.Handler))
}

type Configuration struct {
	Port        int
	RcUsername  string
	RcExtension string
	RcPassword  string
	RcAppKey    string
	RcAppSecret string
	RcServerUrl string
	RcSdk       rcsdk.Sdk
}

func (config *Configuration) LoadSdk() {
	config.RcAppKey = os.Getenv("RC_APP_KEY")
	config.RcAppSecret = os.Getenv("RC_APP_SECRET")
	config.RcSdk = rcsdk.NewSdk(config.RcAppKey, config.RcAppSecret, rcsdk.RC_SERVER_SANDBOX)
}

func (config *Configuration) Address() string {
	return fmt.Sprintf(":%d", config.Port)
}
