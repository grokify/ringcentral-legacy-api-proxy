package ringcentrallegacyapiproxy

import (
	"bytes"
	"net/http"
	"net/url"
	//"encoding/json"
	"fmt"
	//"strings"

	//"github.com/grokify/glip-go-webhook"
	"github.com/grokify/gotilla/fmt/fmtutil"
	"github.com/grokify/gotilla/net/httputil"
	"github.com/valyala/fasthttp"
)

type RingOutHandler struct {
	Config Configuration
}

func NewRingOutHandler(config Configuration) (RingOutHandler, error) {
	h := RingOutHandler{Config: config}
	return h, nil
}

func (h *RingOutHandler) HandleFastHTTP(ctx *fasthttp.RequestCtx) {
	args := ListNumbersLegacyArgsFromRequest(ctx)
	if args.Cmd == "list" {
		h.ListNumbers(ctx)
	}
	fmtutil.PrintJSON(args)

	platform := h.Config.RcSdk.GetPlatform()
	platform.Authorize(args.Username, args.Ext, args.Password, false)
	reader := bytes.NewReader([]byte(""))
	resp, err := platform.Get("account/~/extension/~/forwarding-number", url.Values{}, reader, http.Header{})
	fmtutil.PrintJSON(resp)
	fmtutil.PrintJSON(err)
	if err == nil {
		fmt.Println("GOOD RESPONSE")
	} else {
		fmt.Printf("ERR [%v]\n", err)
	}
	if 1 == 0 {
		body, err := httputil.ResponseBody(resp)
		if err == nil {
			fmtutil.PrintJSON(body)
		}
	}
}

func (h *RingOutHandler) ListNumbers(ctx *fasthttp.RequestCtx) {

}

func ListNumbersLegacyArgsFromRequest(ctx *fasthttp.RequestCtx) ListNumbersLegacyArgs {
	return ListNumbersLegacyArgs{
		Cmd:      string(ctx.FormValue("cmd")),
		Username: string(ctx.FormValue("username")),
		Ext:      string(ctx.FormValue("ext")),
		Password: string(ctx.FormValue("password"))}
}

type ListNumbersLegacyArgs struct {
	Cmd      string
	Username string
	Ext      string
	Password string
}

type RingOutLegacyRequest struct {
}

type RingOutRequest struct {
	From       ContactInfo `json:"from,omitempty"`
	To         ContactInfo `json:"to,omitempty"`
	PlayPrompt bool        `json:"playPrompt,omitempty"`
}

type ContactInfo struct {
	PhoneNumber     string `json:"phoneNumber"`
	ExtensionNumber string `json:"extensionNumber"`
}
