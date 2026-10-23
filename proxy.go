package main

import (
	"context"
	"io"
	"log"
	"net"
	"sync"
	"time"
)


var bufferPool = sync.Pool{
	New: func() any {
		return make([]byte, 32*1024)
	},
}

func proxyConnection(clientConn net.Conn,pool *ServerPool){
	pool.IncrementTotalRequests() //Incrementing requests by 1 at the start
	var targetConn net.Conn
	var target *Backend
	var err error

	maxRetries:= 3

	for attempts:=0 ; attempts < maxRetries ; attempts++{
		target = pool.GetLeastConnectedBackend()
		if target == nil{
			clientConn.Close()
			return
		}
		ctx,cancel:= context.WithTimeout(context.Background(),1*time.Second)

		var dialer net.Dialer

		targetConn,err = dialer.DialContext(ctx,"tcp",target.URL.Host)

		cancel()

		if err == nil{ //If the backend answered YES
			break
		}
		
		pool.IncrementFailedDials()
		target.SetAlive(false)

		log.Printf("Server %s marked as Dead as it didn't respond within 1 Second",target.URL.Host)

		if targetConn == nil{ //If Connection not open 
			clientConn.Close()
			return
		}


		//If server answers , we clean up and increase connection count on that server..
		target.IncrementConnectionCount()
		defer target.DecrementConnectionCount()

		//Making sure backend and client closes connection after this function exits..
		defer targetConn.Close()
		defer clientConn.Close()

	}

	var wg sync.WaitGroup

	wg.Add(2)
	go func ()  {
		defer wg.Done()
		buf:= bufferPool.Get().([]byte)
		defer bufferPool.Put(buf)
		io.CopyBuffer(clientConn,targetConn,buf)
		if tcp,ok:= clientConn.(*net.TCPConn) ; ok{
			tcp.CloseWrite()
		}
	}()

	go func ()  {
		defer wg.Done()
		buf := bufferPool.Get().([]byte)
		defer bufferPool.Put(buf)
		io.CopyBuffer(targetConn,clientConn,buf)
		if tcp,ok:=  targetConn.(*net.TCPConn) ; ok{
			tcp.CloseWrite()
		}
	}()

	wg.Wait()

		
}
