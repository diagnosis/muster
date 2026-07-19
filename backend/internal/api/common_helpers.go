package api

import (
	"encoding/json"
	"net/http"

	"github.com/diagnosis/go-toolkit/v2/apperr"
	"github.com/diagnosis/go-toolkit/v2/middleware"
	"github.com/google/uuid"
)

// getAuthenticatedUserID extracts and parses the user ID that RequireAuth placed in the context. A parse failure indicates a signer/middleware bug, not a client error.
func getAuthenticatedUserID(r *http.Request) (uuid.UUID, error) {
	idStr, ok := middleware.GetUserID(r.Context())
	if !ok {
		return uuid.Nil, apperr.Unauthorized("unauthorized", "no access token or token expired")
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, apperr.Unauthorized("unauthorized", "user id claim is not a valid uuid — signer/middleware inconsistency")
	}
	return id, nil
}

// decodeJSON strictly decodes r's body into dst, rejecting unknown fields.
func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return apperr.BadRequest("invalid request body", "json decode failed", err)
	}
	return nil
}

// pathUUID parses the named path parameter as a UUID.
func pathUUID(r *http.Request, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(r.PathValue(name))
	if err != nil {
		return uuid.Nil, apperr.BadRequest("invalid id in path", "path param "+name+" is not a uuid", err)
	}
	return id, nil
}
