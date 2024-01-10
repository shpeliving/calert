package google_chat

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	alertmgrtmpl "github.com/prometheus/alertmanager/template"
	"github.com/shpeliving/calert/internal/metrics"
	"github.com/sirupsen/logrus"
)

type GoogleChatManager struct {
	lo           *logrus.Logger
	metrics      *metrics.Manager
	activeAlerts *ActiveAlerts
	endpoint     string
	room         string
	client       *http.Client
	msgTmpl      *template.Template
	dryRun       bool
	v2           bool
}

type GoogleChatOpts struct {
	Log         *logrus.Logger
	Metrics     *metrics.Manager
	DryRun      bool
	MaxIdleConn int
	Timeout     time.Duration
	ProxyURL    string
	Endpoint    string
	Room        string
	Template    string
	ThreadTTL   time.Duration
	V2          bool
}

// NewGoogleChat initializes a Google Chat provider object.
func NewGoogleChat(opts GoogleChatOpts) (*GoogleChatManager, error) {
	transport := &http.Transport{
		MaxIdleConnsPerHost: opts.MaxIdleConn,
	}

	// Add a proxy to make upstream requests if specified in config.
	if opts.ProxyURL != "" {
		u, err := url.Parse(opts.ProxyURL)
		if err != nil {
			return nil, fmt.Errorf("error parsing proxy URL: %s", err)
		}
		transport.Proxy = http.ProxyURL(u)
	}

	// Initialise a generic HTTP Client for communicating with the G-Chat APIs.
	client := &http.Client{
		Timeout:   opts.Timeout,
		Transport: transport,
	}

	// Initialise the map of active alerts.
	alerts := make(map[string]AlertDetails, 0)

	// Initialise message template functions.
	templateFuncMap := template.FuncMap{
		"Title":      strings.Title,
		"toUpper":    strings.ToUpper,
		"Contains":   strings.Contains,
		"escapeJSON": escapeJSON,
		"Text":       replaceNewLines,
		"isEmpty":    isEmpty,
	}

	// Load the template.
	tmpl, err := template.New(filepath.Base(opts.Template)).Funcs(templateFuncMap).ParseFiles(opts.Template)
	if err != nil {
		return nil, err
	}

	mgr := &GoogleChatManager{
		lo:       opts.Log,
		metrics:  opts.Metrics,
		client:   client,
		endpoint: opts.Endpoint,
		room:     opts.Room,
		activeAlerts: &ActiveAlerts{
			alerts:  alerts,
			lo:      opts.Log,
			metrics: opts.Metrics,
		},
		msgTmpl: tmpl,
		dryRun:  opts.DryRun,
		v2:      opts.V2,
	}
	// Start a background worker to cleanup alerts based on TTL mechanism.
	go mgr.activeAlerts.startPruneWorker(1*time.Hour, opts.ThreadTTL)

	return mgr, nil
}

// Push accepts the list of alerts and dispatches them to Webhook API endpoint.
func (m *GoogleChatManager) Push(alerts []alertmgrtmpl.Alert) error {
	m.lo.WithField("count", len(alerts)).Info("dispatching alerts to google chat")

	// For each alert, lookup the UUID and send the alert.
	for _, a := range alerts {
		// If it's a new alert whose fingerprint isn't in the active alerts map, add it first.
		if m.activeAlerts.loookup(a.Fingerprint) == "" {
			m.activeAlerts.add(a)
		}

		threadKey := m.activeAlerts.alerts[a.Fingerprint].UUID.String()

		// Prepare a list of messages to send.
		var msgs []ChatMessage
		var err error
		if m.v2 {
			msgs, err = m.prepareMessageV2(a, threadKey)
		} else {
			msgs, err = m.prepareMessage(a)
		}

		if err != nil {
			m.lo.WithError(err).Error("error preparing message")
			continue
		}

		// Dispatch an HTTP request for each message.
		for _, msg := range msgs {
			now := time.Now()

			m.metrics.Increment(fmt.Sprintf(`alerts_dispatched_total{provider="%s", room="%s"}`, m.ID(), m.Room()))

			// Send message to API.
			if m.dryRun {
				m.lo.WithField("room", m.Room()).Info("dry_run is enabled for this room. skipping pushing notification")
			} else {
				var sendErr error
				if m.v2 {
					sendErr = m.sendMessageV2(msg)
				} else {
					sendErr = m.sendMessage(msg, threadKey)
				}
				if sendErr != nil {
					m.metrics.Increment(fmt.Sprintf(`alerts_dispatched_errors_total{provider="%s", room="%s"}`, m.ID(), m.Room()))
					m.lo.WithError(sendErr).Error("error sending message")
					continue
				}
			}

			m.metrics.Duration(fmt.Sprintf(`alerts_dispatched_duration_seconds{provider="%s", room="%s"}`, m.ID(), m.Room()), now)
		}
	}

	return nil
}

// Room returns the name of room for which this provider is configured.
func (m *GoogleChatManager) Room() string {
	return m.room
}

// ID returns the provider name.
func (m *GoogleChatManager) ID() string {
	return "google_chat"
}

func escapeJSON(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return ""
	}
	// Trim the leading and trailing quotes added by json.Marshal
	return string(b[1 : len(b)-1])
}

func replaceNewLines(s string) string {
	return strings.ReplaceAll(s, "\n", "\\n")
}

func isEmpty(input interface{}) bool {
	if str, ok := input.(string); ok {
		if len(str) > 0 {
			return false
		}
	}

	return true
}
