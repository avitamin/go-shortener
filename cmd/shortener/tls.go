package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"strings"
	"time"
)

// launchServer запускает HTTP или HTTPS-сервер.
// При HTTPS используется самоподписанный сертификат, который генерируется при каждом старте.
// Из-за этого браузер будет считать сертификат новым/недоверенным и показывать предупреждение.
func launchServer(server *http.Server, enableHTTPS bool) error {
	if !enableHTTPS {
		return server.ListenAndServe()
	}

	cert, err := generateSelfSignedCertificate(server.Addr)
	if err != nil {
		return fmt.Errorf("generating TLS certificate: %w", err)
	}

	listener, err := tls.Listen("tcp", server.Addr, &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	})
	if err != nil {
		return fmt.Errorf("creating TLS listener: %w", err)
	}

	log.Printf("HTTPS включен для адреса %s\n", server.Addr)
	log.Println("Используется самоподписанный сертификат, сгенерированный при запуске: браузер может показать предупреждение о безопасности")
	return server.Serve(listener)
}

func generateSelfSignedCertificate(addr string) (tls.Certificate, error) {
	hosts := []string{"localhost"}

	host, _, err := net.SplitHostPort(addr)
	if err == nil && host != "" {
		hosts = append(hosts, strings.Trim(host, "[]"))
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, err
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: "go-shortener",
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
			continue
		}

		template.DNSNames = append(template.DNSNames, h)
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})

	return tls.X509KeyPair(certPEM, keyPEM)
}
