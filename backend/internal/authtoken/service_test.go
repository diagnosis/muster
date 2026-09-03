package authtoken

import (
	"context"
	"testing"
	"time"

	"github.com/diagnosis/go-toolkit/v3/apperr"
	"github.com/google/uuid"
)

func wantStatus(t *testing.T, err error, want apperr.Status) {
	t.Helper()
	se, ok := apperr.AsStatusErr(err)
	if !ok || se.Status != want {
		t.Fatalf("got %v, want status %v", err, want)
	}
}

func TestService_Consume(t *testing.T) {
	svc := NewService(newFakeStore())
	hikerID := uuid.New()
	raw, err := svc.Mint(context.Background(), hikerID, PurposeEmailVerification, time.Hour)
	if err != nil {
		t.Fatalf("expected no error on minting got %v", err)
	}
	if raw == "" {
		t.Fatal("expected raw is not empty string")
	}
	got, err := svc.Consume(context.Background(), raw, PurposeEmailVerification)
	if err != nil {
		t.Fatalf("expected no error on consume got %v", err)
	}
	if got != hikerID {
		t.Errorf("expected hikerID %v got %v", hikerID, got)
	}
}

func TestService_DoubleConsume(t *testing.T) {
	svc := NewService(newFakeStore())
	hikerID := uuid.New()
	raw, err := svc.Mint(context.Background(), hikerID, PurposeEmailVerification, time.Hour)
	if err != nil {
		t.Fatalf("expected no error on minting got %v", err)
	}
	if raw == "" {
		t.Fatal("expected raw is not empty string")
	}
	_, err = svc.Consume(context.Background(), raw, PurposeEmailVerification)
	if err != nil {
		t.Fatalf("expected no error on consume got %v", err)
	}
	_, err = svc.Consume(context.Background(), raw, PurposeEmailVerification)
	if err == nil {
		t.Fatalf("expected error got no error")
	}
	wantStatus(t, err, apperr.CodeConflict)
}
func TestService_WrongPurpose(t *testing.T) {
	svc := NewService(newFakeStore())
	hikerID := uuid.New()
	raw, err := svc.Mint(context.Background(), hikerID, PurposeEmailVerification, time.Hour)
	if err != nil {
		t.Fatalf("expected no error on minting got %v", err)
	}
	if raw == "" {
		t.Fatal("expected raw is not empty string")
	}
	_, err = svc.Consume(context.Background(), raw, PurposeForgotPassword)
	if err == nil {
		t.Fatal("expected error got no error")
	}
	wantStatus(t, err, apperr.CodeConflict)
}

func TestService_Expired(t *testing.T) {
	svc := NewService(newFakeStore())
	hikerID := uuid.New()
	raw, err := svc.Mint(context.Background(), hikerID, PurposeEmailVerification, -time.Hour)
	if err != nil {
		t.Fatalf("expected no error on minting got %v", err)
	}
	if raw == "" {
		t.Fatal("expected raw is not empty string")
	}
	_, err = svc.Consume(context.Background(), raw, PurposeEmailVerification)
	if err == nil {
		t.Fatal("expected error got no error")
	}
	wantStatus(t, err, apperr.CodeConflict)
}
