package main

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"time"

	"github.com/tg123/phabrik/transport"
)

func fabricping(conn net.Conn, tlsconf *tls.Config, interval time.Duration, timeout time.Duration, count int) error {
	log.Printf("starting fabric handshake and send init transport message")
	c, err := transport.Connect(conn, transport.ClientConfig{
		Config: transport.Config{
			TLS: tlsconf,
		},
	})
	if err != nil {
		log.Fatalf("fabric level handshake failed, error: %v", err)
	}
	defer c.Close()
	go c.Wait()

	log.Printf("fabric level handshake success")

	var lasterr error
	for {
		func() {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			d, err := c.Ping(ctx)
			if err != nil {
				log.Printf("fabric heartbeat error: %v", err)
				lasterr = err
				return
			}

			log.Printf("fabric heartbeat response from %v time = %v", conn.RemoteAddr().String(), d)
		}()

		count--
		if count == 0 {
			break
		}

		time.Sleep(interval)
	}

	return lasterr
}
