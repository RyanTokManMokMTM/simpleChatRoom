package main

import (
	"io"
	"log"
	"net"
	"os"
)

func main() {
	//connect to tcp server
	conn, err := net.Dial("tcp", ":2022")
	if err != nil {
		panic(err)
	}

	done := make(chan struct{})
	go func() {
		io.Copy(os.Stdout, conn) // NOTE: ignoring errors
		log.Println("done")
		done <- struct{}{} // signal the main goroutine
	}()

	copyMessage(conn, os.Stdin)
	conn.Close()
	<-done

}

func copyMessage(dst io.Writer, src io.Reader) {
	if _, err := io.Copy(dst, src); err != nil {
		log.Fatalln(err)
	}
}
