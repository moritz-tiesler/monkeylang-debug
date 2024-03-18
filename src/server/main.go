package main

import (
	"flag"
	"log"
	"os"
)

func main() {

	logFile, err := os.OpenFile("/tmp/debugserver.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic("could not open log file")
	}
	log.SetOutput(logFile)
	flag.Parse()
	stream := StdioReadWriteCloser{}
	StartSession(stream)
	if err != nil {
		log.Fatal("Could not start server: ", err)
	}
}
