package notifier

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"

	"go-upwork-job/internal/store"
)

type Notifier struct {
	host string
	port int
	user string
	pass string
	from string
	to   string
}

func New(host string, port int, user, pass, from, to string) *Notifier {
	return &Notifier{host: host, port: port, user: user, pass: pass, from: from, to: to}
}

func (n *Notifier) SendNewJob(job store.Job) error {
	subject := oneLine("New Upwork Job: " + job.Title)
	body := fmt.Sprintf(
		"Upwork Job ID: %s\n\nTitle: %s\n\nSummary:\n%s\n\nLink:\n%s\n",
		job.UpworkJobID, job.Title, job.Summary, job.Link,
	)
	return n.send(subject, body)
}

func (n *Notifier) send(subject, body string) error {
	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=\"UTF-8\"\r\n\r\n%s",
		n.from, n.to, subject, body,
	)

	addr := fmt.Sprintf("%s:%d", n.host, n.port)
	auth := smtp.PlainAuth("", n.user, n.pass, n.host)

	if n.port == 465 {
		conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: n.host})
		if err != nil {
			return err
		}
		c, err := smtp.NewClient(conn, n.host)
		if err != nil {
			return err
		}
		defer c.Close()

		if err := c.Auth(auth); err != nil {
			return err
		}
		if err := c.Mail(n.from); err != nil {
			return err
		}
		if err := c.Rcpt(n.to); err != nil {
			return err
		}
		w, err := c.Data()
		if err != nil {
			return err
		}
		if _, err := w.Write([]byte(msg)); err != nil {
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}
		return c.Quit()
	}

	return smtp.SendMail(addr, auth, n.from, []string{n.to}, []byte(msg))
}

func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}