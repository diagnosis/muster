package email

import (
	"strings"
	"testing"
)

func Test_Render_FullContent(t *testing.T) {
	c := Content{
		Subject:  "Test Subject",
		Heading:  "Verify Your Email",
		Body:     "Click the button to verify.",
		CTALabel: "Verify email",
		CTAURL:   "https://muster.safadev.app/verify-email?token=1bc",
	}
	html, err := Render(c)
	if err != nil {
		t.Fatalf("expected not error got %v", err)
	}
	for _, want := range []string{c.Heading, c.Body, c.CTALabel, c.CTAURL} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered html missing %q", want)
		}
	}
}

func Test_Render_NoCTA(t *testing.T) {
	html, err := Render(Content{Heading: "Password changed", Body: "If this wasn't you..."})
	if err != nil {
		t.Fatalf("expected no error got %v", err)
	}
	if strings.Contains(html, "<a ") {
		t.Error("expected no anchor when CTALabel empty")
	}
}
