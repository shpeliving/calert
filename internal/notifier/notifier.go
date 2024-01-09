package notifier

import (
	"fmt"

	alertmgrtmpl "github.com/prometheus/alertmanager/template"
	"github.com/shpeliving/calert/internal/providers"
	"github.com/sirupsen/logrus"
)

// Notifier represents an instance that pushes out notifications to
// upstream providers.
type Notifier struct {
	providers map[string]providers.Provider
	lo        *logrus.Logger
}

type Opts struct {
	Providers []providers.Provider
	Log       *logrus.Logger
}

// Init initialises a new instance of the Notifier.
func Init(opts Opts) (Notifier, error) {
	// Initialise a map with room as the key and their corresponding provider instance.
	m := make(map[string]providers.Provider, 0)

	for _, prov := range opts.Providers {
		room := prov.Room()
		m[room] = prov
	}

	return Notifier{
		lo:        opts.Log,
		providers: m,
	}, nil
}

// Dispatch pushes out a notification to an upstream provider.
func (n *Notifier) Dispatch(payload alertmgrtmpl.Data, room string) error {
	n.lo.WithField("payload", payload).Debug("dispatch request payload")

	n.lo.WithField("count", len(payload.Alerts)).Info("dispatching alerts")

	// Lookup for the provider by the room name.
	if _, ok := n.providers[room]; !ok {
		n.lo.WithField("room", room).Warn("no provider available for room")
		return fmt.Errorf("no provider configured for room: %s", room)
	}
	// Push the batch of alerts.
	n.providers[room].Push(payload.Alerts)

	return nil
}
