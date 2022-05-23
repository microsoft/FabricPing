package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tg123/phabrik/federation"
	"github.com/tg123/phabrik/lease"
	"github.com/tg123/phabrik/transport"
	"github.com/urfave/cli/v2"
)

func discover(conn net.Conn, tlsconf *tls.Config, fabricaddr string, c *cli.Context) error {
	if strings.ToLower(fabricaddr) == "auto" {
		ip, err := guessLocalIp()
		if err != nil {
			return err
		}
		fabricaddr = net.JoinHostPort(ip, "0")
	}

	log.Printf("starting fabric handshake and send init transport message")
	_, err := transport.Connect(conn, transport.ClientConfig{
		Config: transport.Config{
			TLS: tlsconf,
		},
	})
	if err != nil {
		log.Fatalf("fabric level handshake failed, error: %v", err)
		return err
	}

	s, err := transport.ListenTCP(fabricaddr, transport.ServerConfig{
		Config: transport.Config{
			TLS: tlsconf,
		},
	})
	if err != nil {
		return err
	}

	log.Printf("fabric agent listening at [%v]", s.Addr().String())

	// dummy lease agent here only, do nothing
	leaseConfig := lease.AgentConfig{}
	leaseConfig.SetDefault()

	leaselistener, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", "0"))
	if err != nil {
		return err
	}

	l, err := lease.NewAgent(leaseConfig, leaselistener, func(addr string) (net.Conn, error) { return nil, nil })
	if err != nil {
		return err
	}

	now := int(time.Now().Unix())
	fakeid := federation.NodeIDFromMD5(strconv.Itoa(now))
	myid := federation.NodeIDFromMD5("FabricPing")

	timeout := c.Duration("timeout")

	config := federation.SiteNodeConfig{
		ClientDialer: func(addr string) (*transport.Client, error) {
			conn, err := net.DialTimeout("tcp", addr, timeout)
			if err != nil {
				return nil, fmt.Errorf("cannot establish tcp connection to %v error: %v", addr, err)
			}

			log.Printf("voteping tcp connected, resolved address: %v, local address: %v", conn.RemoteAddr().String(), conn.LocalAddr().String())

			return transport.Connect(conn, transport.ClientConfig{
				Config: transport.Config{
					TLS: getTlsConfig(conn, c),
				},
			})
		},
		TransportServer: s,
		LeaseAgent:      l,
		Instance: federation.NodeInstance{
			Id:         myid,
			InstanceId: uint64(now),
		},
		SeedNodes: []federation.SeedNodeInfo{
			{
				Id:      fakeid,
				Address: conn.RemoteAddr().String(),
			},
		},
	}

	sitenode, err := federation.NewSiteNode(config)
	if err != nil {
		return err
	}
	go sitenode.Serve()
	defer sitenode.Close()

	partners, err := sitenode.Discover(context.Background())
	if err != nil {
		return err
	}

	zero := federation.NodeID{}

	fmt.Printf("%v\t%v\t%v", "InstanceId", "Address", "Phase")
	fmt.Println()

	sort.Slice(partners, func(i, j int) bool {
		return strings.Compare(partners[i].Address, partners[j].Address) < 0
	})

	found := false

	for _, p := range partners {
		if p.Instance.Id == fakeid {
			continue
		}

		if p.Instance.Id == myid {
			continue
		}

		fmt.Printf("%v\t%v\t%v", p.Instance, p.Address, p.Phase)

		if p.Token.Range.Contains(zero) && p.Phase == federation.NodePhaseRouting {
			fmt.Printf("\tFMM")
		}

		fmt.Println()
		found = true
	}

	if !found {
		fmt.Println()
		fmt.Println("!! Seems the node is in Zombie mode, please delete StartStopNode.txt in the fabric data directory !!")
		fmt.Println()
	}

	return nil
}
