//===----------- dcrontab - distributed crontab written in go  -------------===
//
//  Copyright (c) 2018 Andrew Grosser. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/nats-io/nats.go"
)

type NatsService struct { //Implements 'session'
	Configuration *Service
	nc            *nats.Conn
	ec            *nats.EncodedConn
	AppConfig     *Configuration
}


////////////////////////////////////////
// Interface Implementations
////////////////////////////////////////

//////////////////////////////////////// NATS
// Connect initiates the primary connection to the range of provided URLs
func (i *NatsService) connect() error {
	err := fmt.Errorf("Could not connect to NATS")

	certFile := i.Configuration.Cert
	keyFile := i.Configuration.Key
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatalf("[ERROR] Parsing X509 certificate/key pair: %v", err)
	}

	rootPEM, err := ioutil.ReadFile(i.Configuration.CACert)

	pool := x509.NewCertPool()
	ok := pool.AppendCertsFromPEM([]byte(rootPEM))
	if !ok {
		log.Fatalln("[ERROR] Failed to parse root certificate.")
	}

	config := &tls.Config{
		//ServerName:         i.Configuration.Hosts[0],
		Certificates:       []tls.Certificate{cert},
		RootCAs:            pool,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: i.Configuration.Secure, //TODO: SECURITY THREAT
	}

	if i.nc, err = nats.Connect(strings.Join(i.Configuration.Hosts[:], ","), nats.Secure(config)); err != nil {
		fmt.Println("[ERROR] Connecting to NATS:", err)
		return err
	}
	if i.ec, err = nats.NewEncodedConn(i.nc, nats.JSON_ENCODER); err != nil {
		fmt.Println("[ERROR] Encoding NATS:", err)
		return err
	}

	return nil
}

//////////////////////////////////////// NATS
// Close
//will terminate the session to the backend, returning error if an issue arises
func (i *NatsService) close() error {
	i.ec.Drain()
	i.ec.Close()
	i.nc.Drain()
	i.nc.Close()
	return nil
}


//////////////////////////////////////// NATS
// Write
func (i *NatsService) publish(channel string, msg string) error {
	// sendCh := make(chan *map[string]interface{})
	// i.ec.Publish(i.Configuration.Context, w.Values)
	// i.ec.BindSendChan(i.Configuration.Context, sendCh)
	// sendCh <- w.Values
	return i.nc.Publish(channel, []byte(msg))
}