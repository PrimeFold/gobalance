package main

import (
	"fmt"
	"net"
	"net/url"
	"sync"
	"testing"
	"time"
)


func TestOutageFunction(t *testing.T){
	proxyPort:="9800"

	pool:= &ServerPool{}
	

	p1:= "9801"
	p2:= "9802"
	p3:= "9803"


	u1, _ := url.Parse("localhost:" + p1)
	u3, _ := url.Parse("localhost:" + p3)
	u2, _ := url.Parse("localhost:" + p2)

	b1:= &Backend{
		URL: u1,
		Alive: false,
	}

	b2:= &Backend{
		URL: u2,
		Alive: false,
	}
	b3:= &Backend{
		URL: u3,
		Alive: true,
	}

	backendSlice:= []*Backend{b1,b2,b3}

	for _,b:= range backendSlice{
		pool.AddBackend(b)
	}

	go startEchoServer(p3)

	proxyL,err:= net.Listen("tcp","localhost:"+proxyPort)
	time.Sleep(50*time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to start proxy server : %v",err)
	}

	defer proxyL.Close()

	go func ()  {
		for{
			clientConn,err:= proxyL.Accept()
			if err != nil{
				return
			}
			go proxyConnection(clientConn,pool)
		}
	}()

	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done() // 🔒 Essential! Decrement counter when done!
			SendBytePayload(clientID, t)
		}(i)
	}

	wg.Wait()

	// 6. Assert zero connection leaks on Backend 3
	if b3.GetActiveConnections() != 0 {
		t.Errorf("Connection leak on backend 3! Expected 0, got %d", b3.GetActiveConnections())
	}

	
	
}



func SendBytePayload(clientId int , t *testing.T){
	conn,err:= net.Dial("tcp","localhost:9800")

	if err != nil{
		t.Errorf("Error Dialing : %v",err)
		return
	}

	defer conn.Close()

	msg:= fmt.Sprintf("Ping from client %d",clientId)

	if _,err:= conn.Write([]byte(msg)); err != nil{
		t.Errorf("Client %d failed to write : %v",clientId,err)
		return
	}
	

	// Reading the echo response..
	buf:= make([]byte,1024)

	n,err:= conn.Read(buf)

	if err != nil{
		t.Errorf("Client %d failed to read : %v",clientId,err)
		return 
	}

	if string(buf[:n]) != msg {
		t.Errorf("Client %d : expected '%s' , got '%s' ",clientId,msg,string(buf[:n]))
	}
}
