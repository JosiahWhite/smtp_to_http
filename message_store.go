package main

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/flashmob/go-guerrilla/mail"
	"github.com/jhillyerd/enmime"
)

type Message struct {
	From    string
	To      []string
	Headers map[string][]string
	Text    string
	HTML    string

	ExpireTime time.Time
}

type MessageStore struct {
	expireInterval time.Duration

	messages      map[string][]*Message
	messagesMutex *sync.RWMutex

	stopRunning bool
}

func NewMessageStore(expireInterval time.Duration) *MessageStore {
	return &MessageStore{
		expireInterval: expireInterval,
		messages:       make(map[string][]*Message),
		messagesMutex:  &sync.RWMutex{},
	}
}

func (ms *MessageStore) Run() error {
	go ms.expireWorker()

	return nil
}

func (ms *MessageStore) Stop() {
	ms.stopRunning = true
}

func (ms *MessageStore) SaveMessage(incoming *mail.Envelope) error {
	ms.messagesMutex.Lock()
	defer ms.messagesMutex.Unlock()

	reader := incoming.NewReader()
	env, err := enmime.ReadEnvelope(reader)
	if err != nil {
		return fmt.Errorf("error occurred during email parsing: %v", err)
	}
	text := env.Text

	for _, part := range env.Inlines {
		if bytes.Equal(part.Content, []byte(env.Text)) {
			continue
		}
		if text == "" && part.ContentType == "text/plain" && part.FileName == "" {
			text = string(part.Content)
			continue
		}
	}

	for _, part := range env.Attachments {
		if bytes.Equal(part.Content, []byte(env.Text)) {
			continue
		}
		if text == "" && part.ContentType == "text/plain" && part.FileName == "" {
			text = string(part.Content)
			continue
		}
	}

	if text == "" {
		text = incoming.Data.String()
	}

	var toAddrs []string
	for _, addr := range incoming.RcptTo {
		toAddrs = append(toAddrs, addr.String())
	}

	headers := make(map[string][]string)
	for _, key := range env.GetHeaderKeys() {
		headers[key] = env.GetHeaderValues(key)
	}

	mailItem := &Message{
		From:    incoming.MailFrom.String(),
		To:      toAddrs,
		Headers: headers,
		Text:    text,
		HTML:    env.HTML,

		ExpireTime: time.Now().Add(ms.expireInterval),
	}

	for _, target := range incoming.RcptTo {
		targetAddress := strings.ToLower(target.String())

		ms.messages[targetAddress] = append(ms.messages[targetAddress], mailItem)
	}

	return nil
}

func (ms *MessageStore) FetchMessages(email string) []*Message {
	ms.messagesMutex.RLock()
	defer ms.messagesMutex.RUnlock()

	return ms.messages[email]
}

func (ms *MessageStore) RemoveMessages(email string) {
	ms.messagesMutex.Lock()
	defer ms.messagesMutex.Unlock()

	delete(ms.messages, email)
}

func (ms *MessageStore) expireWorker() {
	for !ms.stopRunning {
		ms.messagesMutex.Lock()
		for email, messageList := range ms.messages {
			// remove expired items from messageList without allocating a new array
			// in-place deletion taken from here: https://stackoverflow.com/a/20551116

			i := 0
			for _, message := range messageList {
				// if this message is expired, skip it
				if time.Now().After(message.ExpireTime) {
					continue
				}

				// copy and increment index
				messageList[i] = message
				i++
			}

			// if the message list is empty, just delete this entire map entry
			if i == 0 {
				delete(ms.messages, email)
				continue
			}

			// clear dead values
			for j := i; j < len(messageList); j++ {
				messageList[j] = nil
			}

			// update slice reference
			ms.messages[email] = messageList[:i]
		}
		ms.messagesMutex.Unlock()

		// TODO: Make this configurable
		time.Sleep(30 * time.Second)
	}
}
