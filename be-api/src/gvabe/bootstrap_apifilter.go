package gvabe

import (
	"log"
	"os"
	"strings"

	"main/src/goapi"
	"main/src/itineris"
)

/*
Setup API filters: application register its api-handlers by calling router.SetHandler(apiName, apiHandlerFunc)

    - api-handler function must has the following signature: func (itineris.ApiContext, itineris.ApiAuth, itineris.ApiParams) *itineris.ApiResult
*/
func initApiFilters(apiRouter *itineris.ApiRouter) {
	var apiFilter itineris.IApiFilter = nil
	// appName := goapi.AppConfig.GetString("app.name")
	// appVersion := goapi.AppConfig.GetString("app.version")

	// filters are LIFO:
	// - request goes through the last filter to the first one
	// - response goes through the first filter to the last one
	// suggested order of filters:
	// - Request logger should be the last one to capture full request/response

	if DEBUG {
		// apiFilter = itineris.NewAddPerfInfoFilter(goapi.ApiRouter, apiFilter)
		apiFilter = itineris.NewLoggingFilter(
			goapi.ApiRouter,
			apiFilter,
			itineris.NewWriterPerfLogger(
				os.Stderr,
				goapi.AppConfig.GetString("app.name"),
				goapi.AppConfig.GetString("app.version")))
	}
	apiFilter = &GVAFEAuthenticationFilter{BaseApiFilter: &itineris.BaseApiFilter{ApiRouter: apiRouter, NextFilter: apiFilter}}
	// if DEBUG {
	// 	apiFilter = itineris.NewLoggingFilter(
	// 		goapi.ApiRouter,
	// 		apiFilter,
	// 		itineris.NewWriterRequestLogger(
	// 			os.Stdout,
	// 			goapi.AppConfig.GetString("app.name"),
	// 			goapi.AppConfig.GetString("app.version")))
	// }
	apiRouter.SetApiFilter(apiFilter)
}

const ctxFieldSession = "_session"

/*
GVAFEAuthenticationFilter performs authentication check before calling API and issues new access token if existing one is about to expire.

	- AppId must be "exter_fe"
	- AccessToken must be valid (allocated and active)
*/
type GVAFEAuthenticationFilter struct {
	*itineris.BaseApiFilter
}

/*
Call implements IApiFilter.Call

	- This function first authenticates API call.
	- If authentication is successful, *SessionClaims instance is populated to 'ctx' under field 'ctxFieldSession'
	- Finally, if the login session is about to expire, this function renews the login token and returns it in result's "extra" field.
*/
func (f *GVAFEAuthenticationFilter) Call(handler itineris.IApiHandler, ctx *itineris.ApiContext, auth *itineris.ApiAuth, params *itineris.ApiParams) *itineris.ApiResult {
	sessionClaim, err := f.authenticate(ctx, auth)
	if err != nil {
		return itineris.NewApiResult(itineris.StatusNoPermission).SetMessage(err.Error())
	}
	ctx.SetContextValue(ctxFieldSession, sessionClaim)
	if f.NextFilter != nil {
		return f.NextFilter.Call(handler, ctx, auth, params)
	}
	result := handler(ctx, auth, params)
	// if sessionClaim != nil && sessionClaim.isGoingExpired(loginSessionNearExpiry) {
	// 	// extends login session
	// 	expiry := time.Now().Add(loginSessionTtl * time.Second)
	// 	sessionClaim.ExpiresAt = expiry.Unix()
	// 	jws, _ := genJws(sessionClaim)
	// 	result.AddExtraInfo(apiResultExtraAccessToken, jws)
	// }
	return result
}

/*
authenticate authenticates an API call.

This function expects auth.access_token is a JWT.
Upon successful authentication, this function returns the SessionClaims decoded from JWT; otherwise, error is returned.
*/
func (f *GVAFEAuthenticationFilter) authenticate(ctx *itineris.ApiContext, auth *itineris.ApiAuth) (*SessionClaims, error) {
	publicApi, ok := publicApis[ctx.GetApiName()]
	if !ok || !publicApi {
		// need app-id
		if !strings.HasPrefix(auth.GetAppId(), frontendAppIdPrefix) {
			return nil, errorInvalidClient
		}
	}
	if ok {
		// for public APIs, there is no access_token required
		return nil, nil
	}
	sessionClaim, err := parseLoginToken(auth.GetAccessToken())
	if err != nil {
		log.Printf("Cannot decode JWT [API: %s / Error: %e", ctx.GetApiName(), err)
		return nil, errorInvalidJwt
	}
	if sessionClaim.isExpired() {
		return nil, errorExpiredJwt
	}
	return sessionClaim, nil
}
