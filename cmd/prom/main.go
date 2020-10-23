package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/config"
	config2 "github.com/prometheus/prometheus/discovery/config"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"gopkg.in/yaml.v2"
)

func main() {
	version := flag.Bool("version", false, "print version")
	flag.Parse()

	if *version {
		fmt.Println("Version:", "v0.0.1")
		return
	}

	defCfg := config.Config{
		GlobalConfig: config.GlobalConfig{EvaluationInterval: model.Duration(1 * time.Second)},
	}

	for i, ip := range flag.Args() {
		defCfg.ScrapeConfigs = append(defCfg.ScrapeConfigs, &config.ScrapeConfig{
			JobName:         fmt.Sprintf("fx-chain-%s-%d", ip, i),
			HonorTimestamps: true,
			ScrapeInterval:  model.Duration(1 * time.Second),
			ScrapeTimeout:   model.Duration(1 * time.Second),
			MetricsPath:     "/metrics",
			Scheme:          "http",
			ServiceDiscoveryConfig: config2.ServiceDiscoveryConfig{
				StaticConfigs: []*targetgroup.Group{
					{
						Targets: []model.LabelSet{
							{
								model.AddressLabel: model.LabelValue(fmt.Sprintf("%s:26660", ip)),
							},
						},
					},
				},
			},
		})
	}
	if out, err := yaml.Marshal(defCfg); err != nil {
		println(err.Error())
		os.Exit(1)
	} else {
		if err := ioutil.WriteFile("/prometheus/prometheus.yml", out, os.ModePerm); err != nil {
			println(err.Error())
			os.Exit(1)
		}
	}
}
