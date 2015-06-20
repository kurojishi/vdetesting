package main

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/tatsushid/go-fastping"
)

const (
	bandwidth int32 = 1
	latency   int32 = 2
	load      int32 = 3
	stress    int32 = 4
	die       int32 = 0
)

const (
	stop  int32 = 1
	ready int32 = 2
)

type zeroFile struct{}

type nullFile struct{}

func (d *nullFile) Write(p []byte) (int, error) {
	return len(p), nil
}

func (d *zeroFile) Read(p []byte) (int, error) {
	return len(p), nil
}

var devNull = &nullFile{}
var devZero = &zeroFile{}

func sendControlSignal(address string, msg int32) error {
	conn, err := net.Dial("tcp", address)
	defer conn.Close()
	if err != nil {
		return err
	}
	err = binary.Write(conn, binary.LittleEndian, msg)
	if err != nil {
		return err
	}
	return nil
}

func signalLoop(control string, cch chan int32) {
	for msg := range cch {
		err := sendControlSignal(control, msg)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func devNullConnection(conn net.Conn, deliver chan int64) {

	nbytes, err := io.Copy(devNull, conn)
	if err != nil {
		log.Fatalf("data receive error: %v", err)
	}
	deliver <- nbytes
	return
}

//BandwidthTest is..
func BandwidthTest(iface, port, address string, snaplen int64, cch chan int32) {
	log.Printf("Starting bandwidth test")
	sync := make(chan int32, 1)
	go TCPStats(iface, snaplen, port, sync)
	<-sync
	ticker, sch := PollStats(pid, "bandwidth")
	listener, err := net.Listen("tcp", address)
	defer listener.Close()
	if err != nil {
		log.Fatalf("ReceiveData %v", err)
	}
	cch <- bandwidth
	conn, err := listener.Accept()
	if err != nil {
		log.Fatal(err)
	}

	_, err = io.Copy(devNull, conn)
	if err != nil {
		log.Fatalf("data receive error: %v", err)
	}
	ticker.Stop()
	sch <- true
	close(sch)

	<-sync
	log.Print("Finished bandwidth test")

}

//LatencyTest use ping to control latency
func LatencyTest(address string) {
	rttch := make(chan time.Duration, 10)
	ra, err := net.ResolveIPAddr("ip", address)
	if err != nil {
		log.Fatal(err)
	}
	pinger := fastping.NewPinger()
	pinger.AddIPAddr(ra)
	pinger.OnRecv = func(addr *net.IPAddr, rtt time.Duration) {
		rttch <- rtt

	}
	//set message size to 64 byte
	pinger.Size = 64
	for i := 0; i < 10; i++ {
		pinger.Run()
	}
	close(rttch)

	var sum time.Duration
	var i int
	for rtt := range rttch {
		sum += rtt
		i++
		log.Printf("Test ping:latency %v ms", (float32(rtt)/2)/float32(time.Millisecond))
	}
	log.Printf("Medium Latency is: %v", (float32(sum/2)/float32(i))/float32(time.Millisecond))

}

func manageConnections(address string, sch chan int32, deliver chan int64) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal(err)
	}
	for {
		select {
		case <-sch:
			log.Print("i'm out of this gorouitine")
			return
		default:
			conn, err := listener.Accept()
			if err != nil {
				log.Fatal(err)
			}
			go devNullConnection(conn, deliver)

		}
	}

}

//StressTest lauch a test to see what the vde_switch will do on very intensive traffic
func StressTest(address string, startingPort int, cch chan int32) {
	log.Print("Starting stress test")
	schContainer := make([]chan int32, 0, 150)
	ticker, sch := PollStats(pid, "bandwidth")
	results := make(chan int64)
	for i := 0; i < 150; i++ {
		sch := make(chan int32)
		finalAddr := address + ":" + strconv.Itoa(startingPort+i)
		manageConnections(finalAddr, sch, results)
		schContainer = append(schContainer, sch)

	}
	cch <- stress
	timer := time.NewTimer(1 * time.Minute)
	<-timer.C
	for i := 0; i < len(schContainer); i++ {
		schContainer[i] <- stop
	}
	ticker.Stop()
	sch <- true
	close(sch)
	var dataReceived int64
	for nbytes := range results {
		dataReceived += nbytes
	}
	log.Printf("Received %v bytes", dataReceived)
	log.Print("Finished stress test")

}
