package mailer

import (
	"net/smtp"
	"testing"
	"time"

	"chatapp/internal/config"
)

func TestSendOTPReturnsWithoutBlockingOnSMTP(t *testing.T) {
	started := make(chan struct{})
	done := make(chan struct{})

	m := &smtpMailer{
		cfg:   &config.Config{SMTPHost: "smtp.test.local", SMTPPort: "587"},
		queue: make(chan emailJob, 1),
		sender: func(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
			close(started)
			<-done
			return nil
		},
	}
	go m.run()

	go func() {
		<-started
		close(done)
	}()

	start := time.Now()
	if err := m.SendOTP("user@example.com", "123456", "2fa_login"); err != nil {
		t.Fatalf("SendOTP returned error: %v", err)
	}

	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Fatalf("SendOTP blocked for too long: %v", elapsed)
	}
}
