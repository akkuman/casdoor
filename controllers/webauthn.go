// Copyright 2022 The Casdoor Authors. All Rights Reserved.
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

package controllers

import (
	"bytes"
	"io/ioutil"

	"github.com/casdoor/casdoor/object"
	"github.com/casdoor/casdoor/util"
	"github.com/duo-labs/webauthn/protocol"
	"github.com/duo-labs/webauthn/webauthn"
)

// @Title WebAuthnSignupBegin
// @Tag User API
// @Description WebAuthn Registration Flow 1st stage
// @Success 200 {object} protocol.CredentialCreation The CredentialCreationOptions object
// @router /webauthn/signup/begin [get]
func (c *ApiController) WebAuthnSignupBegin() {
	webauthnObj := object.GetWebAuthnObject(c.Ctx.Request.Host)
	user := c.getCurrentUser()
	if user == nil {
		c.ResponseError("Please login first.")
		return
	}

	registerOptions := func(credCreationOpts *protocol.PublicKeyCredentialCreationOptions) {
		credCreationOpts.CredentialExcludeList = user.CredentialExcludeList()
	}
	options, sessionData, err := webauthnObj.BeginRegistration(
		user,
		registerOptions,
	)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	c.SetSession("registration", *sessionData)
	c.Data["json"] = options
	c.ServeJSON()
}

// @Title WebAuthnSignupFinish
// @Tag User API
// @Description WebAuthn Registration Flow 2nd stage
// @Param   body    body   protocol.CredentialCreationResponse  true        "authenticator attestation Response"
// @Success 200 {object} Response "The Response object"
// @router /webauthn/signup/finish [post]
func (c *ApiController) WebAuthnSignupFinish() {
	webauthnObj := object.GetWebAuthnObject(c.Ctx.Request.Host)
	user := c.getCurrentUser()
	if user == nil {
		c.ResponseError("Please login first.")
		return
	}
	sessionObj := c.GetSession("registration")
	sessionData, ok := sessionObj.(webauthn.SessionData)
	if !ok {
		c.ResponseError("Please call WebAuthnSignupBegin first")
		return
	}
	c.Ctx.Request.Body = ioutil.NopCloser(bytes.NewBuffer(c.Ctx.Input.RequestBody))

	credential, err := webauthnObj.FinishRegistration(user, sessionData, c.Ctx.Request)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	isGlobalAdmin := c.IsGlobalAdmin()
	user.AddCredentials(*credential, isGlobalAdmin)
	c.ResponseOk()
}

// @Title WebAuthnSigninBegin
// @Tag Login API
// @Description WebAuthn Login Flow 1st stage
// @Param   owner     query    string  true        "owner"
// @Param   name     query    string  true        "name"
// @Success 200 {object} protocol.CredentialAssertion The CredentialAssertion object
// @router /webauthn/signin/begin [get]
func (c *ApiController) WebAuthnSigninBegin() {
	webauthnObj := object.GetWebAuthnObject(c.Ctx.Request.Host)
	userOwner := c.Input().Get("owner")
	userName := c.Input().Get("name")
	user := object.GetUserByFields(userOwner, userName)
	if user == nil {
		c.ResponseError("Please Giveout Owner and Username.")
		return
	}
	options, sessionData, err := webauthnObj.BeginLogin(user)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	c.SetSession("authentication", *sessionData)
	c.Data["json"] = options
	c.ServeJSON()
}

// @Title WebAuthnSigninBegin
// @Tag Login API
// @Description WebAuthn Login Flow 2nd stage
// @Param   body    body   protocol.CredentialAssertionResponse  true        "authenticator assertion Response"
// @Success 200 {object} Response "The Response object"
// @router /webauthn/signin/finish [post]
func (c *ApiController) WebAuthnSigninFinish() {
	webauthnObj := object.GetWebAuthnObject(c.Ctx.Request.Host)
	sessionObj := c.GetSession("authentication")
	sessionData, ok := sessionObj.(webauthn.SessionData)
	if !ok {
		c.ResponseError("Please call WebAuthnSigninBegin first")
		return
	}
	c.Ctx.Request.Body = ioutil.NopCloser(bytes.NewBuffer(c.Ctx.Input.RequestBody))
	userId := string(sessionData.UserID)
	user := object.GetUser(userId)
	_, err := webauthnObj.FinishLogin(user, sessionData, c.Ctx.Request)
	if err != nil {
		c.ResponseError(err.Error())
		return
	}
	c.SetSessionUsername(userId)
	util.LogInfo(c.Ctx, "API: [%s] signed in", userId)
	c.ResponseOk(userId)
}
