package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/tg123/phabrik/lease"
)

func guessLocalIp() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		default:
			continue
		}

		if ip.IsLoopback() {
			continue
		}

		if ip.IsUnspecified() {
			continue
		}

		if ip.To4() == nil {
			continue
		}

		if !ip.IsPrivate() {
			continue
		}

		return ip.String(), nil
	}

	return "", fmt.Errorf("no ip found")
}

func leaseping(conn net.Conn, tlsconf *tls.Config, interval time.Duration, timeout time.Duration, leaseaddr string, count int) error {
	config := lease.AgentConfig{
		TLS: tlsconf,
	}
	config.SetDefault()

	if strings.ToLower(leaseaddr) == "auto" {
		ip, err := guessLocalIp()
		if err != nil {
			return err
		}
		leaseaddr = net.JoinHostPort(ip, "0")
	}

	l, err := net.Listen("tcp", leaseaddr)
	if err != nil {
		return err
	}

	log.Printf("lease agent listening at [%v]", l.Addr().String())

	agent, err := lease.NewAgent(config, l, func(addr string) (net.Conn, error) {
		return conn, nil
	})

	if err != nil {
		return err
	}
	defer agent.Close()
	go agent.Wait()

	s, err := agent.Establish(conn.RemoteAddr().String())
	if err != nil {
		return err
	}

	log.Printf("starting lease ping")

	var lasterr error

	for {
		func() {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			now := time.Now()
			err := s.Ping(ctx)
			if err != nil {
				log.Printf("lease ping error %v", err)
				lasterr = err
				return
			}

			d := s.LastPongTime().Sub(now)
			log.Printf("lease pong from %v time = %v", conn.RemoteAddr().String(), d)
		}()

		time.Sleep(interval)

		count--
		if count == 0 {
			break
		}
	}

	return lasterr
}
