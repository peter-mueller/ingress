package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	var (
		production bool
		c          Config
		err        error
	)
	flag.BoolVar(&production, "production", false, "enable production mode")
	flag.Parse()

	LoadEnv(&c)

	ingress := NewIngress(c)
	if production {
		err = ingress.ListenAndServeProduction(":443")
	} else {
		addr := ":8080"
		fmt.Println("Starting ingress at ", addr)
		err = http.ListenAndServe(addr, ingress)
	}
	if err != nil {
		log.Fatalln(err)
	}
}

func LoadEnv(c *Config) {
	var err error

	c.Domain = os.Getenv("INGRESS_DOMAIN")
	if c.Domain == "" {
		log.Fatalln("INGRESS_DOMAIN not set")
	}
	mappingsEnv := os.Getenv("INGRESS_HOSTMAPPINGS")
	if mappingsEnv == "" {
		log.Fatalln("INGRESS_HOSTMAPPINGS not set")
	}
	c.Mappings, err = MappingsFromText(mappingsEnv)
	if err != nil {
		log.Fatalln(err)
	}
}