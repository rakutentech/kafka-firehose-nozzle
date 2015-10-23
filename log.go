package main

import "log"

type LoggerPrinter struct {
	logger *log.Logger
}

func (p LoggerPrinter) Print(title, dump string) {
	p.logger.Printf("[DEBUG] %s: %s", title, dump)
}
