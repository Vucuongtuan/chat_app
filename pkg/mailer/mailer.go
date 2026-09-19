package mailer

import (
	"fmt"
	"log"
	"net/smtp"
	"strings"

	"chatapp/internal/config"
)

type Mailer interface {
	SendOTP(to string, code string, purpose string) error
}

type smtpMailer struct {
	cfg *config.Config
}

func NewMailer(cfg *config.Config) Mailer {
	return &smtpMailer{cfg: cfg}
}

func (m *smtpMailer) SendOTP(to string, code string, purpose string) error {
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
		return nil
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

	err := smtp.SendMail(addr, auth, from, []string{to}, msg)
	if err != nil {
		log.Printf("[MAILER ERROR] Lỗi gửi email tới %s: %v", to, err)
		return err
	}

	return nil
}
