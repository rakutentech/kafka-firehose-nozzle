package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/Shopify/sarama"
)

func TestTLS(t *testing.T) {
	_, err := NewKafkaProducer(nil, NewStats(), &Config{
		Kafka: Kafka{
			EnableTLS:  true,
			ClientCert: "",
			ClientKey:  "",
		},
	})
	if err == nil || err.Error() != "please specify client_certificate" {
		t.Fatal("expected fail:", err)
	}
	_, err = NewKafkaProducer(nil, NewStats(), &Config{
		Kafka: Kafka{
			EnableTLS:  true,
			ClientCert: "foo",
			ClientKey:  "",
		},
	})
	if err == nil || err.Error() != "please specify private_key" {
		t.Fatal("expected fail:", err)
	}
	_, err = NewKafkaProducer(nil, NewStats(), &Config{
		Kafka: Kafka{
			EnableTLS:  true,
			ClientCert: "foo",
			ClientKey:  "bar",
		},
	})
	if err == nil || err.Error() != "tls: failed to find any PEM data in certificate input" {
		t.Fatal("expected fail:", err)
	}

	cakey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	clientkey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	hostkey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	nvb := time.Now().Add(-1 * time.Hour)
	nva := time.Now().Add(1 * time.Hour)

	caTemplate := &x509.Certificate{
		Subject:      pkix.Name{CommonName: "ca"},
		Issuer:       pkix.Name{CommonName: "ca"},
		SerialNumber: big.NewInt(0),
		NotAfter:     nva,
		NotBefore:    nvb,
		IsCA:         true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	caDer, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &cakey.PublicKey, cakey)
	if err != nil {
		t.Fatal(err)
	}
	caFinalCert, err := x509.ParseCertificate(caDer)
	if err != nil {
		t.Fatal(err)
	}

	hostDer, err := x509.CreateCertificate(rand.Reader, &x509.Certificate{
		Subject:      pkix.Name{CommonName: "host"},
		Issuer:       pkix.Name{CommonName: "ca"},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
		SerialNumber: big.NewInt(0),
		NotAfter:     nva,
		NotBefore:    nvb,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}, caFinalCert, &hostkey.PublicKey, cakey)
	if err != nil {
		t.Fatal(err)
	}

	clientDer, err := x509.CreateCertificate(rand.Reader, &x509.Certificate{
		Subject:      pkix.Name{CommonName: "client"},
		Issuer:       pkix.Name{CommonName: "ca"},
		SerialNumber: big.NewInt(0),
		NotAfter:     nva,
		NotBefore:    nvb,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}, caFinalCert, &clientkey.PublicKey, cakey)
	if err != nil {
		t.Fatal(err)
	}

	clientCertPem := string(pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: clientDer,
	}))
	clientKeyPem := string(pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(clientkey),
	}))

	hostCertPem := string(pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: hostDer,
	}))
	hostKeyPem := string(pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(hostkey),
	}))

	_, err = NewKafkaProducer(nil, NewStats(), &Config{
		Kafka: Kafka{
			EnableTLS:  true,
			ClientCert: clientCertPem,
			ClientKey:  clientKeyPem,
		},
	})
	if err == nil || err.Error() != "brokers are not provided" {
		t.Fatal("expected fail:", err)
	}

	caPem := string(pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caDer,
	}))

	pool := x509.NewCertPool()
	pool.AddCert(caFinalCert)

	// Fail with system CAs
	doListenerTLSTest(t, false, &tls.Config{
		Certificates: []tls.Certificate{tls.Certificate{
			Certificate: [][]byte{hostDer},
			PrivateKey:  hostkey,
		}},
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  pool,
	}, func(addr string) {
		_, err = NewKafkaProducer(nil, NewStats(), &Config{
			Kafka: Kafka{
				EnableTLS:  true,
				ClientCert: clientCertPem,
				ClientKey:  clientKeyPem,
				Brokers:    []string{addr},
				Topic: Topic{
					LogMessage: "foo",
				},
			},
		})
		if err == nil {
			t.Fatal("Should fail as we have the wrong system CA")
		}
	})

	// Fail with no TLS
	doListenerTLSTest(t, false, &tls.Config{
		Certificates: []tls.Certificate{tls.Certificate{
			Certificate: [][]byte{hostDer},
			PrivateKey:  hostkey,
		}},
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  pool,
	}, func(addr string) {
		_, err = NewKafkaProducer(nil, NewStats(), &Config{
			Kafka: Kafka{
				EnableTLS: false,
				Brokers:   []string{addr},
				Topic: Topic{
					LogMessage: "foo",
				},
			},
		})
		if err == nil {
			t.Fatal("Should fail as we have the wrong system CA")
		}
	})

	// Fail with wrong key for cert
	doListenerTLSTest(t, false, &tls.Config{
		Certificates: []tls.Certificate{tls.Certificate{
			Certificate: [][]byte{hostDer},
			PrivateKey:  hostkey,
		}},
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  pool,
	}, func(addr string) {
		_, err = NewKafkaProducer(nil, NewStats(), &Config{
			Kafka: Kafka{
				EnableTLS:  true,
				ClientCert: hostCertPem,
				ClientKey:  hostKeyPem,
				CACerts:    []string{caPem},
				Brokers:    []string{addr},
				Topic: Topic{
					LogMessage: "foo",
				},
			},
		})
		if err == nil {
			t.Fatal("wrong type of cert")
		}
	})

	// Try to actually work
	doListenerTLSTest(t, true, &tls.Config{
		Certificates: []tls.Certificate{tls.Certificate{
			Certificate: [][]byte{hostDer},
			PrivateKey:  hostkey,
		}},
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  pool,
	}, func(addr string) {
		_, err = NewKafkaProducer(nil, NewStats(), &Config{
			Kafka: Kafka{
				EnableTLS:  true,
				ClientCert: clientCertPem,
				ClientKey:  clientKeyPem,
				CACerts:    []string{caPem},
				Brokers:    []string{addr},
				Topic: Topic{
					LogMessage: "foo",
				},
			},
		})
		if err != nil {
			t.Fatal("Expecting to work:", err)
		}
	})
}

func doListenerTLSTest(t *testing.T, willWork bool, tlsConf *tls.Config, f func(addr string)) {
	//sarama.Logger = log.New(os.Stderr, "", log.LstdFlags)

	seedListener, err := tls.Listen("tcp", "127.0.0.1:0", tlsConf)
	if err != nil {
		t.Fatal("cannot open listener", err)
	}

	var childT *testing.T
	if willWork {
		childT = t
	} else {
		childT = &testing.T{} // we want to swallow errors
	}

	seed := sarama.NewMockBrokerListener(childT, int32(0), seedListener)
	defer seed.Close()

	if willWork {
		seed.Returns(new(sarama.MetadataResponse))
	}

	f(seed.Addr())
}
