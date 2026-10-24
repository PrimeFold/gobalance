package main

import (
	"bytes"
	"io"
	"sync"
	"testing"
)


type dummyStream struct{
	data []byte
	offset int
}

func (d *dummyStream) Read(p []byte) (n int, err error){
	if d.offset >= len(d.data){
		return 0,io.EOF
	}
	n = copy(p,d.data[d.offset:])
	d.offset += n
	return n,nil
}

func (d *dummyStream) Write( p []byte) (n int , err error) {
	return len(p),nil
}

func (d *dummyStream) Reset(){
	d.offset = 0
}

// 1. Standard io.Copy: Allocates a fresh 32KB buffer on EVERY call
func BenchmarkStandardCopy(b *testing.B) {
	b.ReportAllocs()
	payload := make([]byte, 1024) // 1 KB payload
	
	for b.Loop() {
		src := bytes.NewReader(payload)
		var dst bytes.Buffer
		_, _ = io.Copy(&dst, src)
	}
}


func BenchmarkPooledCopy(b *testing.B){
	payload:= make([]byte,1024)
	src:= &dummyStream{data: payload}
	dst:= &dummyStream{}
	
	var testPool = sync.Pool{
		New : func ()  any {
			b:= make([]byte,32*1024)
			return &b
		},
	}
	
	b.ResetTimer()
	b.ReportAllocs()


	for b.Loop(){
		src.Reset()
		buffPtr:= testPool.Get().(*[]byte)
		_,_ = io.CopyBuffer(dst,src,*buffPtr)
		testPool.Put(buffPtr)
	}
}
