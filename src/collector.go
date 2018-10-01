package main

import (
	"github.com/ovh/go-ovh/ovh"
	"github.com/prometheus/client_golang/prometheus"
)

type collector struct {
	ovhClient *ovh.Client
}

type smsInfo struct {
	CreditLeft float64 `json:"creditsLeft"`
}

func getSMS(client *ovh.Client) map[string]float64 {
	var smsInfo smsInfo
	credits := make(map[string]float64)
	var accounts []string
	err := client.Get("/sms", &accounts)
	if err != nil {
		log.Errorln("Error: loading sms accounts: ", err)
		return map[string]float64{}
	}
	log.Debugln("Get SMS accounts: ", accounts)

	for _, account := range accounts {
		err = client.Get("/sms/"+account, &smsInfo)
		if err != nil {
			log.Errorln("Error: get sms info for account "+account+": ", err)
		} else {
			log.Debugln("Account: "+account+" ->", smsInfo.CreditLeft)
			credits[account] = smsInfo.CreditLeft
		}
	}

	return credits
}

func (c collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- prometheus.NewDesc("dummy", "dummy", nil, nil)
}

func (c collector) Collect(ch chan<- prometheus.Metric) {

	credits := getSMS(c.ovhClient)
	for account, credit := range credits {
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc("sms_credits_left", "Tickets Channel Statistics", []string{"account"}, nil),
			prometheus.GaugeValue,
			credit,
			account)
	}
}
