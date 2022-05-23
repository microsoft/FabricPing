package main

import (
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/urfave/cli/v2"
)

var mainver string = "(devel)"

func version() string {

	var v = mainver

	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return v
	}

	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			v = fmt.Sprintf("%v, %v", v, s.Value[:9])
		case "vcs.time":
			v = fmt.Sprintf("%v, %v", v, s.Value)
		}
	}

	v = fmt.Sprintf("%v, %v", v, bi.GoVersion)

	return v
}

var certcache sync.Map

func getTlsConfig(conn net.Conn, c *cli.Context) *tls.Config {
	if c.Bool("non-secure") {
		return nil
	}

	var remotetps []string

	findcert := func() (*tls.Certificate, error) {

		certkeyword := c.String("cert")
		if certkeyword != "" {

			cachecert, ok := certcache.Load(certkeyword)
			if ok {
				return cachecert.(*tls.Certificate), nil
			}

			cert, err := searchCert(certkeyword)
			if err != nil {
				return nil, fmt.Errorf("search cert return error: %v", err)
			}

			log.Printf("using certificate thumbprint [%v]", fmt.Sprintf("%x", sha1.Sum(cert.Certificate[0])))

			certcache.Store(certkeyword, cert)

			return cert, nil
		}

		for _, remotetp := range remotetps {
			if remotetp != "" {

				cachecert, ok := certcache.Load(remotetp)
				if ok {
					return cachecert.(*tls.Certificate), nil
				}

				log.Printf("discovering certificate on machine with thumbprint [%v]", remotetp)
				cert, err := searchCert(remotetp)
				if err != nil {
					log.Printf("did not find certifcate thumbprint [%v], error: %v", remotetp, err)
					continue
				}

				certcache.Store(remotetp, cert)

				return cert, nil
			}
		}

		return nil, fmt.Errorf("did not find any certificate")
	}

	return &tls.Config{
		InsecureSkipVerify: true,
		GetCertificate: func(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
			return findcert()
		},
		GetClientCertificate: func(cri *tls.CertificateRequestInfo) (*tls.Certificate, error) {
			return findcert()
		},
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			remotetps = nil
			for _, rawCert := range rawCerts {
				cert, err := x509.ParseCertificate(rawCert)

				if err != nil {
					log.Printf("error read remote cert error %v", err)
					continue
				}

				valid := true
				valid = valid && !time.Now().After(cert.NotAfter)
				valid = valid && !time.Now().Before(cert.NotBefore)

				thumb := fmt.Sprintf("%x", sha1.Sum(rawCert))
				remotetps = append(remotetps, thumb)
				log.Printf("[%v] is presenting certificate [CN=%v] [expired=%v] [not before=%v] [not after=%v] [thumbprint=%v]", conn.RemoteAddr().String(), cert.Subject.CommonName, !valid, cert.NotBefore, cert.NotAfter, thumb)
			}
			return nil
		},
	}
}

func main() {

	app := &cli.App{
		Name:      "Fabric Ping",
		Usage:     "Ping a Fabric/FabricLease endpoint",
		UsageText: "FabricPing [OPTIONS] <address:port>",
		Description: `Ping Fabric:  FabricPing 127.0.0.1:1025
Ping Lease:   FabricPing -l 127.0.0.1:1026
Discover:     FabricPing -d 127.0.0.1:1025
		`,
		Version: version(),
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "lease",
				Aliases: []string{"l"},
				Usage:   "ping a lease endpoint, [listen-addr] must be reachable from the remote lease server",
				Value:   false,
			},
			&cli.BoolFlag{
				Name:    "discover",
				Aliases: []string{"d"},
				Usage:   "discover nodes in the cluster, [listen-addr] must be reachable from the fabric server",
				Value:   false,
			},
			&cli.StringFlag{
				Name:  "listen-addr",
				Usage: "lease agent (lease ping) or federation agent (discover) listening address, use [auto] to create one automatically",
				Value: "auto",
			},
			&cli.StringFlag{
				Name:        "cert",
				Aliases:     []string{"c"},
				Usage:       "thumbprint, CN or a pem path to certificate",
				DefaultText: "search on the machine",
			},
			&cli.StringFlag{
				Name:  "cert-path",
				Usage: "certificate files search path",
				Value: certSearchPath,
			},
			&cli.DurationFlag{
				Name:    "interval",
				Aliases: []string{"i"},
				Value:   time.Second * 2,
			},
			&cli.DurationFlag{
				Name:  "timeout",
				Value: time.Second * 30,
			},
			&cli.BoolFlag{
				Name:  "non-secure",
				Value: false,
			},
			&cli.IntFlag{
				Name:        "count",
				Value:       0,
				Usage:       "stop after <count> pings, set 0 for infinity",
				DefaultText: "infinity",
			},
		},

		Action: func(c *cli.Context) error {

			addr := c.Args().First()

			if addr == "" {
				return cli.ShowAppHelp(c)
			}

			timeout := c.Duration("timeout")
			interval := c.Duration("interval")
			certSearchPath = c.String("cert-path")

			log.Printf("start fabric ping target: %v, timeout %v", addr, timeout)

			conn, err := net.DialTimeout("tcp", addr, timeout)
			if err != nil {
				return fmt.Errorf("cannot establish tcp connection to %v error: %v", addr, err)
			}
			defer conn.Close()

			log.Printf("tcp connected, resolved address: %v, local address: %v", conn.RemoteAddr().String(), conn.LocalAddr().String())

			mode := 0

			if c.Bool("lease") {
				mode |= 0b01
			}

			if c.Bool("discover") {
				mode |= 0b10
			}

			count := c.Int("count")
			listenaddr := c.String("listen-addr")

			switch mode {
			case 0b11:
				return fmt.Errorf("cannot use --lease and --discover together")
			case 0b01:
				log.Printf("starting lease ping, ctrl + c to break")
				return leaseping(conn, getTlsConfig(conn, c), interval, timeout, listenaddr, count)
			case 0b10:
				log.Printf("starting discovering, ctrl + c to break")
				return discover(conn, getTlsConfig(conn, c), listenaddr, c)
			default:
				log.Printf("starting fabric ping, ctrl + c to break")
				return fabricping(conn, getTlsConfig(conn, c), interval, timeout, count)
			}
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
