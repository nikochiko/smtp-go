package smtpstatus

const (
	WelcomeOK         = 220
	ReplyOK           = 250
	IntermediateReply = 354
	UnknownCommand    = 500
	BadSequence       = 503
	WelcomeNotOK      = 554
)
