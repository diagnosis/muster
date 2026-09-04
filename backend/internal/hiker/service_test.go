package hiker

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/diagnosis/go-toolkit/v3/apperr"
	"github.com/diagnosis/go-toolkit/v3/secure"
	"github.com/diagnosis/muster/internal/authtoken"
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

func newTestService(t *testing.T, fm *fakeMailer) (*Service, *fakeStore, *tokenFakeStore) {
	f := newFakeStore()
	jwt, err := getTestJWTSigner()
	if err != nil {
		t.Fatalf("expected no error on jwt signer got %v", err)
	}
	tf := newTokenFakeStore()
	ts := authtoken.NewService(tf)
	cfg := ServiceConfig{
		Store:     f,
		Token:     ts,
		Mail:      fm,
		JWT:       jwt,
		BaseURL:   "http://test.com",
		VerifyTTL: 24 * time.Hour,
	}
	svc := NewService(cfg)
	return svc, f, tf
}

func getRawFromEmailBody(t *testing.T, fm *fakeMailer) string {
	if len(fm.sent) != 1 {
		t.Fatalf("expected 1 mail got %d", len(fm.sent))
	}
	emailSubject := fm.sent[0].subject
	if emailSubject != "Welcome to Muster - Please verify your email" {
		t.Errorf("expected Welcome to Muster - Please verify your email got %s", emailSubject)
	}
	emailBody := fm.sent[0].body
	var link string
	for _, section := range strings.Split(emailBody, " ") {
		if strings.HasPrefix(section, "http://") || strings.HasPrefix(section, "https://") {
			link = section
		}
	}
	linkURL, err := url.Parse(link)
	if err != nil {
		t.Errorf("expected no parse error got %v", err)
	}
	params, ok := linkURL.Query()["token"]
	if !ok || len(params) == 0 || params[0] == "" {
		t.Errorf("no token in link %q", link)
	}
	raw := params[0]
	return raw
}
func getRawFromEmailBodyString(t *testing.T, body string) string {
	var link string
	for _, section := range strings.Split(body, " ") {
		if strings.HasPrefix(section, "http://") || strings.HasPrefix(section, "https://") {
			link = section
		}
	}
	linkURL, err := url.Parse(link)
	if err != nil {
		t.Errorf("expected no parse error got %v", err)
	}
	params, ok := linkURL.Query()["token"]
	if !ok || len(params) == 0 || params[0] == "" {
		t.Errorf("no token in link %q", link)
	}
	raw := params[0]
	return raw
}
func Test_Register_Hiker(t *testing.T) {
	svc, _, _ := newTestService(t, &fakeMailer{})
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

func Test_Register_ExistingEmail(t *testing.T) {
	svc, _, _ := newTestService(t, &fakeMailer{})
	_, err := svc.Register(context.Background(), RegisterInput{
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

func Test_Register_SendsVerificationEmail(t *testing.T) {

	fm := &fakeMailer{}
	svc, _, _ := newTestService(t, fm)
	_, err := svc.Register(context.Background(), RegisterInput{
		Email:      "test@test.com",
		Password:   "Password123",
		Name:       "safa test2",
		Experience: ExperienceIntermediate,
		HomeArea:   nil,
		Bio:        nil,
		Gender:     nil,
	})
	if err != nil {
		t.Fatalf("expected error got %v", err)
	}
	if len(fm.sent) != 1 {
		t.Fatalf("expected to send 1 mail got %d", len(fm.sent))
	}

}

func Test_Register_ResendDown(t *testing.T) {

	fm := &fakeMailer{err: errors.New("resend down")}
	svc, _, _ := newTestService(t, fm)

	hiker, err := svc.Register(context.Background(), RegisterInput{
		Email:      "test@test.com",
		Password:   "Password123",
		Name:       "safa test2",
		Experience: ExperienceIntermediate,
		HomeArea:   nil,
		Bio:        nil,
		Gender:     nil,
	})
	if err != nil {
		t.Fatalf("expected no error but got %v", err)
	}
	if hiker == nil {
		t.Fatal("expected hiker is created but got nil")
	}
	if len(fm.sent) != 0 {
		t.Errorf("expected nothing is sent but got %d item sent", len(fm.sent))
	}

}

func Test_Register_Recipient(t *testing.T) {

	fm := &fakeMailer{}

	svc, _, _ := newTestService(t, fm)
	_, err := svc.Register(context.Background(), RegisterInput{
		Email:      "test@test.com",
		Password:   "Password123",
		Name:       "safa test2",
		Experience: ExperienceIntermediate,
		HomeArea:   nil,
		Bio:        nil,
		Gender:     nil,
	})
	if err != nil {
		t.Fatalf("expected no error but got %v", err)
	}

	if fm.sent[0].to[0] != "test@test.com" {
		t.Errorf("expected email is sent to test@test.com got %s", fm.sent[0].to[0])
	}

}

func Test_Register_EmailContainsVerificationLink(t *testing.T) {

	fm := &fakeMailer{}

	svc, f, tf := newTestService(t, fm)
	_, err := svc.Register(context.Background(), RegisterInput{
		Email:      "test@test.com",
		Password:   "Password123",
		Name:       "safa test2",
		Experience: ExperienceIntermediate,
		HomeArea:   nil,
		Bio:        nil,
		Gender:     nil,
	})
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	raw := getRawFromEmailBody(t, fm)

	hash := secure.HashRefreshToken(raw)

	tok, err := tf.GetByHash(context.Background(), hash)
	if err != nil {
		t.Errorf("expected no error on getting token by hash got %v", err)
	}
	hiker, err := f.GetHikerByEmail(context.Background(), "test@test.com")
	if err != nil {
		t.Fatalf("expected no error on getting hiker got %v", err)
	}
	if tok.HikerID != hiker.ID {
		t.Errorf("expected %q got %q", hiker.ID, tok.HikerID)
	}
	if tok.Purpose != authtoken.PurposeEmailVerification {
		t.Errorf("expected %s got %s", authtoken.PurposeEmailVerification, tok.Purpose)
	}

}

func Test_Register_MintFails(t *testing.T) {
	f := newFakeStore()
	jwt, err := getTestJWTSigner()
	if err != nil {
		t.Fatalf("expected no error on jwt signer got %v", err)
	}
	tf := &tokenFakeStore{
		tokens:  make(map[string]*authtoken.Token),
		saveErr: errors.New("minting fail test"),
	}
	ts := authtoken.NewService(tf)
	fm := &fakeMailer{}
	cfg := ServiceConfig{
		Store:     f,
		Token:     ts,
		Mail:      fm,
		JWT:       jwt,
		BaseURL:   "http://test.com",
		VerifyTTL: 24 * time.Hour,
	}
	svc := NewService(cfg)
	h, err := svc.Register(context.Background(), RegisterInput{
		Email:      "test@test.com",
		Password:   "Password123",
		Name:       "safa test2",
		Experience: ExperienceIntermediate,
		HomeArea:   nil,
		Bio:        nil,
		Gender:     nil,
	})
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if h == nil {
		t.Error("expected hiker created")
	}
	if len(fm.sent) != 0 {
		t.Errorf("expected no mail sent got %d", len(fm.sent))
	}

}

func Test_Register_VerifyEmail(t *testing.T) {
	fm := &fakeMailer{}
	svc, f, _ := newTestService(t, fm)
	_, err := svc.Register(context.Background(), RegisterInput{
		Email:      "test@test.com",
		Password:   "Password123",
		Name:       "safa test2",
		Experience: ExperienceIntermediate,
		HomeArea:   nil,
		Bio:        nil,
		Gender:     nil,
	})
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	raw := getRawFromEmailBody(t, fm)

	err = svc.VerifyEmail(context.Background(), raw)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	h, err := f.GetHikerByEmail(context.Background(), "test@test.com")
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if h.VerifiedAt == nil {
		t.Error("expected verified_at set")
	}
	err = svc.VerifyEmail(context.Background(), raw)
	wantStatus(t, err, apperr.CodeConflict)

}

func Test_ResendEmailVerification(t *testing.T) {
	fm := &fakeMailer{}
	svc, _, tf := newTestService(t, fm)
	h, err := svc.Register(context.Background(), RegisterInput{
		Email:      "test@test.com",
		Password:   "Password123",
		Name:       "safa test2",
		Experience: ExperienceIntermediate,
		HomeArea:   nil,
		Bio:        nil,
		Gender:     nil,
	})
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	err = svc.ResendEmailVerification(context.Background(), h.Email)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if len(fm.sent) != 2 {
		t.Errorf("expected 2 mails got %d", len(fm.sent))
	}
	body := fm.sent[1].body
	raw := getRawFromEmailBodyString(t, body)
	token, err := tf.GetByHash(context.Background(), secure.HashRefreshToken(raw))
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if token.HikerID != h.ID {
		t.Errorf("expected %s got %s", h.ID, token.HikerID)
	}

}

func Test_ResendEmailVerification_UnknownEmail(t *testing.T) {
	fm := &fakeMailer{}
	svc, _, _ := newTestService(t, fm)
	err := svc.ResendEmailVerification(context.Background(), "zafer@deliburun.com")
	if err != nil {
		t.Fatalf("expected no error but got %v", err)
	}
	if len(fm.sent) != 0 {
		t.Errorf("expected 0 but got %d", len(fm.sent))
	}
}

func Test_ResendEmailVerification_Verified(t *testing.T) {
	fm := &fakeMailer{}
	svc, _, _ := newTestService(t, fm)
	h, err := svc.Register(context.Background(), RegisterInput{
		Email:      "test@test.com",
		Password:   "Password123",
		Name:       "safa test2",
		Experience: ExperienceIntermediate,
		HomeArea:   nil,
		Bio:        nil,
		Gender:     nil,
	})
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	raw := getRawFromEmailBody(t, fm)
	err = svc.VerifyEmail(context.Background(), raw)
	if err != nil {
		t.Fatalf("expected no error got error %v", err)
	}
	err = svc.ResendEmailVerification(context.Background(), h.Email)
	if err != nil {
		t.Fatalf("expected no error but got %v", err)
	}
	if len(fm.sent) != 1 {
		t.Errorf("expected 1 but got %d", len(fm.sent))
	}
}

func TestService_ResendEmailVerification_EmailServiceError(t *testing.T) {
	fm := &fakeMailer{err: apperr.Internal("something internal happened", "internal error")}
	svc, _, _ := newTestService(t, fm)
	h, err := svc.Register(context.Background(), RegisterInput{
		Email:      "test@test.com",
		Password:   "Password123",
		Name:       "safa test2",
		Experience: ExperienceIntermediate,
		HomeArea:   nil,
		Bio:        nil,
		Gender:     nil,
	})
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}

	err = svc.ResendEmailVerification(context.Background(), h.Email)
	if err == nil {
		t.Fatalf("expected error got no error")
	}
	if len(fm.sent) != 0 {
		t.Fatalf("expected 0 got %d", len(fm.sent))
	}
	wantStatus(t, err, apperr.CodeInternalError)
}

func Test_Login_EmailNotVerified(t *testing.T) {
	fm := &fakeMailer{}
	svc, _, _ := newTestService(t, fm)
	h, err := svc.Register(context.Background(), RegisterInput{
		Email:      "test@test.com",
		Password:   "Password123",
		Name:       "safa test2",
		Experience: ExperienceIntermediate,
		HomeArea:   nil,
		Bio:        nil,
		Gender:     nil,
	})
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if _, err = svc.Login(context.Background(), h.Email, "Password123", PlatformWeb); err == nil {
		t.Fatalf("expected error got no error")
	}
	wantStatus(t, err, apperr.CodeAccountInactive)
}

func Test_Login_EmailVerified(t *testing.T) {
	fm := &fakeMailer{}
	svc, _, _ := newTestService(t, fm)
	h, err := svc.Register(context.Background(), RegisterInput{
		Email:      "test@test.com",
		Password:   "Password123",
		Name:       "safa test2",
		Experience: ExperienceIntermediate,
		HomeArea:   nil,
		Bio:        nil,
		Gender:     nil,
	})
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if len(fm.sent) == 0 {
		t.Fatal("expected 1 got 0 mail")
	}
	raw := getRawFromEmailBodyString(t, fm.sent[0].body)
	err = svc.VerifyEmail(context.Background(), raw)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	sess, err := svc.Login(context.Background(), h.Email, "Password123", PlatformWeb)
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if sess.RefreshToken == "" {
		t.Error("expected refresh token is not empty got empty")
	}
	if sess.AccessToken == "" {
		t.Error("expected access token is not empty got empty")
	}

}
