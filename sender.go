// Package main provides ...
package main

import (
	"encoding/binary"
	"io"
	"log"
	"net"
)

//controlServer start the controls channel on the client
func controlServer(bind, address string) {
	clistener, err := net.Listen("tcp", bind)
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Control Server Started")
	for {
		conn, err := clistener.Accept()
		if err != nil {
			log.Print(err)
			continue
		}
		var buf int32
		binary.Read(conn, binary.LittleEndian, &buf)
		log.Printf("control message arrived")
		//TODO: define the other cases
		switch buf {
		case bandwidth:
			sendData(address, 150)
			//case latency:
			//case load:
			//case stress:
		case die:
			break

		default:
			continue
		}
	}
}

//sendData send size data (in megabytes)to the string addr
func sendData(addr string, size int64) {
	log.Println("sending data")
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatalf("sendData: %v", err)
	}
	n, err := io.CopyN(conn, devZero, size*(1000000))
	if err != nil {
		log.Fatal(err)
	}
	if n != size*1000000 {
		log.Fatalf("couldnt send %v Megabytes", float64(n)/float64(1000000))
	}
	log.Printf("sent %v MB", size)
}