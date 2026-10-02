package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func Test_HandleStartDM(t *testing.T) {
	me := uuid.New()
	other := uuid.New()
	cases := []struct {
		name     string
		user     *uuid.UUID
		body     map[string]string
		err      error
		wantCode int
		wantCall bool
	}{
		{"unauthorized", nil, map[string]string{"hiker_id": other.String()}, nil, 401, false},
		{"bad json", &me, nil, nil, 400, false},
		{"bad hiker id", &me, map[string]string{"hiker_id": "1233456"}, nil, 400, false},
		{"forbidden", &me, map[string]string{"hiker_id": other.String()}, apperr.Forbidden("x", "x"), 403, true},
		{"created", &me, map[string]string{"hiker_id": other.String()}, nil, 201, true},
	}
	target := "/api/dms"
	for _, cc := range cases {
		t.Run(cc.name, func(t *testing.T) {
			body, _ := json.Marshal(cc.body)
			if cc.body == nil {
				body = []byte("bad json")
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, target, bytes.NewReader(body))
			if cc.user != nil {
				r = r.WithContext(middleware.SetUserID(r.Context(), cc.user.String()))
			}

			f := newFakeMessageService()
			if cc.err != nil {
				f.err = cc.err
			}
			if cc.wantCode == 201 {
				f.conversation = &message.Conversation{ID: uuid.New(), Kind: message.ConversationKindDM}
			}

			s := &Server{messages: f}
			s.handleStartDM(w, r)
			if w.Code != cc.wantCode {
				t.Errorf("expected %d got %d", cc.wantCode, w.Code)
			}
			if w.Code == 201 {
				var resp struct {
					Data struct {
						Kind string `json:"kind"`
					} `json:"data"`
				}
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if resp.Data.Kind != "dm" {
					t.Errorf("kind = %q", resp.Data.Kind)
				}
			}

			if cc.wantCall {
				if f.gotHikerID != me || f.gotOtherID != other {
					t.Errorf("service got %v %v", f.gotHikerID, f.gotOtherID)
				}
			} else if f.gotOtherID != uuid.Nil {
				t.Error("service should not have been called")
			}

		})
	}
}

func Test_HandleDMTransitions(t *testing.T) {
	me := uuid.New()
	convID := uuid.New()
	handlers := []struct {
		name string
		call func(s *Server, w http.ResponseWriter, r *http.Request)
	}{
		{"accept", func(s *Server, w http.ResponseWriter, r *http.Request) { s.handleAcceptDM(w, r) }},
		{"decline", func(s *Server, w http.ResponseWriter, r *http.Request) { s.handleDeclineDM(w, r) }},
		{"reopen", func(s *Server, w http.ResponseWriter, r *http.Request) { s.handleReopenDM(w, r) }},
	}
	rows := []struct {
		name     string
		user     *uuid.UUID
		path     string
		err      error
		wantCode int
		wantCall bool
	}{
		{"unauthorized", nil, convID.String(), nil, 401, false},
		{"bad uuid", &me, "nope", nil, 400, false},
		{"forbidden", &me, convID.String(), apperr.Forbidden("x", "x"), 403, true},
		{"conflict", &me, convID.String(), apperr.Conflict("x", "x"), 409, true},
		{"ok", &me, convID.String(), nil, 204, true},
	}
	for _, h := range handlers {
		for _, cc := range rows {
			target := fmt.Sprintf("/api/conversations/%s/%s", cc.path, h.name)
			t.Run(h.name+"/"+cc.name, func(t *testing.T) {
				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodPost, target, nil)
				r.SetPathValue("id", cc.path)
				if cc.user != nil {
					r = r.WithContext(middleware.SetUserID(r.Context(), cc.user.String()))
				}

				f := newFakeMessageService()
				if cc.err != nil {
					f.err = cc.err
				}
				s := &Server{messages: f}
				h.call(s, w, r)
				if w.Code != cc.wantCode {
					t.Errorf("expected %d got %d", cc.wantCode, w.Code)
				}
				if cc.wantCall {
					if convID != f.gotConvID || me != f.gotHikerID {
						t.Errorf("service got %v %v", f.gotConvID, f.gotHikerID)
					}
				} else if f.gotConvID != uuid.Nil {
					t.Error("service should not called")
				}
			})
		}
	}
}

func Test_HandleListConversations(t *testing.T) {
	me := uuid.New()
	two := []*message.ConversationSummary{
		{ID: uuid.New(), Kind: message.ConversationKindOuting, Title: "Ingalls"},
		{ID: uuid.New(), Kind: message.ConversationKindDM, Title: "Amber"},
	}
	cases := []struct {
		name      string
		user      *uuid.UUID
		summaries []*message.ConversationSummary
		wantCode  int
		wantLen   int
	}{
		{"unauthorized", nil, nil, 401, 0},
		{"empty is [] not null", &me, nil, 200, 0},
		{"two rows", &me, two, 200, 2},
	}

	for _, cc := range cases {
		target := "/api/conversations"
		t.Run(cc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, target, nil)
			if cc.user != nil {
				r = r.WithContext(middleware.SetUserID(r.Context(), cc.user.String()))
			}

			f := newFakeMessageService()
			if cc.wantCode == 200 {
				f.conversationSummaries = cc.summaries
			}
			s := &Server{messages: f}

			s.handleListConversations(w, r)

			if w.Code != cc.wantCode {
				t.Errorf("expected %d got %d", cc.wantCode, w.Code)
			}
			if w.Code == 200 {
				if !strings.Contains(w.Body.String(), `"conversations":[`) {
					t.Error("expected contains summaries")
				}
				var resp struct {
					Data struct {
						Conversations []*message.ConversationSummary `json:"conversations"`
					} `json:"data"`
				}
				err := json.NewDecoder(w.Body).Decode(&resp)
				if err != nil {
					t.Fatalf("expected no error got %v", err)
				}
				if len(resp.Data.Conversations) != cc.wantLen {
					t.Error("len not equal")
				}
			}

		})
	}
}

func Test_HandleGetConversation(t *testing.T) {
	stranger := uuid.New()
	hikerA, hikerB := uuid.New(), uuid.New()
	dmStatus := message.DMStatusAccepted
	c := &message.ConversationView{
		ID:           uuid.New(),
		Kind:         message.ConversationKindDM,
		DmA:          &hikerA,
		DmB:          &hikerB,
		DmInitiator:  &hikerA,
		DmStatus:     &dmStatus,
		DmDeclinedBy: nil,
		CreatedAt:    time.Now(),
		Participants: []message.Participant{{HikerID: hikerA, Name: "A"}, {HikerID: hikerB, Name: "B"}},
	}
	cases := []struct {
		name     string
		user     *uuid.UUID
		path     string
		wantCode int
	}{
		{name: "unauthorized", user: nil, wantCode: 401, path: c.ID.String()},
		{name: "forbidden", user: &stranger, wantCode: 403, path: c.ID.String()},
		{name: "baduuid", user: &hikerA, path: "bad-uuid", wantCode: 400},
		{name: "not found", user: &hikerA, path: uuid.New().String(), wantCode: 404},
		{name: "success", user: &hikerB, wantCode: 200, path: c.ID.String()},
	}
	for _, cc := range cases {
		t.Run(cc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/api/conversations", nil)
			r.SetPathValue("id", cc.path)
			if cc.user != nil {
				r = r.WithContext(middleware.SetUserID(r.Context(), cc.user.String()))
			}

			f := newFakeMessageService()
			if cc.wantCode == 403 {
				f.err = apperr.Forbidden("forbidden", "forbidden")
			}
			if cc.wantCode == 404 {
				f.err = apperr.NotFound("not found", "not found")
			}
			f.conversationView = c
			s := &Server{messages: f}

			s.handleGetConversation(w, r)
			if cc.wantCode != w.Code {
				t.Fatalf("expected %d gor %d", cc.wantCode, w.Code)
			}
			if cc.wantCode == 200 {

				if f.gotConvID != c.ID {
					t.Errorf("expected convID: %v got %v", c.ID, f.gotConvID)
				}
				if f.gotHikerID != *c.DmB && f.gotHikerID != *c.DmA {
					t.Error("no member got in conversation")
				}
				resp := struct {
					Data *message.ConversationView `json:"data"`
				}{}
				dec := json.NewDecoder(w.Body)
				if err := dec.Decode(&resp); err != nil {
					t.Fatalf("expected no error got %v", err)
				}
				if resp.Data.ID != c.ID {
					t.Errorf("expected convID: %v got %v", c.ID, resp.Data.ID)
				}
				if len(resp.Data.Participants) != 2 {
					t.Errorf("expected 2 got %d", len(resp.Data.Participants))
				}
				hikerNames := make(map[string]bool)
				hikerNames["A"] = true
				hikerNames["B"] = true
				for _, p := range resp.Data.Participants {
					if _, ok := hikerNames[p.Name]; !ok {
						t.Errorf("unexpected hiker name %s", p.Name)
					}
				}
			}
		})
	}
}
