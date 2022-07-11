package comm_test

import (
	"io"
	"log"
	"net"
	"testing"
	"time"

	"github.com/nasa-jpl/golaborate/comm"
)

func tcpEchoServer(addr string) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal("could not listen, debug test aborted")
	}
	log.Println("tcp loopback started successfully")
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("error accepting connection:", err)
		}
		log.Println("new conn accepted")
		go func() { io.Copy(conn, conn) }() // use goroutines to handle multiple connections
	}
}

func LoggingDebugWithEchoToCapacity(poolSize int) {
	go tcpEchoServer("localhost:8765")
	maker := func() (io.ReadWriteCloser, error) {
		return net.Dial("tcp", "localhost:8765")
	}
	pool := comm.NewPool(poolSize, time.Second, maker)
	for i := 0; i < poolSize; i++ {
		log.Println("taking connection", i+1, "from pool")
		conn, err := pool.Get()
		if err != nil {
			log.Fatal("could not get connection:", err)
		}
		log.Println("got conn", i+1, conn)
	}
}

func LoggingDebugWithEchoReleasesReuse(poolSize int) {
	go tcpEchoServer("localhost:8765")
	maker := func() (io.ReadWriteCloser, error) {
		return net.Dial("tcp", "localhost:8765")
	}
	pool := comm.NewPool(poolSize, time.Second, maker)
	for i := 0; i < poolSize; i++ {
		log.Println("taking connection", i+1, "from pool")
		conn, err := pool.Get()
		if err != nil {
			log.Fatal("could not get connection:", err)
		}
		log.Println("got conn", i+1, conn)
		pool.Put(conn)
		log.Println("returned conn", i+1, conn)
	}
	time.Sleep(time.Duration(poolSize) * time.Millisecond * 100)
	log.Println(pool.Size())
}

func LoggingDebugWithEchoReleasesExpires(poolSize int) {
	go tcpEchoServer("localhost:8765")
	maker := func() (io.ReadWriteCloser, error) {
		return net.Dial("tcp", "localhost:8765")
	}
	pool := comm.NewPool(poolSize, 100*time.Nanosecond, maker) // don't blow up the CPU by running a sleep every ~4 clocks
	for i := 0; i < poolSize; i++ {
		log.Println("taking connection", i+1, "from pool")
		conn, err := pool.Get()
		if err != nil {
			log.Fatal("could not get connection:", err)
		}
		log.Println("got conn", i+1, conn)
		pool.Put(conn)
		log.Println("returned conn", i+1, conn)
	}
	time.Sleep(time.Duration(poolSize) * time.Millisecond * 100)
	log.Println(pool.Size())
}

func LoggingDebugDeadlocksIfTryToTakeTooMany(poolSize int) {
	go tcpEchoServer("localhost:8765")
	maker := func() (io.ReadWriteCloser, error) {
		return net.Dial("tcp", "localhost:8765")
	}
	pool := comm.NewPool(poolSize, 1*time.Second, maker) // don't blow up the CPU by running a sleep every ~4 clocks
	held := []io.ReadWriter{}
	for i := 0; i < poolSize; i++ {
		rw, err := pool.Get()
		if err != nil {
			log.Fatal("could not get connection:", err)
		}
		held = append(held, rw)
	}
	newConn := make(chan io.ReadWriter, 1)
	// now that they are all taken out, try to get a new one
	go func() {
		rw, _ := pool.Get()
		newConn <- rw
	}()
	select {
	case <-newConn:
		log.Fatal("failed to prevent pool overflow")
	case <-time.After(3 * time.Second):
		log.Println("succeeded in maintaining pool size")
	}
}

func TestLoggingDebugWithEchoToCapacity(t *testing.T) {
	LoggingDebugWithEchoToCapacity(3)
}

func TestLoggingDebugWithEchoReleasesReuse(t *testing.T) {
	LoggingDebugWithEchoReleasesReuse(3)
}

func TestLoggingDebugWithEchoReleasesExpires(t *testing.T) {
	LoggingDebugWithEchoReleasesExpires(3)
}

func TestLoggingDebugWithEchoMaintainsSize(t *testing.T) {
	LoggingDebugDeadlocksIfTryToTakeTooMany(3)
}
