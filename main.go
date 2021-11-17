package main

import (
	"github.com/nikochiko/smtp-go/common"
	"github.com/nikochiko/smtp-go/server"
)

func main() {
	s := server.Server{
		Domain: "localhost",
		Port:   8000,
	}

	err := s.ServeSMTP()
	common.CheckError(err)
}
