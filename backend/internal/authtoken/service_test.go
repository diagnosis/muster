package authtoken

import (
	"context"
	"errors"
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

func Test_HikersCodesDontCollide(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)
	hiker1, hiker2 := uuid.New(), uuid.New()
	svc.otp = func() (string, error) {
		return "070788", nil
	}
	_, err := svc.MintCode(context.Background(), hiker1, 10*time.Minute)
	if err != nil {
		t.Fatalf("expected to get no error got %v", err)
	}
	_, err = svc.MintCode(context.Background(), hiker2, 10*time.Minute)
	if err != nil {
		t.Fatalf("expected to get no error got %v", err)
	}
	if len(f.tokens) != 2 {
		t.Fatalf("Expected len 2 got %d", len(f.tokens))
	}
	tokens := []*Token{}
	for _, tok := range f.tokens {
		tokens = append(tokens, tok)
	}
	if tokens[0].Hash == tokens[1].Hash {
		t.Errorf("expected tokens are you unique got %s==%s", tokens[0].Hash, tokens[1].Hash)
	}

}

func Test_PasswordlessLogin_SecondSendBurnsFirstCode(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)
	clock := time.Now().Add(-10 * time.Hour)
	svc.now = func() time.Time {
		return clock
	}
	hiker1 := uuid.New()
	code1, err := svc.MintCode(context.Background(), hiker1, 10*time.Minute)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	clock = clock.Add(30 * time.Second)

	_, err = svc.MintCode(context.Background(), hiker1, 10*time.Minute)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	tok := tokenFor(t, f, hiker1, code1)
	if tok.Attempts != 0 {
		t.Errorf("expected attempt 0 got %d", tok.Attempts)
	}
	if tok.UsedAt == nil {
		t.Error("expected tok used at to be not nil")
	}

}

func Test_ConsumeWrongCode(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)
	hiker := uuid.New()
	svc.otp = func() (string, error) {
		return "123456", nil
	}
	_, err := svc.MintCode(context.Background(), hiker, 10*time.Minute)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	err = svc.ConsumeCode(context.Background(), hiker, "000000")
	wantStatus(t, err, apperr.CodeUnauthorized)

	tok := tokenFor(t, f, hiker, "123456")
	if tok.Attempts != 1 {
		t.Errorf("expected attempt 1 got %d", tok.Attempts)
	}
	if tok.UsedAt != nil {
		t.Error("one wrong try must not burn the code")
	}

}
func tokenFor(t *testing.T, f *fakeStore, hiker uuid.UUID, code string) *Token {
	t.Helper()

	for _, tok := range f.tokens {
		if tok.Hash == hashCode(hiker, code) {

			return tok
		}
	}
	t.Fatalf("no token found")

	return nil
}

func Test_ConsumeWrongCode5Times(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)
	hiker := uuid.New()
	svc.otp = func() (string, error) {
		return "123456", nil
	}
	_, err := svc.MintCode(context.Background(), hiker, 10*time.Minute)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	for range 5 {
		err = svc.ConsumeCode(context.Background(), hiker, "000000")
		wantStatus(t, err, apperr.CodeUnauthorized)
	}
	err = svc.ConsumeCode(context.Background(), hiker, "123456")
	wantStatus(t, err, apperr.CodeUnauthorized)

	tok := tokenFor(t, f, hiker, "123456")
	if tok.Attempts != 5 {
		t.Errorf("expected attempt 5 got %d", tok.Attempts)
	}
	if tok.UsedAt == nil {
		t.Error("expected tok used at to be not nil")
	}

}

func Test_ConsumeWrongCode4Times(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)
	hiker := uuid.New()
	svc.otp = func() (string, error) {
		return "123456", nil
	}
	_, err := svc.MintCode(context.Background(), hiker, 10*time.Minute)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	for range 4 {
		err = svc.ConsumeCode(context.Background(), hiker, "000000")
		wantStatus(t, err, apperr.CodeUnauthorized)
	}
	err = svc.ConsumeCode(context.Background(), hiker, "123456")
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}

	tok := tokenFor(t, f, hiker, "123456")
	if tok.Attempts != 4 {
		t.Errorf("expected attempt 4 got %d", tok.Attempts)
	}
	if tok.UsedAt == nil {
		t.Error("expected tok used at to be not nil")
	}

}

func Test_ConsumeAfter11m(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)
	clock := time.Now()
	svc.now = func() time.Time {
		return clock
	}
	hiker := uuid.New()
	svc.otp = func() (string, error) {
		return "123456", nil
	}
	_, err := svc.MintCode(context.Background(), hiker, 10*time.Minute)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	clock = clock.Add(11 * time.Minute)
	err = svc.ConsumeCode(context.Background(), hiker, "123456")
	wantStatus(t, err, apperr.CodeUnauthorized)
}
func Test_ConsumeTwice(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)
	hiker := uuid.New()
	svc.otp = func() (string, error) {
		return "123456", nil
	}
	_, err := svc.MintCode(context.Background(), hiker, 10*time.Minute)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	err = svc.ConsumeCode(context.Background(), hiker, "123456")
	if err != nil {
		t.Fatalf("expected error got %v", err)
	}
	err = svc.ConsumeCode(context.Background(), hiker, "123456")
	wantStatus(t, err, apperr.CodeUnauthorized)
}

func Test_RequestCodesBackToBack(t *testing.T) {
	f := newFakeStore()
	svc := NewService(f)
	hiker := uuid.New()
	clock := time.Now()
	svc.now = func() time.Time {
		return clock
	}
	_, err := svc.MintCode(context.Background(), hiker, 10*time.Minute)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	clock = clock.Add(25 * time.Second)
	_, err = svc.MintCode(context.Background(), hiker, 10*time.Minute)
	if !errors.Is(err, ErrTooSoon) {
		t.Errorf("expected to get ErrTooSoon got %v", err)
	}
	if len(f.tokens) != 1 {
		t.Errorf("expected 1 token got %v", len(f.tokens))
	}
	clock = clock.Add(31 * time.Second)
	_, err = svc.MintCode(context.Background(), hiker, 10*time.Minute)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if len(f.tokens) != 2 {
		t.Errorf("expected 1 token got %v", len(f.tokens))
	}
}
