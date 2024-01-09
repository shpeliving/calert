package google_chat

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"

	alertmgrtmpl "github.com/prometheus/alertmanager/template"
)

// prepareMessage prepares a v2 message to be sent to google chat
func (m *GoogleChatManager) prepareMessageV2(alert alertmgrtmpl.Alert, threadKey string, isThreadReply bool) ([]ChatMessage, error) {
	var (
		to  bytes.Buffer
		msg *ComplexChatMessage
	)

	messages := make([]ChatMessage, 0)

	// Render a template with alert data.
	err := m.msgTmpl.Execute(&to, alert)
	if err != nil {
		m.lo.WithError(err).Error("Error parsing values in v2 template")
		return messages, err
	}

	// Unmarshal the json to ComplexChatMessage struct
	err = json.Unmarshal(to.Bytes(), &msg)
	if err != nil {
		m.lo.WithError(err).Error("Error unmarshalling json in v2 template")
		return messages, err
	}

	// Add thread key to the struct
	msg.Thread = Thread{
		ThreadKey: threadKey,
	}

	// Add whether this is a thread reply or not
	msg.ThreadReply = isThreadReply

	// Add the message to batch.
	messages = append(messages, msg)

	return messages, nil
}

// sendMessage pushes out a v2 alert to a Google Chat space.
func (m *GoogleChatManager) sendMessageV2(msg ChatMessage) error {
	buffer, err := msg.ToBuffer()
	if err != nil {
		return err
	}

	// Prepare the request.
	req, err := http.NewRequest("POST", m.endpoint, buffer)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	// Send the request.
	m.lo.WithField("url", m.endpoint).WithField("msg", msg).Debug("sending v2 alert")
	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// If response is non 200, log and throw the error.
	if resp.StatusCode != http.StatusOK {
		m.lo.WithField("status", resp.StatusCode).Error("Non OK HTTP Response received from Google Chat Webhook endpoint")
		return errors.New("non ok response from gchat")
	}

	return nil
}
