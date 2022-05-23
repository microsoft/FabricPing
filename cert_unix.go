//go:build !windows
// +build !windows

package main

import (
	"crypto/tls"
)

var certSearchPath = "/var/lib/sfcerts/"

func searchCert(keyword string) (*tls.Certificate, error) {
	return searchPkcs8(certSearchPath, keyword)
}
