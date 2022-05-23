package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func searchPkcs8(searchpath, keyword string) (*tls.Certificate, error) {
	// input is a file
	if _, err := os.Stat(keyword); err == nil {
		return loadPkcs8(keyword)
	}

	if searchpath == "" {
		return nil, fmt.Errorf("cert file [%v] not found", keyword)
	}

	matches, err := filepath.Glob(path.Join(searchpath, "*"))
	if err != nil {
		return nil, err
	}

	for _, f := range matches {
		if strings.Contains(strings.ToLower(filepath.Base(f)), keyword) {

			cert, err := loadPkcs8(f)
			if err != nil {
				log.Printf("skip matched file %v, err [%v]", f, err)
				continue
			}

			log.Printf("using certificate file [%v]", f)
			return cert, nil
		}
	}

	return nil, fmt.Errorf("no certificate file matches in %v", searchpath)
}

func loadPkcs8(path string) (*tls.Certificate, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cert tls.Certificate

	for {
		block, rest := pem.Decode(data)
		if block == nil {
			break
		}

		switch block.Type {
		case "CERTIFICATE":
			cert.Certificate = append(cert.Certificate, block.Bytes)
		case "PRIVATE KEY":
			key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				continue
			}

			cert.PrivateKey = key
		}

		data = rest
	}

	if len(cert.Certificate) == 0 {
		return nil, fmt.Errorf("no certificate in pem")
	}

	if cert.PrivateKey == nil {
		return nil, fmt.Errorf("no private key in pem")
	}

	return &cert, nil
}
