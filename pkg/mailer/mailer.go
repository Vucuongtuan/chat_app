package mailer

import (
	"fmt"
	"log"
	"net/smtp"
	"strings"
	"sync"

	"chatapp/internal/config"
)

type Mailer interface {
	SendOTP(to string, code string, purpose string) error
}

type emailJob struct {
	to      string
	code    string
	purpose string
}

type smtpSendFunc func(addr string, a smtp.Auth, from string, to []string, msg []byte) error

type smtpMailer struct {
	cfg    *config.Config
	queue  chan emailJob
	sender smtpSendFunc
	once   sync.Once
}

func NewMailer(cfg *config.Config) Mailer {
	m := &smtpMailer{
		cfg:   cfg,
		queue: make(chan emailJob, 100),
	}
	m.sender = smtpSendMail
	m.once.Do(func() { go m.run() })
	return m
}

func smtpSendMail(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
	return smtp.SendMail(addr, a, from, to, msg)
}

func (m *smtpMailer) run() {
	for job := range m.queue {
		m.sendOTPAsync(job.to, job.code, job.purpose)
	}
}

func (m *smtpMailer) SendOTP(to string, code string, purpose string) error {
	if m == nil || m.cfg == nil {
		return nil
	}
	if strings.TrimSpace(to) == "" {
		return nil
	}

	if m.queue == nil {
		m.queue = make(chan emailJob, 100)
	}
	if m.sender == nil {
		m.sender = smtpSendMail
	}

	select {
	case m.queue <- emailJob{to: to, code: code, purpose: purpose}:
		return nil
	default:
		go m.sendOTPAsync(to, code, purpose)
		return nil
	}
}

func (m *smtpMailer) sendOTPAsync(to string, code string, purpose string) {
	if m == nil || m.cfg == nil {
		return
	}

	subject := "Mã xác thực tài khoản ChatApp"
	switch purpose {
	case "2fa_login":
		subject = "[ChatApp] Mã xác thực đăng nhập 2 bước"
	case "2fa_enable":
		subject = "[ChatApp] Mã xác thực kích hoạt bảo mật 2 bước"
	case "forgot_password":
		subject = "[ChatApp] Yêu cầu đặt lại mật khẩu"
	}

	body := fmt.Sprintf(
		"Xin chào,\n\nBạn vừa thực hiện yêu cầu: %s.\nMã xác thực OTP của bạn là: %s\nMã này có hiệu lực trong 5 phút. Vui lòng không chia sẻ mã này cho bất kỳ ai.\n\nTrân trọng,\nĐội ngũ ChatApp",
		subject, code,
	)

	log.Printf("[MAILER DEBUG] Gửi mã OTP [%s] tới email [%s] - Mục đích: %s", code, to, purpose)

	if strings.TrimSpace(m.cfg.SMTPHost) == "" {
		return
	}

	from := m.cfg.SMTPFrom
	if from == "" {
		from = "no-reply@chatapp.com"
	}

	msg := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		from, to, subject, body,
	))

	addr := fmt.Sprintf("%s:%s", m.cfg.SMTPHost, m.cfg.SMTPPort)
	var auth smtp.Auth
	if m.cfg.SMTPUser != "" && m.cfg.SMTPPassword != "" {
		auth = smtp.PlainAuth("", m.cfg.SMTPUser, m.cfg.SMTPPassword, m.cfg.SMTPHost)
	}

	if m.sender == nil {
		m.sender = smtpSendMail
	}

	err := m.sender(addr, auth, from, []string{to}, msg)
	if err != nil {
		log.Printf("[MAILER ERROR] Lỗi gửi email tới %s: %v", to, err)
	}
}
