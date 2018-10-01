package main

import (
	"OVH-Exporter/src/config"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/ovh/go-ovh/ovh"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	promlog "github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	sc = config.SafeConfig{
		C: &config.Config{},
	}

	log promlog.Logger

	configFile    = kingpin.Flag("config.file", "OVH configuration file.").Default("ovh.yml").String()
	listenAddress = kingpin.Flag("web.listen-address", "The address to listen on for HTTP requests.").Default(":9147").String()
	logLevel      = kingpin.Flag("log.level", "Only log messages with the given severity or above. Valid levels: [debug, info, warn, error, fatal]").Default("info").String()
)

func init() {
	prometheus.MustRegister(version.NewCollector("ovh_exporter"))
}

func setOvhClient(o *config.Ovh) (*ovh.Client, error) {
	return ovh.NewClient(o.Endpoint, o.AppKey, o.AppSecret, o.ConsumerKey)
}

func ovhHandler(w http.ResponseWriter, r *http.Request, o *ovh.Client) {
	registry := prometheus.NewRegistry()
	collector := collector{ovhClient: o}
	registry.MustRegister(collector)

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}

func main() {
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	log = promlog.Base()
	if err := log.SetLevel(*logLevel); err != nil {
		log.Fatal("Error: ", err)
	}

	log.Infoln("Starting OVH-Exporter")

	if err := sc.ReloadConfig(*configFile); err != nil {
		log.Fatal("Error loading config: ", err)
		os.Exit(1)
	}
	log.Infoln("Loaded config file")
	sc.Lock()
	conf := sc.C
	sc.Unlock()

	hup := make(chan os.Signal)
	reloadCh := make(chan chan error)
	signal.Notify(hup, syscall.SIGHUP)

	go func() {
		for {
			select {
			case <-hup:
				if err := sc.ReloadConfig(*configFile); err != nil {
					log.Errorln("Error reloading config:", err)
					continue
				}
				log.Infoln("Reloaded config file")
			case rc := <-reloadCh:
				if err := sc.ReloadConfig(*configFile); err != nil {
					log.Errorln("Error reloading config:", err)
					rc <- err
				} else {
					log.Infoln("Reloaded config file")
					rc <- nil
				}
			}
		}
	}()

	ovh, err := setOvhClient(&conf.Ovh)
	if err != nil {
		log.Fatalln(err)
	}

	http.HandleFunc("/-/reload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, "This endpoint requires a POST request.\n")
			return
		}

		rc := make(chan error)
		reloadCh <- rc
		if err := <-rc; err != nil {
			http.Error(w, fmt.Sprintf("Failed to reload config: %s", err), http.StatusInternalServerError)
			return
		}
		tmp, err := setOvhClient(&conf.Ovh)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to reload config: %s", err), http.StatusInternalServerError)
			return
		}
		ovh = tmp
	})

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/ovh", func(w http.ResponseWriter, r *http.Request) {
		ovhHandler(w, r, ovh)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<html>
				<head>
					<title>OVH-Exporter</title>
				</head>
				<body>
					<h1>OVH-Exporter</h1>
					<p><a href="/ovh">OVH Metrics</a></p>
				</body>
			</html>`))
	})

	log.Infoln("Listening on:", *listenAddress)
	if err := http.ListenAndServe(*listenAddress, nil); err != nil {
		log.Fatal("Error: Can't starting HTTP server: ", err)
		os.Exit(1)
	}
}
