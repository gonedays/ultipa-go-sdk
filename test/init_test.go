package test

import (
	"log"
	"testing"
	"ultipa-go-sdk/sdk"
	"ultipa-go-sdk/sdk/api"
	"ultipa-go-sdk/sdk/configuration"
)

var client *api.UltipaAPI

var hosts []string

func TestMain(m *testing.M) {
	var err error

	hosts = []string{
		//"210.13.32.147:60095",
		"192.168.1.85:61090",
		"192.168.1.87:61090",
		"192.168.1.88:61090",
	}
	client, err = GetClient(hosts, "gosdk")

	if err != nil {
		log.Fatalln(err)
	}

	m.Run()
}

func TestPing(t *testing.T) {
	client, _ = GetClient(hosts, "default")
	client.Test()
}

func GetClient(hosts []string, graphName string) (*api.UltipaAPI, error) {
	var err error
	config := configuration.NewUltipaConfig(&configuration.UltipaConfig{
		Hosts:        hosts,
		Username:     "root",
		Password:     "root",
		DefaultGraph: graphName,
		Debug:        true,
	})

	client, err = sdk.NewUltipa(config)

	if err != nil {
		log.Fatalln(err)
	}

	return client, err
}
