package main

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/james-lawrence/bw/agent/acme"
	"github.com/james-lawrence/bw/internal/x/protox"
)

func main() {
	var (
		challenge *acme.ChallengeResponse
	)

	log.Println("args", os.Args)
	priv, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		log.Fatalln("failed to read private key", os.Args[1])
	}

	cert, err := ioutil.ReadFile(os.Args[2])
	if err != nil {
		log.Fatalln("failed to read cert", os.Args[2])
	}

	auth, err := ioutil.ReadFile(os.Args[3])
	if err != nil {
		log.Fatalln("failed to read authority", os.Args[3])
	}

	challenge = &acme.ChallengeResponse{
		Private:     priv,
		Certificate: cert,
		Authority:   auth,
	}

	protox.WriteFile(os.Args[4], 0600, challenge)
}
