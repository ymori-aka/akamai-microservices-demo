// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	pb "github.com/GoogleCloudPlatform/microservices-demo/src/frontend/genproto"
	"github.com/GoogleCloudPlatform/microservices-demo/src/frontend/money"
	"github.com/GoogleCloudPlatform/microservices-demo/src/frontend/validator"
)

type platformDetails struct {
	css      string
	provider string
}

var (
	frontendMessage  = strings.TrimSpace(os.Getenv("FRONTEND_MESSAGE"))
	isCymbalBrand    = "true" == strings.ToLower(os.Getenv("CYMBAL_BRANDING"))
	assistantEnabled = "true" == strings.ToLower(os.Getenv("ENABLE_ASSISTANT"))
	templates        = template.Must(template.New("").
				Funcs(template.FuncMap{
			"renderMoney":        renderMoney,
			"renderCurrencyLogo": renderCurrencyLogo,
			"jaName": func(id string) string {
				if t, ok := jaTranslations[id]; ok {
					return t.Name
				}
				return ""
			},
			"jaDesc": func(id string) string {
				if t, ok := jaTranslations[id]; ok {
					return t.Description
				}
				return ""
			},
		}).ParseGlob("templates/*.html"))
	plat platformDetails
)

var validEnvs = []string{"local", "gcp", "azure", "aws", "onprem", "alibaba", "akamai"}

func (fe *frontendServer) homeHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
	log.WithField("currency", currentCurrency(r)).Info("home")
	currencies, err := fe.getCurrencies(r.Context())
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "could not retrieve currencies"), http.StatusInternalServerError)
		return
	}
	products, err := fe.getProducts(r.Context())
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "could not retrieve products"), http.StatusInternalServerError)
		return
	}
	cart, err := fe.getCart(r.Context(), sessionID(r))
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "could not retrieve cart"), http.StatusInternalServerError)
		return
	}

	type productView struct {
		Item  *pb.Product
		Price *pb.Money
	}
	ps := make([]productView, len(products))
	for i, p := range products {
		price, err := fe.convertCurrency(r.Context(), p.GetPriceUsd(), currentCurrency(r))
		if err != nil {
			renderHTTPError(log, r, w, errors.Wrapf(err, "failed to do currency conversion for product %s", p.GetId()), http.StatusInternalServerError)
			return
		}
		ps[i] = productView{p, price}
	}

	// Prepend admin-only products (not in gRPC catalog) to the top of the listing
	loadAdminData()
	existingIDs := make(map[string]bool)
	for _, p := range products {
		existingIDs[p.GetId()] = true
	}
	adminMu.RLock()
	var adminPs []productView
	for _, ap := range adminProducts {
		if existingIDs[ap.ID] || ap.Hidden {
			continue
		}
		pbP := adminToPbProduct(ap)
		price, err := fe.convertCurrency(r.Context(), pbP.GetPriceUsd(), currentCurrency(r))
		if err != nil {
			price = pbP.GetPriceUsd()
		}
		adminPs = append(adminPs, productView{pbP, price})
	}
	adminMu.RUnlock()
	// Sort by ID descending so newest products appear first
	sort.Slice(adminPs, func(i, j int) bool {
		return adminPs[i].Item.Id > adminPs[j].Item.Id
	})
	ps = append(adminPs, ps...)

	// Set ENV_PLATFORM (default to local if not set; use env var if set; otherwise detect GCP, which overrides env)_
	var env = os.Getenv("ENV_PLATFORM")
	// Only override from env variable if set + valid env
	if env == "" || stringinSlice(validEnvs, env) == false {
		fmt.Println("env platform is either empty or invalid")
		env = "local"
	}
	// Autodetect GCP
	addrs, err := net.LookupHost("metadata.google.internal.")
	if err == nil && len(addrs) >= 0 {
		log.Debugf("Detected Google metadata server: %v, setting ENV_PLATFORM to GCP.", addrs)
		env = "gcp"
	}

	log.Debugf("ENV_PLATFORM is: %s", env)
	plat = platformDetails{}
	plat.setPlatformDetails(strings.ToLower(env))

	if err := templates.ExecuteTemplate(w, "home", injectCommonTemplateData(r, map[string]interface{}{
		"show_currency": true,
		"currencies":    currencies,
		"products":      ps,
		"cart_size":     cartSize(cart),
		"banner_color":  os.Getenv("BANNER_COLOR"), // illustrates canary deployments
		"ad":            fe.chooseAd(r.Context(), []string{}, log),
	})); err != nil {
		log.Error(err)
	}
}

func (plat *platformDetails) setPlatformDetails(env string) {
	if env == "aws" {
		plat.provider = "AWS"
		plat.css = "aws-platform"
	} else if env == "onprem" {
		plat.provider = "On-Premises"
		plat.css = "onprem-platform"
	} else if env == "azure" {
		plat.provider = "Azure"
		plat.css = "azure-platform"
	} else if env == "gcp" {
		plat.provider = "Google Cloud"
		plat.css = "gcp-platform"
	} else if env == "alibaba" {
		plat.provider = "Alibaba Cloud"
		plat.css = "alibaba-platform"
	} else if env == "akamai" {
		plat.provider = "Akamai Cloud"
		plat.css = "akamai-platform"
	} else {
		plat.provider = "local"
		plat.css = "local"
	}
}

func (fe *frontendServer) productHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
	id := mux.Vars(r)["id"]
	if id == "" {
		renderHTTPError(log, r, w, errors.New("product id not specified"), http.StatusBadRequest)
		return
	}
	log.WithField("id", id).WithField("currency", currentCurrency(r)).
		Debug("serving product page")

	p, err := fe.getProductWithAdminFallback(r.Context(), id)
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "could not retrieve product"), http.StatusInternalServerError)
		return
	}
	currencies, err := fe.getCurrencies(r.Context())
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "could not retrieve currencies"), http.StatusInternalServerError)
		return
	}

	cart, err := fe.getCart(r.Context(), sessionID(r))
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "could not retrieve cart"), http.StatusInternalServerError)
		return
	}

	price, err := fe.convertCurrency(r.Context(), p.GetPriceUsd(), currentCurrency(r))
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "failed to convert currency"), http.StatusInternalServerError)
		return
	}

	// ignores the error retrieving recommendations since it is not critical
	recommendations, err := fe.getRecommendations(r.Context(), sessionID(r), []string{id})
	if err != nil {
		log.WithField("error", err).Warn("failed to get product recommendations")
	}

	product := struct {
		Item  *pb.Product
		Price *pb.Money
	}{p, price}

	// Fetch packaging info (weight/dimensions) of the product
	// The packaging service is an optional microservice you can run as part of a Google Cloud demo.
	var packagingInfo *PackagingInfo = nil
	if isPackagingServiceConfigured() {
		packagingInfo, err = httpGetPackagingInfo(id)
		if err != nil {
			fmt.Println("Failed to obtain product's packaging info:", err)
		}
	}

	if err := templates.ExecuteTemplate(w, "product", injectCommonTemplateData(r, map[string]interface{}{
		"ad":              fe.chooseAd(r.Context(), p.Categories, log),
		"show_currency":   true,
		"currencies":      currencies,
		"product":         product,
		"recommendations": recommendations,
		"cart_size":       cartSize(cart),
		"packagingInfo":   packagingInfo,
	})); err != nil {
		log.Println(err)
	}
}

func (fe *frontendServer) addToCartHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
	quantity, _ := strconv.ParseUint(r.FormValue("quantity"), 10, 32)
	productID := r.FormValue("product_id")
	payload := validator.AddToCartPayload{
		Quantity:  quantity,
		ProductID: productID,
	}
	if err := payload.Validate(); err != nil {
		renderHTTPError(log, r, w, validator.ValidationErrorResponse(err), http.StatusUnprocessableEntity)
		return
	}
	log.WithField("product", payload.ProductID).WithField("quantity", payload.Quantity).Debug("adding to cart")

	p, err := fe.getProductWithAdminFallback(r.Context(), payload.ProductID)
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "could not retrieve product"), http.StatusInternalServerError)
		return
	}

	if err := fe.insertCart(r.Context(), sessionID(r), p.GetId(), int32(payload.Quantity)); err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "failed to add to cart"), http.StatusInternalServerError)
		return
	}
	w.Header().Set("location", baseUrl + "/cart")
	w.WriteHeader(http.StatusFound)
}

func (fe *frontendServer) emptyCartHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
	log.Debug("emptying cart")

	if err := fe.emptyCart(r.Context(), sessionID(r)); err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "failed to empty cart"), http.StatusInternalServerError)
		return
	}
	w.Header().Set("location", baseUrl + "/")
	w.WriteHeader(http.StatusFound)
}

func (fe *frontendServer) viewCartHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
	log.Debug("view user cart")
	currencies, err := fe.getCurrencies(r.Context())
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "could not retrieve currencies"), http.StatusInternalServerError)
		return
	}
	cart, err := fe.getCart(r.Context(), sessionID(r))
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "could not retrieve cart"), http.StatusInternalServerError)
		return
	}

	// ignores the error retrieving recommendations since it is not critical
	recommendations, err := fe.getRecommendations(r.Context(), sessionID(r), cartIDs(cart))
	if err != nil {
		log.WithField("error", err).Warn("failed to get product recommendations")
	}

	shippingCost, err := fe.getShippingQuote(r.Context(), cart, currentCurrency(r))
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "failed to get shipping quote"), http.StatusInternalServerError)
		return
	}

	type cartItemView struct {
		Item     *pb.Product
		Quantity int32
		Price    *pb.Money
	}
	items := make([]cartItemView, len(cart))
	totalPrice := pb.Money{CurrencyCode: currentCurrency(r)}
	for i, item := range cart {
		p, err := fe.getProductWithAdminFallback(r.Context(), item.GetProductId())
		if err != nil {
			renderHTTPError(log, r, w, errors.Wrapf(err, "could not retrieve product #%s", item.GetProductId()), http.StatusInternalServerError)
			return
		}
		price, err := fe.convertCurrency(r.Context(), p.GetPriceUsd(), currentCurrency(r))
		if err != nil {
			renderHTTPError(log, r, w, errors.Wrapf(err, "could not convert currency for product #%s", item.GetProductId()), http.StatusInternalServerError)
			return
		}

		multPrice := money.MultiplySlow(*price, uint32(item.GetQuantity()))
		items[i] = cartItemView{
			Item:     p,
			Quantity: item.GetQuantity(),
			Price:    &multPrice}
		totalPrice = money.Must(money.Sum(totalPrice, multPrice))
	}
	totalPrice = money.Must(money.Sum(totalPrice, *shippingCost))
	year := time.Now().Year()

	if err := templates.ExecuteTemplate(w, "cart", injectCommonTemplateData(r, map[string]interface{}{
		"currencies":       currencies,
		"recommendations":  recommendations,
		"cart_size":        cartSize(cart),
		"shipping_cost":    shippingCost,
		"show_currency":    true,
		"total_cost":       totalPrice,
		"items":            items,
		"expiration_years": []int{year, year + 1, year + 2, year + 3, year + 4},
	})); err != nil {
		log.Println(err)
	}
}

func (fe *frontendServer) placeOrderHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
	log.Debug("placing order")

	var (
		email         = r.FormValue("email")
		streetAddress = r.FormValue("street_address")
		zipCode, _    = strconv.ParseInt(r.FormValue("zip_code"), 10, 32)
		city          = r.FormValue("city")
		state         = r.FormValue("state")
		country       = r.FormValue("country")
		ccNumber      = r.FormValue("credit_card_number")
		ccMonth, _    = strconv.ParseInt(r.FormValue("credit_card_expiration_month"), 10, 32)
		ccYear, _     = strconv.ParseInt(r.FormValue("credit_card_expiration_year"), 10, 32)
		ccCVV, _      = strconv.ParseInt(r.FormValue("credit_card_cvv"), 10, 32)
	)

	payload := validator.PlaceOrderPayload{
		Email:         email,
		StreetAddress: streetAddress,
		ZipCode:       zipCode,
		City:          city,
		State:         state,
		Country:       country,
		CcNumber:      ccNumber,
		CcMonth:       ccMonth,
		CcYear:        ccYear,
		CcCVV:         ccCVV,
	}
	if err := payload.Validate(); err != nil {
		renderHTTPError(log, r, w, validator.ValidationErrorResponse(err), http.StatusUnprocessableEntity)
		return
	}

	order, err := pb.NewCheckoutServiceClient(fe.checkoutSvcConn).
		PlaceOrder(r.Context(), &pb.PlaceOrderRequest{
			Email: payload.Email,
			CreditCard: &pb.CreditCardInfo{
				CreditCardNumber:          payload.CcNumber,
				CreditCardExpirationMonth: int32(payload.CcMonth),
				CreditCardExpirationYear:  int32(payload.CcYear),
				CreditCardCvv:             int32(payload.CcCVV)},
			UserId:       sessionID(r),
			UserCurrency: currentCurrency(r),
			Address: &pb.Address{
				StreetAddress: payload.StreetAddress,
				City:          payload.City,
				State:         payload.State,
				ZipCode:       int32(payload.ZipCode),
				Country:       payload.Country},
		})
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "failed to complete the order"), http.StatusInternalServerError)
		return
	}
	log.WithField("order", order.GetOrder().GetOrderId()).Info("order placed")

	order.GetOrder().GetItems()
	recommendations, _ := fe.getRecommendations(r.Context(), sessionID(r), nil)

	totalPaid := *order.GetOrder().GetShippingCost()
	for _, v := range order.GetOrder().GetItems() {
		multPrice := money.MultiplySlow(*v.GetCost(), uint32(v.GetItem().GetQuantity()))
		totalPaid = money.Must(money.Sum(totalPaid, multPrice))
	}

	currencies, err := fe.getCurrencies(r.Context())
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "could not retrieve currencies"), http.StatusInternalServerError)
		return
	}

	if err := templates.ExecuteTemplate(w, "order", injectCommonTemplateData(r, map[string]interface{}{
		"show_currency":   false,
		"currencies":      currencies,
		"order":           order.GetOrder(),
		"total_paid":      &totalPaid,
		"recommendations": recommendations,
	})); err != nil {
		log.Println(err)
	}
}

func (fe *frontendServer) assistantHandler(w http.ResponseWriter, r *http.Request) {
	currencies, err := fe.getCurrencies(r.Context())
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "could not retrieve currencies"), http.StatusInternalServerError)
		return
	}

	if err := templates.ExecuteTemplate(w, "assistant", injectCommonTemplateData(r, map[string]interface{}{
		"show_currency": false,
		"currencies":    currencies,
	})); err != nil {
		log.Println(err)
	}
}

func (fe *frontendServer) logoutHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
	log.Debug("logging out")
	for _, c := range r.Cookies() {
		c.Expires = time.Now().Add(-time.Hour * 24 * 365)
		c.MaxAge = -1
		http.SetCookie(w, c)
	}
	w.Header().Set("Location", baseUrl + "/")
	w.WriteHeader(http.StatusFound)
}

func (fe *frontendServer) getProductByID(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["ids"]
	if id == "" {
		return
	}

	p, err := fe.getProductWithAdminFallback(r.Context(), id)
	if err != nil {
		http.Error(w, "product not found", http.StatusNotFound)
		return
	}

	// Return a JSON-friendly struct including Japanese translations
	type priceJSON struct {
		CurrencyCode string  `json:"currency_code"`
		Units        int64   `json:"units"`
		Nanos        int32   `json:"nanos"`
	}
	type productJSON struct {
		ID            string    `json:"id"`
		Name          string    `json:"name"`
		NameJa        string    `json:"name_ja,omitempty"`
		Description   string    `json:"description"`
		DescriptionJa string    `json:"description_ja,omitempty"`
		Picture       string    `json:"picture"`
		PriceUSD      priceJSON `json:"price_usd"`
	}

	nameJa := ""
	descJa := ""
	if t, ok := jaTranslations[id]; ok {
		nameJa = t.Name
		descJa = t.Description
	}

	out := productJSON{
		ID:            p.GetId(),
		Name:          p.GetName(),
		NameJa:        nameJa,
		Description:   p.GetDescription(),
		DescriptionJa: descJa,
		Picture:       p.GetPicture(),
		PriceUSD: priceJSON{
			CurrencyCode: p.GetPriceUsd().GetCurrencyCode(),
			Units:        p.GetPriceUsd().GetUnits(),
			Nanos:        p.GetPriceUsd().GetNanos(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

// chatBotHandler handles the shopping assistant chat.
// It fetches the product catalog, builds an OpenAI-compatible prompt,
// calls the Gemma 4 LLM directly, and returns the response.
func (fe *frontendServer) chatBotHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)

	// --- parse request ---
	type IncomingMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type IncomingReq struct {
		Message string        `json:"message"`
		History []IncomingMsg `json:"history"`
		Lang    string        `json:"lang"`
	}
	var req IncomingReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "failed to decode request"), http.StatusBadRequest)
		return
	}
	if req.Lang == "" {
		req.Lang = "en"
	}

	// --- build system prompt with product catalog ---
	products, err := fe.getProducts(r.Context())
	if err != nil {
		log.WithError(err).Warn("chatbot: failed to fetch product catalog, using empty catalog")
	}
	var catalogLines strings.Builder
	for _, p := range products {
		price := float64(p.GetPriceUsd().GetUnits()) + float64(p.GetPriceUsd().GetNanos())/1e9
		catalogLines.WriteString(fmt.Sprintf(
			"- [%s] %s — $%.2f — %s\n",
			p.GetId(), p.GetName(), price, p.GetDescription(),
		))
	}

	var systemPrompt string
	if req.Lang == "ja" {
		systemPrompt = fmt.Sprintf(`あなたはAkamai Storeのショッピングアシスタントです。
お客様の質問に日本語で答えてください。商品の推薦を行う際は、必ず以下のカタログから選び、
商品IDを [AKMT001] のような形式でメッセージ内に含めてください（最大3件）。
カタログにない商品は絶対に作らないでください。

【商品カタログ】
%s`, catalogLines.String())
	} else {
		systemPrompt = fmt.Sprintf(`You are a helpful shopping assistant for the Akamai Store.
Answer the customer's questions in English. When recommending products, choose from the catalog below
and include the product ID in [AKMT001] format in your message (up to 3 items).
Never invent products that are not in the catalog.

[Product Catalog]
%s`, catalogLines.String())
	}

	// --- build OpenAI messages array ---
	type LLMMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	messages := []LLMMessage{{Role: "system", Content: systemPrompt}}
	for _, h := range req.History {
		if h.Role == "user" || h.Role == "assistant" {
			messages = append(messages, LLMMessage{Role: h.Role, Content: h.Content})
		}
	}
	messages = append(messages, LLMMessage{Role: "user", Content: req.Message})

	// --- call Spin shopping-assistant-service (which proxies to GPU) ---
	// LKE pods cannot reach the GPU server directly; route via Akamai Functions.
	assistantURL := os.Getenv("SHOPPING_ASSISTANT_SERVICE_ADDR")
	if assistantURL == "" {
		renderHTTPError(log, r, w, errors.New("SHOPPING_ASSISTANT_SERVICE_ADDR is not configured"), http.StatusServiceUnavailable)
		return
	}
	// Trim trailing slash for safety
	assistantURL = strings.TrimRight(assistantURL, "/")

	type SpinRequest struct {
		Messages    []LLMMessage `json:"messages"`
		MaxTokens   int          `json:"max_tokens"`
		Temperature float64      `json:"temperature"`
	}
	spinReqBody, _ := json.Marshal(SpinRequest{
		Messages:    messages,
		MaxTokens:   512,
		Temperature: 0.7,
	})

	spinReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost,
		assistantURL+"/chat", strings.NewReader(string(spinReqBody)))
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "failed to create assistant request"), http.StatusInternalServerError)
		return
	}
	spinReq.Header.Set("Content-Type", "application/json")

	spinResp, err := http.DefaultClient.Do(spinReq)
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "failed to call assistant service"), http.StatusInternalServerError)
		return
	}
	defer spinResp.Body.Close()

	var spinResult struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(spinResp.Body).Decode(&spinResult); err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "failed to decode assistant response"), http.StatusInternalServerError)
		return
	}

	reply := spinResult.Message

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": reply})
}

func (fe *frontendServer) setCurrencyHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)
	cur := r.FormValue("currency_code")
	payload := validator.SetCurrencyPayload{Currency: cur}
	if err := payload.Validate(); err != nil {
		renderHTTPError(log, r, w, validator.ValidationErrorResponse(err), http.StatusUnprocessableEntity)
		return
	}
	log.WithField("curr.new", payload.Currency).WithField("curr.old", currentCurrency(r)).
		Debug("setting currency")

	if payload.Currency != "" {
		http.SetCookie(w, &http.Cookie{
			Name:   cookieCurrency,
			Value:  payload.Currency,
			MaxAge: cookieMaxAge,
		})
	}
	referer := r.Header.Get("referer")
	if referer == "" {
		referer = baseUrl + "/"
	}
	w.Header().Set("Location", referer)
	w.WriteHeader(http.StatusFound)
}

// chooseAd queries for advertisements available and randomly chooses one, if
// available. It ignores the error retrieving the ad since it is not critical.
func (fe *frontendServer) chooseAd(ctx context.Context, ctxKeys []string, log logrus.FieldLogger) *pb.Ad {
	ads, err := fe.getAd(ctx, ctxKeys)
	if err != nil {
		log.WithField("error", err).Warn("failed to retrieve ads")
		return nil
	}
	return ads[rand.Intn(len(ads))]
}

func renderHTTPError(log logrus.FieldLogger, r *http.Request, w http.ResponseWriter, err error, code int) {
	log.WithField("error", err).Error("request error")
	errMsg := fmt.Sprintf("%+v", err)

	w.WriteHeader(code)

	if templateErr := templates.ExecuteTemplate(w, "error", injectCommonTemplateData(r, map[string]interface{}{
		"error":       errMsg,
		"status_code": code,
		"status":      http.StatusText(code),
	})); templateErr != nil {
		log.Println(templateErr)
	}
}

func injectCommonTemplateData(r *http.Request, payload map[string]interface{}) map[string]interface{} {
	data := map[string]interface{}{
		"session_id":        sessionID(r),
		"request_id":        r.Context().Value(ctxKeyRequestID{}),
		"user_currency":     currentCurrency(r),
		"user_lang":         currentLang(r),
		"ja_translations":   jaTranslations,
		"platform_css":      plat.css,
		"platform_name":     plat.provider,
		"is_cymbal_brand":   isCymbalBrand,
		"assistant_enabled": assistantEnabled,
		"deploymentDetails": deploymentDetailsMap,
		"frontendMessage":   frontendMessage,
		"currentYear":       time.Now().Year(),
		"baseUrl":           baseUrl,
	}

	for k, v := range payload {
		data[k] = v
	}

	return data
}

func currentCurrency(r *http.Request) string {
	c, _ := r.Cookie(cookieCurrency)
	if c != nil {
		return c.Value
	}
	return defaultCurrency
}

func currentLang(r *http.Request) string {
	c, _ := r.Cookie(cookieLang)
	if c != nil && (c.Value == "ja" || c.Value == "en") {
		return c.Value
	}
	return "en"
}

func (fe *frontendServer) setLangHandler(w http.ResponseWriter, r *http.Request) {
	lang := r.FormValue("lang")
	if lang == "ja" || lang == "en" {
		http.SetCookie(w, &http.Cookie{
			Name:   cookieLang,
			Value:  lang,
			MaxAge: cookieMaxAge,
		})
	}
	referer := r.Header.Get("referer")
	if referer == "" {
		referer = baseUrl + "/"
	}
	w.Header().Set("Location", referer)
	w.WriteHeader(http.StatusFound)
}

func sessionID(r *http.Request) string {
	v := r.Context().Value(ctxKeySessionID{})
	if v != nil {
		return v.(string)
	}
	return ""
}

func cartIDs(c []*pb.CartItem) []string {
	out := make([]string, len(c))
	for i, v := range c {
		out[i] = v.GetProductId()
	}
	return out
}

// get total # of items in cart
func cartSize(c []*pb.CartItem) int {
	cartSize := 0
	for _, item := range c {
		cartSize += int(item.GetQuantity())
	}
	return cartSize
}

func renderMoney(money pb.Money) string {
	currencyLogo := renderCurrencyLogo(money.GetCurrencyCode())
	return fmt.Sprintf("%s%d.%02d", currencyLogo, money.GetUnits(), money.GetNanos()/10000000)
}

func renderCurrencyLogo(currencyCode string) string {
	logos := map[string]string{
		"USD": "$",
		"CAD": "$",
		"JPY": "¥",
		"EUR": "€",
		"TRY": "₺",
		"GBP": "£",
	}

	logo := "$" //default
	if val, ok := logos[currencyCode]; ok {
		logo = val
	}
	return logo
}

func stringinSlice(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// ---- Inventory / Product Management ----

// AdminProduct holds admin-editable product data (overlays the gRPC catalog)
type AdminProduct struct {
	ID          string  `json:"id"`
	NameEN      string  `json:"name_en"`
	NameJA      string  `json:"name_ja"`
	Description string  `json:"description"`
	DescJA      string  `json:"desc_ja"`
	Price       float64 `json:"price"` // USD
	Picture     string  `json:"picture"`
	Categories  string  `json:"categories"` // comma-separated
	Stock       int     `json:"stock"`
	Hidden      bool    `json:"hidden"`
}

var (
	adminMu       sync.RWMutex
	adminProducts = map[string]*AdminProduct{} // productID -> overrides
)

const adminDataFile = "/inventory/admin_products.json"

func loadAdminData() {
	adminMu.Lock()
	defer adminMu.Unlock()
	data, err := os.ReadFile(adminDataFile)
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, &adminProducts)
	// Sync JA translations for admin-only products
	for id, ap := range adminProducts {
		if ap.NameJA != "" || ap.DescJA != "" {
			jaTranslations[id] = jaProduct{Name: ap.NameJA, Description: ap.DescJA}
		}
	}
}

// getProductWithAdminFallback tries the gRPC catalog first, then falls back to admin-only products.
// Use this everywhere a product ID might be admin-only (not in the gRPC catalog).
func (fe *frontendServer) getProductWithAdminFallback(ctx context.Context, id string) (*pb.Product, error) {
	p, err := fe.getProduct(ctx, id)
	if err != nil {
		loadAdminData()
		adminMu.RLock()
		ap, ok := adminProducts[id]
		adminMu.RUnlock()
		if !ok {
			return nil, err
		}
		return adminToPbProduct(ap), nil
	}
	return p, nil
}

// adminToPbProduct converts an AdminProduct to a pb.Product for rendering.
func adminToPbProduct(ap *AdminProduct) *pb.Product {
	units := int64(ap.Price)
	nanos := int32((ap.Price - float64(units)) * 1e9)
	var cats []string
	for _, c := range strings.Split(ap.Categories, ",") {
		if t := strings.TrimSpace(c); t != "" {
			cats = append(cats, t)
		}
	}
	return &pb.Product{
		Id:          ap.ID,
		Name:        ap.NameEN,
		Description: ap.Description,
		Picture:     ap.Picture,
		PriceUsd:    &pb.Money{CurrencyCode: "USD", Units: units, Nanos: nanos},
		Categories:  cats,
	}
}

func saveAdminData() error {
	adminMu.RLock()
	data, err := json.MarshalIndent(adminProducts, "", "  ")
	adminMu.RUnlock()
	if err != nil {
		return err
	}
	_ = os.MkdirAll("/inventory", 0755)
	return os.WriteFile(adminDataFile, data, 0644)
}

func getAdminProduct(p *pb.Product) *AdminProduct {
	if ap, ok := adminProducts[p.GetId()]; ok {
		return ap
	}
	// Create from gRPC product defaults
	jaT := jaTranslations[p.GetId()]
	price := float64(p.GetPriceUsd().GetUnits()) + float64(p.GetPriceUsd().GetNanos())/1e9
	return &AdminProduct{
		ID:          p.GetId(),
		NameEN:      p.GetName(),
		NameJA:      jaT.Name,
		Description: p.GetDescription(),
		DescJA:      jaT.Description,
		Price:       price,
		Picture:     p.GetPicture(),
		Categories:  strings.Join(p.GetCategories(), ","),
		Stock:       100,
		Hidden:      false,
	}
}

// adminBasicAuth wraps a handler with HTTP Basic Auth.
// Credentials are read from env vars ADMIN_USER / ADMIN_PASSWORD (defaults: admin / akamai-demo).
func adminBasicAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := os.Getenv("ADMIN_USER")
		pass := os.Getenv("ADMIN_PASSWORD")
		if user == "" {
			user = "admin"
		}
		if pass == "" {
			pass = "akamai-demo"
		}
		u, p, ok := r.BasicAuth()
		if !ok || u != user || p != pass {
			w.Header().Set("WWW-Authenticate", `Basic realm="Inventory Admin"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func (fe *frontendServer) inventoryHandler(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(ctxKeyLog{}).(logrus.FieldLogger)

	loadAdminData()

	products, err := fe.getProducts(r.Context())
	if err != nil {
		renderHTTPError(log, r, w, errors.Wrap(err, "could not retrieve products"), http.StatusInternalServerError)
		return
	}

	adminMu.RLock()
	items := make([]*AdminProduct, len(products))
	for i, p := range products {
		items[i] = getAdminProduct(p)
	}
	// Append any admin-only added products (not in gRPC catalog)
	existingIDs := make(map[string]bool)
	for _, p := range products {
		existingIDs[p.GetId()] = true
	}
	for id, ap := range adminProducts {
		if !existingIDs[id] {
			items = append(items, ap)
		}
	}
	adminMu.RUnlock()

	flash := r.URL.Query().Get("flash")

	if err := templates.ExecuteTemplate(w, "inventory", injectCommonTemplateData(r, map[string]interface{}{
		"items": items,
		"flash": flash,
	})); err != nil {
		log.Println(err)
	}
}

// uploadProductPicture saves an uploaded image file and returns the URL path.
// fieldName is the multipart form field name. id is used as the filename base.
func uploadProductPicture(r *http.Request, fieldName, id string) string {
	file, header, err := r.FormFile(fieldName)
	if err != nil {
		return ""
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext == "" {
		ext = ".jpg"
	}
	dir := "./static/img/products/custom"
	_ = os.MkdirAll(dir, 0755)
	savePath := filepath.Join(dir, id+ext)
	f, err := os.Create(savePath)
	if err != nil {
		return ""
	}
	defer f.Close()
	if _, err := io.Copy(f, file); err != nil {
		return ""
	}
	return "/static/img/products/custom/" + id + ext
}

func (fe *frontendServer) updateInventoryHandler(w http.ResponseWriter, r *http.Request) {
	// 32 MB limit for file uploads
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		if err2 := r.ParseForm(); err2 != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
	}

	action := r.FormValue("action")

	adminMu.Lock()
	switch action {
	case "delete":
		id := r.FormValue("product_id")
		delete(adminProducts, id)

	case "add":
		id := strings.TrimSpace(r.FormValue("new_id"))
		if id != "" {
			price, _ := strconv.ParseFloat(r.FormValue("new_price"), 64)
			// Picture: prefer uploaded file, fall back to typed path
			picturePath := r.FormValue("new_picture")
			if uploaded := uploadProductPicture(r, "new_picture_file", id); uploaded != "" {
				picturePath = uploaded
			}
			adminProducts[id] = &AdminProduct{
				ID:          id,
				NameEN:      r.FormValue("new_name_en"),
				NameJA:      r.FormValue("new_name_ja"),
				Description: r.FormValue("new_desc_en"),
				DescJA:      r.FormValue("new_desc_ja"),
				Price:       price,
				Picture:     picturePath,
				Categories:  r.FormValue("new_categories"),
				Stock:       100,
				Hidden:      false,
			}
		}

	default: // bulk save
		ids := r.Form["product_ids"]
		for _, id := range ids {
			ap, exists := adminProducts[id]
			if !exists {
				ap = &AdminProduct{ID: id}
				adminProducts[id] = ap
			}
			ap.NameEN = r.FormValue("name_en_" + id)
			ap.NameJA = r.FormValue("name_ja_" + id)
			ap.Description = r.FormValue("desc_en_" + id)
			ap.DescJA = r.FormValue("desc_ja_" + id)
			if price, err := strconv.ParseFloat(r.FormValue("price_"+id), 64); err == nil {
				ap.Price = price
			}
			if qty, err := strconv.Atoi(r.FormValue("stock_" + id)); err == nil && qty >= 0 {
				ap.Stock = qty
			}
			ap.Hidden = r.FormValue("hidden_"+id) == "1"
			// Optional picture replacement per product
			if uploaded := uploadProductPicture(r, "pic_"+id, id); uploaded != "" {
				ap.Picture = uploaded
			}
		}
	}
	adminMu.Unlock()

	_ = saveAdminData()

	w.Header().Set("Location", baseUrl+"/admin/inventory?flash=saved")
	w.WriteHeader(http.StatusFound)
}
