package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/diagnosis/go-toolkit/v3/apperr"
	"github.com/diagnosis/go-toolkit/v3/middleware"
	"github.com/diagnosis/muster/internal/message"
	"github.com/google/uuid"
)

type testMessage struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	HikerID        uuid.UUID `json:"hiker_id"`
	Body           string    `json:"body"`
	Seq            int       `json:"seq"`
}

func Test_Message_HandlePostMessage_NoConversation(t *testing.T) {
	convID := uuid.New()
	hikerID := uuid.New()
	target := fmt.Sprintf("/api/conversations/%s/messages", convID)
	body, _ := json.Marshal(map[string]string{"body": "hello"})
	r := httptest.NewRequest(http.MethodPost, target, bytes.NewReader(body))
	r.SetPathValue("id", convID.String())
	r = r.WithContext(middleware.SetUserID(r.Context(), hikerID.String()))
	w := httptest.NewRecorder()
	f := newFakeMessageService()
	f.err = apperr.NotFound("conversation not found", "no row for id")
	s := &Server{messages: f}
	s.handlePostMessage(w, r)
	if w.Code != 404 {
		t.Errorf("expected 404 got %d", w.Code)
	}
	if f.gotConvID != convID {
		t.Errorf("expected %v got %v", convID, f.gotConvID)
	}
	if f.gotHikerID != hikerID {
		t.Errorf("expected %v got %v", hikerID, f.gotHikerID)
	}
	if f.gotBody != "hello" {
		t.Errorf("expected hello got %s", f.gotBody)
	}
}
func Test_Message_HandlePostMessage_Unauthorized(t *testing.T) {
	convID := uuid.New()
	target := fmt.Sprintf("/api/conversations/%s/messages", convID)
	body, _ := json.Marshal(map[string]string{"body": "hello"})
	r := httptest.NewRequest(http.MethodPost, target, bytes.NewReader(body))
	r.SetPathValue("id", convID.String())
	f := newFakeMessageService()
	w := httptest.NewRecorder()
	s := &Server{
		messages: f,
	}
	s.handlePostMessage(w, r)
	if w.Code != 401 {
		t.Errorf("expected 401 got %d", w.Code)
	}
	if f.gotConvID != uuid.Nil {
		t.Error("service should not have been called")
	}

}
func Test_Message_HandlePostMessage_NonMember(t *testing.T) {
	stranger := uuid.New()
	f := newFakeMessageService()
	convID := uuid.New()
	f.err = apperr.Forbidden("forbidden", "forbidden")
	target := fmt.Sprintf("/api/conversations/%s/messages", convID)
	body, _ := json.Marshal(map[string]string{"body": "hello"})

	r := httptest.NewRequest(http.MethodPost, target, bytes.NewReader(body))
	r.SetPathValue("id", convID.String())
	r = r.WithContext(middleware.SetUserID(r.Context(), stranger.String()))
	w := httptest.NewRecorder()

	s := &Server{
		messages: f,
	}

	s.handlePostMessage(w, r)
	if w.Code != 403 {
		t.Errorf("expected 403 got %d", w.Code)
	}
	if f.gotConvID != convID {
		t.Errorf("expected %v got %v", convID, f.gotConvID)
	}
	if f.gotHikerID != stranger {
		t.Errorf("expected %v got %v", stranger, f.gotHikerID)
	}
	if f.gotBody != "hello" {
		t.Errorf("expected hello got %s", f.gotBody)
	}
}
func Test_Message_HandlePostMessage_BadJSON(t *testing.T) {
	hikerID := uuid.New()
	f := newFakeMessageService()
	convID := uuid.New()

	target := fmt.Sprintf("/api/conversations/%s/messages", convID)

	r := httptest.NewRequest(http.MethodPost, target, bytes.NewReader([]byte("{not json}")))
	r.SetPathValue("id", convID.String())
	r = r.WithContext(middleware.SetUserID(r.Context(), hikerID.String()))
	w := httptest.NewRecorder()

	s := &Server{
		messages: f,
	}

	s.handlePostMessage(w, r)
	if w.Code != 400 {
		t.Errorf("expected 400 got %d", w.Code)
	}
	if f.gotConvID != uuid.Nil {
		t.Error("service should not have been called")
	}
}

func Test_Message_HandlePostMessage_Happy(t *testing.T) {
	hikerID := uuid.New()
	f := newFakeMessageService()
	convID := uuid.New()
	m := &message.Message{
		ConversationID: convID,
		HikerID:        hikerID,
		Body:           "hello",
		Seq:            1,
	}
	f.message = m
	f.messages = append(f.messages, m)
	target := fmt.Sprintf("/api/conversations/%s/messages", convID)
	body, _ := json.Marshal(map[string]string{"body": "hello"})

	r := httptest.NewRequest(http.MethodPost, target, bytes.NewReader(body))
	r.SetPathValue("id", convID.String())
	r = r.WithContext(middleware.SetUserID(r.Context(), hikerID.String()))
	w := httptest.NewRecorder()

	s := &Server{
		messages: f,
	}

	s.handlePostMessage(w, r)
	if w.Code != 201 {
		t.Errorf("expected 201 got %d", w.Code)
	}

	var resp struct {
		Data testMessage `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Data.Body != "hello" {
		t.Errorf("expected hello got %s", resp.Data.Body)
	}
	if resp.Data.HikerID != hikerID {
		t.Errorf("expected %v got %v", hikerID, resp.Data.HikerID)
	}
	if resp.Data.ConversationID != convID {
		t.Errorf("expected %v got %v", convID, resp.Data.ConversationID)
	}
	if resp.Data.Seq != 1 {
		t.Errorf("expected 1 got %d", resp.Data.Seq)
	}
	if f.gotConvID != convID {
		t.Errorf("expected %v got %v", convID, f.gotConvID)
	}
	if f.gotHikerID != hikerID {
		t.Errorf("expected %v got %v", hikerID, f.gotHikerID)
	}
	if f.gotBody != "hello" {
		t.Errorf("expected hello got %s", f.gotBody)
	}

}

func Test_Message_HandlePostMessage_BadUUID(t *testing.T) {
	hikerID := uuid.New()
	f := newFakeMessageService()
	convID := uuid.New()

	target := fmt.Sprintf("/api/conversations/%s/messages", convID)
	body, _ := json.Marshal(map[string]string{"body": "hello"})

	r := httptest.NewRequest(http.MethodPost, target, bytes.NewReader(body))
	r.SetPathValue("id", "not-a-uuid")
	r = r.WithContext(middleware.SetUserID(r.Context(), hikerID.String()))
	w := httptest.NewRecorder()

	s := &Server{
		messages: f,
	}

	s.handlePostMessage(w, r)
	if w.Code != 400 {
		t.Errorf("expected 400 got %d", w.Code)
	}
	if f.gotConvID != uuid.Nil {
		t.Error("service should not have been called")
	}
}

// list handler tests
func Test_Message_HandleListMessages_Unauthorized(t *testing.T) {
	convID := uuid.New()
	target := fmt.Sprintf("/api/conversations/%s/messages", convID)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, target, nil)
	r.SetPathValue("id", convID.String())

	f := newFakeMessageService()
	s := &Server{messages: f}
	s.handleListMessages(w, r)
	if w.Code != 401 {
		t.Errorf("expected error code %d got %d", 401, w.Code)
	}
	if f.gotConvID != uuid.Nil {
		t.Error("service should not have been called")
	}

}
func Test_Message_HandleListMessages_BadUUID(t *testing.T) {
	hikerID := uuid.New()
	convID := uuid.New()
	target := fmt.Sprintf("/api/conversations/%s/messages", convID)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, target, nil)
	r.SetPathValue("id", "not uuid")
	r = r.WithContext(middleware.SetUserID(r.Context(), hikerID.String()))
	f := newFakeMessageService()
	s := &Server{messages: f}
	s.handleListMessages(w, r)
	if w.Code != 400 {
		t.Errorf("expected error code %d got %d", 400, w.Code)
	}
	if f.gotConvID != uuid.Nil {
		t.Error("service should not have been called")
	}
}

func Test_Message_HandleListMessages_Stranger(t *testing.T) {
	stranger := uuid.New()
	f := newFakeMessageService()
	convID := uuid.New()
	f.err = apperr.Forbidden("forbidden", "forbidden")
	target := fmt.Sprintf("/api/conversations/%s/messages", convID)

	r := httptest.NewRequest(http.MethodGet, target, nil)
	r.SetPathValue("id", convID.String())
	r = r.WithContext(middleware.SetUserID(r.Context(), stranger.String()))
	w := httptest.NewRecorder()

	s := &Server{
		messages: f,
	}

	s.handleListMessages(w, r)
	if w.Code != 403 {
		t.Errorf("expected 403 got %d", w.Code)
	}
	if f.gotConvID != convID {
		t.Errorf("expected %v got %v", convID, f.gotConvID)
	}
	if f.gotHikerID != stranger {
		t.Errorf("expected %v got %v", stranger, f.gotHikerID)
	}

}

func Test_HandleListMessages_Happy(t *testing.T) {
	hikerID := uuid.New()
	convID := uuid.New()
	target := fmt.Sprintf("/api/conversations/%s/messages", convID)
	r := httptest.NewRequest(http.MethodGet, target, nil)
	r.SetPathValue("id", convID.String())
	r = r.WithContext(middleware.SetUserID(r.Context(), hikerID.String()))
	w := httptest.NewRecorder()
	f := newFakeMessageService()
	s := &Server{messages: f}
	messages := []*message.Message{
		{ConversationID: convID, HikerID: hikerID, Seq: 1, Body: "hello"},
		{ConversationID: convID, HikerID: hikerID, Seq: 2, Body: "mahmut"},
		{ConversationID: convID, HikerID: hikerID, Seq: 3, Body: "halay"},
		{ConversationID: convID, HikerID: hikerID, Seq: 4, Body: "lolololo"},
	}
	f.messages = messages
	s.handleListMessages(w, r)
	if w.Code != 200 {
		t.Errorf("expected 200 got %d", w.Code)
	}
	var resp struct {
		Data struct {
			Messages []testMessage `json:"messages"`
		} `json:"data"`
	}
	dec := json.NewDecoder(w.Body)
	err := dec.Decode(&resp)
	if err != nil {
		t.Fatalf("got error n decoding %v", err)
	}
	if len(resp.Data.Messages) != 4 {
		t.Errorf("expected list size 4 got %d", len(resp.Data.Messages))
	}
	if f.gotConvID != convID {
		t.Errorf("expected %v got %v", convID, f.gotConvID)
	}
	if f.gotHikerID != hikerID {
		t.Errorf("expected %v got %v", hikerID, f.gotHikerID)
	}
	for i, b := range resp.Data.Messages {
		if b.Seq != i+1 {
			t.Errorf("expected %d got %d", i+1, b.Seq)
		}
	}
}

func Test_HandleListMessages_EmptyList(t *testing.T) {
	hikerID := uuid.New()
	convID := uuid.New()
	target := fmt.Sprintf("/api/conversations/%s/messages", convID)
	r := httptest.NewRequest(http.MethodGet, target, nil)
	r.SetPathValue("id", convID.String())
	r = r.WithContext(middleware.SetUserID(r.Context(), hikerID.String()))
	w := httptest.NewRecorder()
	f := newFakeMessageService()
	s := &Server{messages: f}
	s.handleListMessages(w, r)
	if w.Code != 200 {
		t.Errorf("expected 200 got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"messages":[]`) {
		t.Errorf("expected empty array got %v", w.Body.String())
	}
}

// delete test
func Test_HandleDeleteMessage(t *testing.T) {
	msgID := uuid.New()
	hiker := uuid.New()
	cases := []struct {
		name       string
		user       *uuid.UUID // nil → no SetUserID
		path       string     // msgID.String() or "not-a-uuid"
		err        error      // f.err
		wantCode   int
		wantCalled bool
	}{
		{"unauthorized", nil, msgID.String(), nil, http.StatusUnauthorized, false},
		{"bad uuid", &hiker, "not-a-uuid", nil, http.StatusBadRequest, false},
		{"not found", &hiker, msgID.String(), apperr.NotFound("x", "x"), http.StatusNotFound, true},
		{"forbidden", &hiker, msgID.String(), apperr.Forbidden("x", "x"), http.StatusForbidden, true},
		{"deleted", &hiker, msgID.String(), nil, http.StatusNoContent, true},
	}
	target := fmt.Sprintf("/api/messages/%s", msgID)
	for _, cc := range cases {
		t.Run(cc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodDelete, target, nil)
			r.SetPathValue("id", cc.path)
			if cc.user != nil {
				r = r.WithContext(middleware.SetUserID(r.Context(), cc.user.String()))
			}

			f := newFakeMessageService()
			f.err = cc.err
			s := &Server{messages: f}
			s.handleDeleteMessage(w, r)
			if w.Code != cc.wantCode {
				t.Errorf("expected code %d got %d", cc.wantCode, w.Code)
			}
			if cc.wantCalled {
				if f.gotMsgID != msgID {
					t.Errorf("expected message id: %v got %v", msgID, f.gotMsgID)
				}
				if f.gotHikerID != hiker {
					t.Errorf("expected hiker id: %v got %v", hiker, f.gotHikerID)
				}
			} else if f.gotMsgID != uuid.Nil {
				t.Errorf("expected message id to be nil got %v", f.gotMsgID)

			}

		})
	}
}
