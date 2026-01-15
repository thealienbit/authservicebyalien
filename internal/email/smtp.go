package email

import "gopkg.in/gomail.v2"

type SMTPSender struct {
	Host string
	Port int
	Username string
	Password string
	From string
}

func NewSMTPSender(
	host string,
	port int,
	username, password, from string,
) *SMTPSender {
	return &SMTPSender{
		Host: host,
		Port: port,
		Username: username,
		Password: password,
		From: from,
	}
}

func (s *SMTPSender) Send(to, subject, body string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", s.From)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)

	d := gomail.NewDialer(
		s.Host,
		s.Port,
		s.Username,
		s.Password,
	)

	return d.DialAndSend(m)
}
