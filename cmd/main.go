package main

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/shpeliving/calert/internal/metrics"
	"github.com/shpeliving/calert/internal/notifier"

	"github.com/sirupsen/logrus"
)

var (
	// Version of the build. This is injected at build-time.
	buildString = "unknown"
)

// App is the global contains
// instances of various objects used in the lifecyle of program.
type App struct {
	lo       *logrus.Logger
	metrics  *metrics.Manager
	notifier notifier.Notifier
}

func main() {
	// Initialise logger.
	lo := initLogger()

	// Initialise and load the config.
	ko, err := initConfig(lo, "config.sample.toml", "CALERT_")
	if err != nil {
		// Need to `panic` since logger can only be initialised once config is initialised.
		panic(err.Error())
	}

	var (
		metrics  = initMetrics()
		provs    = initProviders(ko, lo, metrics)
		notifier = initNotifier(ko, lo, provs)
	)

	// Enable debug mode if specified.
	if ko.String("app.log") == "debug" {
		lo.SetLevel(logrus.DebugLevel)
	}

	app := &App{
		lo:       lo,
		notifier: notifier,
		metrics:  metrics,
	}

	app.lo.WithField("version", buildString).Info("booting calert")

	// Initialise HTTP Router.
	r := chi.NewRouter()

	// Add some middlewares.
	r.Use(middleware.RequestID)
	if ko.Bool("app.enable_request_logs") {
		r.Use(middleware.Logger)
	}

	// Register Handlers
	r.Get("/", wrap(app, handleIndex))
	r.Get("/ping", wrap(app, handleHealthCheck))
	r.Get("/metrics", wrap(app, handleMetrics))
	r.Post("/dispatch", wrap(app, handleDispatchNotif))

	// Start HTTP Server.
	app.lo.WithField("addr", ko.MustString("app.address")).Info("starting http server")
	srv := &http.Server{
		Addr:         ko.MustString("app.address"),
		ReadTimeout:  ko.MustDuration("app.server_timeout"),
		WriteTimeout: ko.MustDuration("app.server_timeout"),
		Handler:      r,
	}
	if err := srv.ListenAndServe(); err != nil {
		app.lo.WithError(err).Fatal("couldn't start server")
	}
}
