package main

import (
	"bytes"
	"html/template"
	"log"
	"time"

	"github.com/vanng822/go-premailer/premailer"
	mail "github.com/xhit/go-simple-mail/v2"
)

type Mail struct {
	Domain      string
	Host        string
	Port        int
	Username    string
	Password    string
	Encryption  string
	FromAddress string
	FromName    string
}

type Message struct {
	From        string
	FromName    string
	To          string
	Subject     string
	Attachments []string
	Data        any
	DataMap     map[string]any
}

func (m *Mail) SendSMTPMessage(message Message) error {
	if message.From == "" {
		message.From = m.FromAddress
	}
	if message.FromName == "" {
		message.FromName = m.FromName
	}

	data := map[string]any{
		"message": message.Data,
	}
	message.DataMap = data

	formattedMessage, err := m.buildHTMLMessage(message)
	if err != nil {
		return err
	}

	plainMessage, err := m.buildPlainMessage(message)
	if err != nil {
		return err
	}

	server := mail.NewSMTPClient()
	server.Host = m.Host
	server.Port = m.Port
	server.Username = m.Username
	server.Password = m.Password
	server.Encryption = m.getEncryption(m.Encryption)
	server.KeepAlive = false
	server.ConnectTimeout = 10 * time.Second
	server.SendTimeout = 10 * time.Second

	log.Printf("m object: Host %s Port %d\n", m.Host, m.Port)

	smtpClient, err := server.Connect()
	if err != nil {
		log.Println("Error from server Connect: ", err)
		return err
	}

	email := mail.NewMSG()
	email.SetFrom(message.From).
		AddTo(message.To).
		SetSubject(message.Subject).
		SetBody(mail.TextPlain, plainMessage).
		AddAlternative(mail.TextHTML, formattedMessage)

	if len(message.Attachments) > 0 {
		for _, attachment := range message.Attachments {
			email.AddAttachment(attachment)
		}
	}

	err = email.Send(smtpClient)
	if err != nil {
		log.Println("Error email.Send: ", err)
		return err
	}
	return nil
}

func (m *Mail) buildHTMLMessage(message Message) (string, error) {
	templateToRender := "./templates/mail.html.gohtml"

	t, err := template.New("email-html").ParseFiles(templateToRender)
	if err != nil {
		return "", err
	}

	var tpl bytes.Buffer
	if err = t.ExecuteTemplate(&tpl, "body", message.DataMap); err != nil {
		return "", err
	}

	formattedMssage := tpl.String()
	formattedMssage, err = m.inlineCSS(formattedMssage)

	return formattedMssage, nil
}

func (m *Mail) buildPlainMessage(message Message) (string, error) {
	templateToRender := "./templates/mail.plain.gohtml"

	t, err := template.New("email-plain").ParseFiles(templateToRender)
	if err != nil {
		return "", err
	}

	var tpl bytes.Buffer
	if err = t.ExecuteTemplate(&tpl, "body", message.DataMap); err != nil {
		return "", err
	}

	plainMessage := tpl.String()

	return plainMessage, nil
}

func (m *Mail) inlineCSS(html string) (string, error) {
	options := premailer.Options{
		RemoveClasses:     false,
		CssToAttributes:   false,
		KeepBangImportant: true,
	}
	prem, err := premailer.NewPremailerFromString(html, &options)
	if err != nil {
		return "", err
	}
	html, err = prem.Transform()
	if err != nil {
		return "", err
	}
	return html, nil
}

func (m *Mail) getEncryption(encryption string) mail.Encryption {
	switch encryption {
	case "ssl":
		return mail.EncryptionSSLTLS
	case "tls":
		return mail.EncryptionSTARTTLS
	case "none", "":
		return mail.EncryptionNone
	default:
		return mail.EncryptionSTARTTLS
	}
}
