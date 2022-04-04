package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
	units "github.com/docker/go-units"
)

var CLI struct {
	SMTPAddr           string `name:"smtp-addr" help:"Address to listen for incoming SMTP on" required:"" env:"SMTP_ADDR"`
	SMTPPrimaryHost    string `name:"smtp-primary-host" help:"Primary host advertised to incoming SMTP connections" required:"" env:"SMTP_PRIMARY_HOST"`
	MaxMailSize        string `name:"max-mail-size" help:"Maximum size of incoming emails including attachments." default:"5m" env:"MAX_MAIL_SIZE"`
	MailExpireDuration string `name:"mail-expire-duration" help:"Duration for how long mail should be kept around after it has been received" default:"10m0s" env:"MAIL_EXPIRE_DURATION"`

	HTTPAddr  string `name:"http-addr" help:"Address to listen on for incoming http requests" required:"" env:"HTTP_ADDR"`
	HTTPToken string `name:"http-token" help:"Token string that is required for authentication, blank for none" default:"" env:"HTTP_TOKEN"`
}

func main() {
	ctx := kong.Parse(&CLI)

	mailExpireDuration, err := time.ParseDuration(CLI.MailExpireDuration)
	ctx.FatalIfErrorf(err, "invalid mail expire duration. expected duration string")
	if mailExpireDuration < time.Duration(0) {
		ctx.Fatalf("invalid mail expire duration. expected positive duration")
	}

	smtpMaxEnvelopeSize, err := units.FromHumanSize(CLI.MaxMailSize)
	ctx.FatalIfErrorf(err, "invalid max mail size")

	ms := NewMessageStore(mailExpireDuration)
	err = ms.Run()
	ctx.FatalIfErrorf(err, "failed starting message store")

	hs := NewHTTPServer(CLI.HTTPAddr, CLI.HTTPToken, ms)
	err = hs.Run()
	ctx.FatalIfErrorf(err, "failed starting HTTP server")

	ss := NewSMTPServer(smtpMaxEnvelopeSize, CLI.SMTPAddr, CLI.SMTPPrimaryHost, ms)
	err = ss.Run()
	ctx.FatalIfErrorf(err, "failed starting SMTP server")

	signalChannel := make(chan os.Signal, 1)

	signal.Notify(signalChannel,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGINT,
	)

	for range signalChannel {
		log.Println("Shutdown signal caught")
		go func() {
			// exit if graceful shutdown not finished in 60 sec.
			time.Sleep(60 * time.Second)
			log.Fatalln("graceful shutdown timed out")
		}()
		ss.Stop()
		hs.Stop()
		ms.Stop()
		log.Println("Shutdown completed, exiting.")
		return
	}
}
