package hiker

import (
	"context"
	"testing"
	"time"

	"github.com/diagnosis/go-toolkit/v3/apperr"
	"github.com/diagnosis/go-toolkit/v3/secure"
	"github.com/google/uuid"
)

func getTestJWTSigner() (*secure.JWTSigner, error) {
	j, err := secure.NewJWTSigner(secure.JWTConfig{
		AccessSecret:       "test-access-token",
		RefreshSecret:      "test-refresh-token",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
		Issuer:             "muster",
		Audience:           "muster-users",
		Leeway:             0,
	})
	if err != nil {
		return nil, err
	}
	return j, nil
}

func wantStatus(t *testing.T, err error, want apperr.Status) {
	t.Helper()
	se, ok := apperr.AsStatusErr(err)
	if !ok || se.Status != want {
		t.Fatalf("got %v, want status %v", err, want)
	}
}

func Test_Register_Hiker(t *testing.T) {
	f := newFakeStore()
	jwt, err := getTestJWTSigner()
	if err != nil {
		t.Fatalf("expected no error on jwt signer got %v", err)
	}
	svc := NewService(f, jwt)
	hiker, err := svc.Register(context.Background(), RegisterInput{
		Email:      "test@test.com",
		Password:   "Password123",
		Name:       "safa test",
		Experience: ExperienceBeginner,
		HomeArea:   nil,
		Bio:        nil,
		Gender:     nil,
	})
	if err != nil {
		t.Fatalf("expected to register got %v", err)
	}
	if hiker.Name != "safa test" {
		t.Errorf("Expected safa test got %s", hiker.Name)
	}
	if hiker.ID == uuid.Nil {
		t.Error("expected hiker id not nil got nil")
	}
	if hiker.PasswordHash == "Password123" {
		t.Error("expected hashed password stored got not hashed")
	}
}

func Test_Register_ExistingEmail(t *testing.T){
	f := newFakeStore()
	jwt, err := getTestJWTSigner()
	if err != nil {
		t.Fatalf("expected no error on jwt signer got %v", err)
	}
	svc := NewService(f, jwt)
	_, err = svc.Register(context.Background(), RegisterInput{
		Email:      "test@test.com",
		Password:   "Password123",
		Name:       "safa test",
		Experience: ExperienceBeginner,
		HomeArea:   nil,
		Bio:        nil,
		Gender:     nil,
	})
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	_, err = svc.Register(context.Background(), RegisterInput{
		Email:      "test@test.com",
		Password:   "Password123",
		Name:       "safa test2",
		Experience: ExperienceIntermediate,
		HomeArea:   nil,
		Bio:        nil,
		Gender:     nil,
	})
	if err == nil {
		t.Fatal("expected error got nil")
	}
	wantStatus(t, err, apperr.CodeEmailExists)

}

func Test_Register_SendsVerificationEmail(t *testing.T){

}