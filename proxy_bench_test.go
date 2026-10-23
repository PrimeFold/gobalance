package main

import (
	"bytes"
	"io"
	"sync"
	"testing"
)

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
	b.ReportAllocs()

	payload:= make([]byte,1024)
	var testPool = sync.Pool{
		New : func ()  any {
			return make([]byte,32*1024)
		},
	}

	for b.Loop(){
		src:= bytes.NewReader(payload)
		var dst bytes.Buffer

		buf:= testPool.Get().([]byte)
		_,_ = io.CopyBuffer(&dst,src,buf)
		testPool.Put(buf)
	}
}
