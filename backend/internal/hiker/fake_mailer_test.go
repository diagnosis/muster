package hiker

import (
	"context"

	"github.com/diagnosis/go-toolkit/v3/mailer"
)

type sendMail struct {
	to      []string
	subject string
	body    string
}

type fakeMailer struct {
	sent []sendMail
	err  error
}

func (m *fakeMailer) Send(ctx context.Context, to []string, subject, body string) error {
	if m.err != nil {
		return m.err
	}
	m.sent = append(m.sent, sendMail{to: to, subject: subject, body: body})
	return nil
}

var _ mailer.Mailer = (*fakeMailer)(nil)
