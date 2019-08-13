package main

import (
	"log"
	"testing"
)

func TestPoll(t *testing.T) {
	_, err := poll("https://example.com") // valid URL
	if err != nil {
		t.Fatal(err)
	}
	_, err = poll("https://expired.badssl.com") // expired SSL Certification
	if err != nil {
		log.Println(err)
	}
	_, err = poll("http://127.0.0.254") // not exist address
	if err != nil {
		log.Println(err)
	}
}
