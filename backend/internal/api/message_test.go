package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/diagnosis/go-toolkit/v3/middleware"
	"github.com/diagnosis/muster/internal/message"
	"github.com/diagnosis/muster/internal/outing"
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
	s := &Server{
		messages: message.NewService(newMessageFakeStore(), newFakeBroadcaster()),
	}
	s.handlePostMessage(w, r)
	if w.Code != 404 {
		t.Errorf("expected 404 got %d", w.Code)
	}
}
func Test_Message_HandlePostMessage_Unauthorized(t *testing.T) {
	convID := uuid.New()
	target := fmt.Sprintf("/api/conversations/%s/messages", convID)
	body, _ := json.Marshal(map[string]string{"body": "hello"})
	r := httptest.NewRequest(http.MethodPost, target, bytes.NewReader(body))
	r.SetPathValue("id", convID.String())

	w := httptest.NewRecorder()
	s := &Server{
		messages: message.NewService(newMessageFakeStore(), newFakeBroadcaster()),
	}
	s.handlePostMessage(w, r)
	if w.Code != 401 {
		t.Errorf("expected 401 got %d", w.Code)
	}
}
func Test_Message_HandlePostMessage_NonMember(t *testing.T) {
	hikerID := uuid.New()
	stranger := uuid.New()
	hostID := uuid.New()
	outingID := uuid.New()
	f := newMessageFakeStore()
	conv := f.addOutingConversation(outingID, hostID, outing.StatusOpen, hikerID)

	target := fmt.Sprintf("/api/conversations/%s/messages", conv.ID)
	body, _ := json.Marshal(map[string]string{"body": "hello"})

	r := httptest.NewRequest(http.MethodPost, target, bytes.NewReader(body))
	r.SetPathValue("id", conv.ID.String())
	r = r.WithContext(middleware.SetUserID(r.Context(), stranger.String()))
	w := httptest.NewRecorder()

	s := &Server{
		messages: message.NewService(f, newFakeBroadcaster()),
	}

	s.handlePostMessage(w, r)
	if w.Code != 403 {
		t.Errorf("expected 403 got %d", w.Code)
	}
}
func Test_Message_HandlePostMessage_BadJSON(t *testing.T) {
	hikerID := uuid.New()
	hostID := uuid.New()
	outingID := uuid.New()
	f := newMessageFakeStore()
	conv := f.addOutingConversation(outingID, hostID, outing.StatusOpen, hikerID)

	target := fmt.Sprintf("/api/conversations/%s/messages", conv.ID)

	r := httptest.NewRequest(http.MethodPost, target, bytes.NewReader([]byte("{not json}")))
	r.SetPathValue("id", conv.ID.String())
	r = r.WithContext(middleware.SetUserID(r.Context(), hikerID.String()))
	w := httptest.NewRecorder()

	s := &Server{
		messages: message.NewService(f, newFakeBroadcaster()),
	}

	s.handlePostMessage(w, r)
	if w.Code != 400 {
		t.Errorf("expected 400 got %d", w.Code)
	}
}

func Test_Message_HandlePostMessage_Happy(t *testing.T) {
	hikerID := uuid.New()
	hostID := uuid.New()
	outingID := uuid.New()
	f := newMessageFakeStore()
	conv := f.addOutingConversation(outingID, hostID, outing.StatusOpen, hikerID)

	target := fmt.Sprintf("/api/conversations/%s/messages", conv.ID)
	body, _ := json.Marshal(map[string]string{"body": "hello"})

	r := httptest.NewRequest(http.MethodPost, target, bytes.NewReader(body))
	r.SetPathValue("id", conv.ID.String())
	r = r.WithContext(middleware.SetUserID(r.Context(), hikerID.String()))
	w := httptest.NewRecorder()

	s := &Server{
		messages: message.NewService(f, newFakeBroadcaster()),
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
	if resp.Data.ConversationID != conv.ID {
		t.Errorf("expected %v got %v", conv.ID, resp.Data.ConversationID)
	}
	if resp.Data.Seq != 1 {
		t.Errorf("expected 1 got %d", resp.Data.Seq)
	}

}

func Test_Message_HandlePostMessage_BadUUID(t *testing.T) {
	hikerID := uuid.New()
	hostID := uuid.New()
	outingID := uuid.New()
	f := newMessageFakeStore()
	conv := f.addOutingConversation(outingID, hostID, outing.StatusOpen, hikerID)

	target := fmt.Sprintf("/api/conversations/%s/messages", conv.ID)
	body, _ := json.Marshal(map[string]string{"body": "hello"})

	r := httptest.NewRequest(http.MethodPost, target, bytes.NewReader(body))
	r.SetPathValue("id", "not-a-uuid")
	r = r.WithContext(middleware.SetUserID(r.Context(), hikerID.String()))
	w := httptest.NewRecorder()

	s := &Server{
		messages: message.NewService(f, newFakeBroadcaster()),
	}

	s.handlePostMessage(w, r)
	if w.Code != 400 {
		t.Errorf("expected 400 got %d", w.Code)
	}
}
