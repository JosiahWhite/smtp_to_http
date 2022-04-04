package main

import (
	"fmt"

	"github.com/flashmob/go-guerrilla"
	"github.com/flashmob/go-guerrilla/backends"
	"github.com/flashmob/go-guerrilla/log"
	"github.com/flashmob/go-guerrilla/mail"
)

type SMTPServer struct {
	envelopeSizeBytes    int64
	listenAddress        string
	advertisedDomainName string
	messageStore         *MessageStore

	daemon     *guerrilla.Daemon
	smtpLogger log.Logger
}

func NewSMTPServer(envelopeSizeBytes int64, listenAddress, advertisedDomainName string, messageStore *MessageStore) *SMTPServer {
	return &SMTPServer{
		envelopeSizeBytes:    envelopeSizeBytes,
		listenAddress:        listenAddress,
		advertisedDomainName: advertisedDomainName,
		messageStore:         messageStore,
	}
}

func (smtp *SMTPServer) Run() error {
	cfg := &guerrilla.AppConfig{LogFile: log.OutputStdout.String()}
	cfg.AllowedHosts = []string{"."}

	sc := guerrilla.ServerConfig{
		IsEnabled:       true,
		ListenInterface: smtp.listenAddress,
		MaxSize:         smtp.envelopeSizeBytes,
	}
	cfg.Servers = append(cfg.Servers, sc)

	bcfg := backends.BackendConfig{
		"save_workers_size":  3,
		"save_process":       "HeadersParser|Header|Hasher|MessageStore",
		"log_received_mails": true,
		"primary_mail_host":  smtp.advertisedDomainName,
	}
	cfg.BackendConfig = bcfg

	smtp.daemon = &guerrilla.Daemon{Config: cfg}
	smtp.daemon.AddProcessor("MessageStore", smtp.messageStoreProcessorFactory())

	smtp.smtpLogger = smtp.daemon.Log()

	err := smtp.daemon.Start()
	return err
}

func (smtp *SMTPServer) Stop() {
	smtp.daemon.Shutdown()
}

func (smtp *SMTPServer) messageStoreProcessorFactory() func() backends.Decorator {
	return func() backends.Decorator {
		// https://github.com/flashmob/go-guerrilla/wiki/Backends,-configuring-and-extending

		return func(p backends.Processor) backends.Processor {
			return backends.ProcessWith(
				func(e *mail.Envelope, task backends.SelectTask) (backends.Result, error) {
					if task == backends.TaskSaveMail {
						err := smtp.messageStore.SaveMessage(e)
						if err != nil {
							return backends.NewResult(fmt.Sprintf("554 Error: %s", err)), err
						}
						return p.Process(e, task)
					}
					return p.Process(e, task)
				},
			)
		}
	}
}
