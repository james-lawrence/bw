package main

import (
	"log"
	"os"

	"github.com/james-lawrence/bw/agent/acme"
	"github.com/james-lawrence/bw/internal/protox"
)

func main() {
	var (
		challenge *acme.CertificateResponse
	)

	log.Println("args", os.Args)
	priv, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatalln("failed to read private key", os.Args[1])
	}

	cert, err := os.ReadFile(os.Args[2])
	if err != nil {
		log.Fatalln("failed to read cert", os.Args[2])
	}

	auth, err := os.ReadFile(os.Args[3])
	if err != nil {
		log.Fatalln("failed to read authority", os.Args[3])
	}

	challenge = &acme.CertificateResponse{
		Private:     priv,
		Certificate: cert,
		Authority:   auth,
	}

	if err := protox.WriteFile(os.Args[4], 0600, challenge); err != nil {
		log.Fatalln("failed to write challenge", err)
	}
}
