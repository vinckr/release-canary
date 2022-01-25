/*
 * Copyright © 2015-2018 Aeneas Rekkas <aeneas+oss@aeneas.io>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * @author		Aeneas Rekkas <aeneas+oss@aeneas.io>
 * @Copyright 	2017-2018 Aeneas Rekkas <aeneas+oss@aeneas.io>
 * @license 	Apache-2.0
 */

package consent

import (
	"net/http"

	"github.com/ory/x/errorsx"

	"github.com/gorilla/sessions"

	"github.com/ory/fosite"
	"github.com/ory/x/mapx"

	"github.com/ory/hydra/client"
)

func sanitizeClientFromRequest(ar fosite.AuthorizeRequester) *client.Client {
	return sanitizeClient(ar.GetClient().(*client.Client))
}

func sanitizeClient(c *client.Client) *client.Client {
	cc := new(client.Client)
	// Remove the hashed secret here
	*cc = *c
	cc.Secret = ""
	return cc
}

func matchScopes(scopeStrategy fosite.ScopeStrategy, previousConsent []HandledConsentRequest, requestedScope []string) *HandledConsentRequest {
	for _, cs := range previousConsent {
		var found = true
		for _, scope := range requestedScope {
			if !scopeStrategy(cs.GrantedScope, scope) {
				found = false
				break
			}
		}

		if found {
			return &cs
		}
	}

	return nil
}

func createCsrfSession(w http.ResponseWriter, r *http.Request, store sessions.Store, name, csrf string, secure bool, sameSiteMode http.SameSite, sameSiteLegacyWorkaround bool) error {
	// Errors can be ignored here, because we always get a session session back. Error typically means that the
	// session doesn't exist yet.
	session, _ := store.Get(r, CookieName(secure, name))
	session.Values["csrf"] = csrf
	session.Options.HttpOnly = true
	session.Options.Secure = secure
	session.Options.SameSite = sameSiteMode
	if err := session.Save(r, w); err != nil {
		return errorsx.WithStack(err)
	}
	if sameSiteMode == http.SameSiteNoneMode && sameSiteLegacyWorkaround {
		return createCsrfSession(w, r, store, legacyCsrfSessionName(name), csrf, secure, 0, false)
	}
	return nil
}

func validateCsrfSession(r *http.Request, store sessions.Store, name, expectedCSRF string, sameSiteLegacyWorkaround, secure bool) error {
	if cookie, err := getCsrfSession(r, store, name, sameSiteLegacyWorkaround, secure); err != nil {
		return errorsx.WithStack(fosite.ErrRequestForbidden.WithHint("CSRF session cookie could not be decoded."))
	} else if csrf, err := mapx.GetString(cookie.Values, "csrf"); err != nil {
		return errorsx.WithStack(fosite.ErrRequestForbidden.WithHint("No CSRF value available in the session cookie."))
	} else if csrf != expectedCSRF {
		return errorsx.WithStack(fosite.ErrRequestForbidden.WithHint("The CSRF value from the token does not match the CSRF value from the data store."))
	}

	return nil
}

func getCsrfSession(r *http.Request, store sessions.Store, name string, sameSiteLegacyWorkaround, secure bool) (*sessions.Session, error) {
	cookie, err := store.Get(r, CookieName(secure, name))
	if sameSiteLegacyWorkaround && (err != nil || len(cookie.Values) == 0) {
		return store.Get(r, legacyCsrfSessionName(name))
	}
	return cookie, err
}

func legacyCsrfSessionName(name string) string {
	return name + "_legacy"
}

func CookieName(secure bool, name string) string {
	if !secure {
		return name + "_insecure"
	}
	return name
}
