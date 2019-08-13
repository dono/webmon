package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"time"

	"webmon/slack"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Webmon WebmonConfig `toml:"webmon"`
	Slack  SlackConfig  `toml:"slack"`
}

type WebmonConfig struct {
	URL      string `toml:"url"`
	Timeout  int    `toml:"timeout"`
	Interval int    `toml:"interval"`
	isPrint  bool   `toml:"is_print"`
}

type SlackConfig struct {
	WebhookURL  string `toml:"webhook_url"`
	Channel     string `toml:"channel"`
	Username    string `toml:"username"`
	AlertPrefix string `toml:"alert_prefix"`
}

type stats struct {
	ipAddr  string
	tlsCert *x509.Certificate
	time    struct {
		dnsLookup        time.Duration
		tcpConnection    time.Duration
		tlsHandshake     time.Duration
		serverProcessing time.Duration
		contentTransfer  time.Duration
		ttfb             time.Duration
	}
}

var config Config

func main() {
	_, err := toml.DecodeFile(`./config.toml`, &config)
	if err != nil {
		log.Fatal("cannot read config.toml: ", err)
	}

	isUseSlack := false
	if config.Slack.WebhookURL != "" {
		isUseSlack = true
	}

	sc, err := slack.NewSlack(config.Slack.WebhookURL, config.Slack.Channel, config.Slack.Username)
	if err != nil && isUseSlack {
		log.Fatal(err)
	}

	u, err := url.Parse(config.Webmon.URL)
	if err != nil {
		log.Fatal("Invalid URL:", err)
	}

	t := time.NewTicker(time.Duration(config.Webmon.Interval) * time.Second)
	defer t.Stop()

	for ; true; <-t.C {
		date := time.Now().Format("2006-01-02 15:04:05")
		stats, pollErr := poll(config.Webmon.URL)

		fmt.Println(time.Now())
		if pollErr != nil {
			// Print console
			fmt.Printf("Polling error occurred!: %s\n\n", pollErr)

			if isUseSlack {
				// Post error to Slack
				err := sc.Post(
					config.Webmon.URL,
					config.Slack.AlertPrefix,
					fmt.Sprintf("Date: %s\nError: %s", date, pollErr),
					"danger",
				)
				if err != nil {
					log.Fatal(err)
				}
			}
		} else {
			// Print console
			if u.Scheme == "https" {
				stats.certInfo()
			}
			stats.timeInfo()
			fmt.Println()

			if isUseSlack {
				// Post to Slack
				err := sc.Post(
					config.Webmon.URL,
					"",
					fmt.Sprintf("Date: %s\nLatency: %s", date, stats.time.ttfb),
					"good",
				)
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}
}

func poll(url string) (stats, error) {
	var t0, t1, t2, t3, t4, t5, t6, t7 time.Time
	var s stats

	trace := &httptrace.ClientTrace{
		DNSStart: func(_ httptrace.DNSStartInfo) {
			t0 = time.Now()
		},
		DNSDone: func(ddi httptrace.DNSDoneInfo) {
			t1 = time.Now()
		},
		ConnectStart: func(_, _ string) {
			if t1.IsZero() {
				t1 = time.Now()
			}
		},
		ConnectDone: func(_, addr string, _ error) {
			t2 = time.Now()
			s.ipAddr = addr
		},
		TLSHandshakeStart: func() {
			t3 = time.Now()
		},
		TLSHandshakeDone: func(cs tls.ConnectionState, err error) {
			t4 = time.Now()
			if err == nil {
				s.tlsCert = cs.PeerCertificates[0] // End Entity証明書のみ対応
			}
		},
		GotConn: func(_ httptrace.GotConnInfo) {
			t5 = time.Now()
		},
		GotFirstResponseByte: func() {
			t6 = time.Now()
		},
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	client := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true, // be required
		},
		Timeout: time.Duration(config.Webmon.Timeout) * time.Second,
	}

	_, err = client.Do(req)
	if err != nil {
		return s, err
	}
	t7 = time.Now()

	s.time.dnsLookup = t1.Sub(t0)
	s.time.tcpConnection = t2.Sub(t1)
	s.time.tlsHandshake = t4.Sub(t3)
	s.time.serverProcessing = t6.Sub(t5)
	s.time.contentTransfer = t7.Sub(t6)
	s.time.ttfb = t6.Sub(t0)

	return s, nil
}

func (s *stats) timeInfo() {
	fmt.Println("------------- Time ----------------------------")
	fmt.Printf("DNS Lookup:\t    %s\n", s.time.dnsLookup)
	fmt.Printf("TCP Connection:\t    %s\n", s.time.tcpConnection)
	fmt.Printf("TLS Handshake:\t    %s\n", s.time.tlsHandshake)
	fmt.Printf("Server Processing:  %s\n", s.time.serverProcessing)
	fmt.Printf("Content Transfer:   %s\n", s.time.contentTransfer)
	fmt.Printf("Time to first byte: %s\n", s.time.ttfb)
}

func (s *stats) certInfo() {
	fmt.Println("--------- Certification -----------------------")
	fmt.Printf("Issuer:\t%s\n", s.tlsCert.Issuer.CommonName)
	fmt.Printf("CommonName:\t%s\n", s.tlsCert.Subject.CommonName)
	fmt.Printf("SANs:\t\t%s\n", s.tlsCert.DNSNames)
	fmt.Printf("NotBefore:\t%s\n", s.tlsCert.NotBefore)
	fmt.Printf("NotAfter:\t%s\n", s.tlsCert.NotAfter)
}
