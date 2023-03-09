package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/crypto/acme/autocert"
)

type HostMapping struct {
	Host       string
	TargetPort int
}

func (m *HostMapping) UnmarshalText(text []byte) error {
	host, targetPort, found := strings.Cut(string(text), " ")
	if !found {
		return fmt.Errorf("%s is not a hostmapping text", text)
	}
	port, err := strconv.Atoi(targetPort)
	if err != nil {
		return fmt.Errorf("%s is not a hostmapping text: %s", text, err)
	}
	m.Host = host
	m.TargetPort = port
	return nil
}

func MappingsFromText(text string) (ms []HostMapping, err error) {
	mappings := strings.Split(text, ",")
	ms = make([]HostMapping, len(mappings))
	for i, text := range mappings {
		var m HostMapping
		err := m.UnmarshalText([]byte(text))
		if err != nil {
			return ms, err
		}
		ms[i] = m
	}
	return ms, nil
}

type ProxyByHost map[string]*httputil.ReverseProxy

type Ingress struct {
	Domain            string
	HostMappings      []HostMapping
	HostReversProxies ProxyByHost
}

type Config struct {
	Domain   string
	Mappings []HostMapping
}

func NewIngress(c Config) *Ingress {
	ingress := &Ingress{
		Domain:            c.Domain,
		HostReversProxies: make(ProxyByHost),
		HostMappings:      c.Mappings,
	}
	for _, m := range c.Mappings {
		target := &url.URL{
			Scheme: "http",
			Host:   fmt.Sprintf("localhost:%d", m.TargetPort),
		}
		proxy := httputil.NewSingleHostReverseProxy(target)
		ingress.HostReversProxies[m.Host] = proxy
	}
	return ingress
}

func (i *Ingress) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	proxy, ok := i.HostReversProxies[host]
	if !ok {
		msg := fmt.Sprintf("Forbidden Host: %s", host)
		http.Error(w, msg, 403)
		return
	}
	proxy.ServeHTTP(w, r)
}

func (i *Ingress) ListenAndServeProduction(addr string) error {
	if i.Domain == "" {
		return errors.New("domain not configured")
	}
	fmt.Println("autocert domain:", i.Domain)

	hosts := make([]string, len(i.HostMappings)+1)
	hosts[0] = i.Domain
	for i, m := range i.HostMappings {
		hosts[1+i] = m.Host
	}

	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(hosts...),
		Email:      "peter7995@gmail.com",
		Cache:      autocert.DirCache("certs"),
	}

	tlsConfig := certManager.TLSConfig()
	server := http.Server{
		Addr:      addr,
		Handler:   i,
		TLSConfig: tlsConfig,
	}

	fmt.Println("certManager listening on :80")
	go func() {
		err := http.ListenAndServe(":80", certManager.HTTPHandler(nil))
		if err != nil {
			log.Fatalln(err)
		}
	}()
	return server.ListenAndServeTLS("", "")
}