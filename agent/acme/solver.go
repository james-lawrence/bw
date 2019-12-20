package acme

import (
	"log"
	"time"
)

type solver Service

func (t solver) Present(domain, token, keyAuth string) error {
	log.Println("PRESENT INITIATED", domain, token, keyAuth)
	defer log.Println("PRESENT COMPLETED", domain, token, keyAuth)
	time.Sleep(time.Second)
	return nil
}

func (t solver) CleanUp(domain, token, keyAuth string) error {
	log.Println("CLEANUP INTIATED", domain, token, keyAuth)
	defer log.Println("CLEANUP COMPLETED", domain, token, keyAuth)

	time.Sleep(5 * time.Second)
	return nil
}
