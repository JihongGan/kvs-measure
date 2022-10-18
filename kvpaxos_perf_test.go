// Jihong Gan <jhgan@umich.edu>

package kvpaxos

import (
	"fmt"
	"math/rand"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"
	"unsafe"
)

// cr. https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-go
func randStr(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const (
		letterIdxBits = 6                    // 6 bits to represent a letter index
		letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
		letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
	)
	var src = rand.NewSource(time.Now().UnixNano())
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return *(*string)(unsafe.Pointer(&b))
}

func shuffleSA(a []string) []string {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(a), func(i, j int) { a[i], a[j] = a[j], a[i] })
	return a
}

func benchmarkKVS(vsize, nservers, nclients int) {
	runtime.GOMAXPROCS(4)
	const nexists = 100
	const trialsPerClient = 100
	trials := nclients * trialsPerClient

	var kva []*KVPaxos = make([]*KVPaxos, nservers)
	var kvh []string = make([]string, nservers)
	var cka []*Clerk = make([]*Clerk, nclients)
	defer cleanup(kva)
	for i := 0; i < nservers; i++ {
		kvh[i] = port("measure", i)
	}
	for i := 0; i < nservers; i++ {
		kva[i] = StartServer(kvh, i)
	}
	for i := 0; i < nclients; i++ {
		cka[i] = MakeClerk(shuffleSA(kvh))
	}

	// fill the kvs
	for i := 0; i < nexists; i++ {
		cka[0].Put(strconv.Itoa(i), randStr(vsize))
	}
	// dummy values to test writes
	var vals []string
	for i := 0; i < nexists; i++ {
		vals = append(vals, randStr(vsize))
	}

	fmt.Printf("Benchmarking with value size %v bytes, %v servers, %v clients...\n", vsize, nservers, nclients)
	// read-only workload
	{
		fmt.Printf("Running read-only workload...\n")
		lats := make(chan time.Duration, trials)
		var wg sync.WaitGroup
		start := time.Now()

		for i := 0; i < nclients; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				for j := 0; j < trialsPerClient; j++ {
					key := strconv.Itoa(j % nexists)
					start := time.Now()
					cka[i].Get(key)
					lats <- time.Since(start)
				}
			}(i)
		}
		wg.Wait()
		close(lats)

		thruput := float64(trials) / time.Since(start).Seconds()
		avglat := float64(0)
		for lat := range lats {
			avglat += float64(lat.Milliseconds())
		}
		avglat /= float64(trials)
		fmt.Printf("Throughput: %.2f ops/s\n", thruput)
		fmt.Printf("Average latency: %.2f ms\n", avglat)
	}

	// 50% Get, 25% Put, 25% Append
	{
		fmt.Printf("Running half-read-half-write workload...\n")
		lats := make(chan time.Duration, trials)
		var wg sync.WaitGroup
		start := time.Now()

		for i := 0; i < nclients; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				myck := cka[i]
				for j := 0; j < trialsPerClient; j++ {
					rn := rand.Int() % 4
					if rn == 0 || rn == 1 {
						key := strconv.Itoa(j % nexists)
						start := time.Now()
						myck.Get(key)
						lats <- time.Since(start)
					} else {
						key := strconv.Itoa(rand.Int() % (nexists * 2))
						val := vals[j%len(vals)]
						if rn == 2 {
							start := time.Now()
							myck.Put(key, val)
							lats <- time.Since(start)
						} else if rn == 3 {
							start := time.Now()
							myck.Append(key, val)
							lats <- time.Since(start)
						}
					}
				}
			}(i)
		}
		wg.Wait()
		close(lats)

		thruput := float64(trials) / time.Since(start).Seconds()
		avglat := float64(0)
		for lat := range lats {
			avglat += float64(lat.Milliseconds())
		}
		avglat /= float64(trials)
		fmt.Printf("Throughput: %.2f ops/s\n", thruput)
		fmt.Printf("Average latency: %.2f ms\n", avglat)
	}
}

func TestPerfValSize(t *testing.T) {
	benchmarkKVS(512, 4, 16)
	benchmarkKVS(4096, 4, 16)
	benchmarkKVS(64000, 4, 16)
	benchmarkKVS(512000, 4, 16)
}

func TestPerfNClients(t *testing.T) {
	benchmarkKVS(1024, 4, 4)
	benchmarkKVS(1024, 4, 8)
	benchmarkKVS(1024, 4, 16)
	benchmarkKVS(1024, 4, 32)
}

func TestPerfNServers(t *testing.T) {
	benchmarkKVS(1024, 1, 16)
	benchmarkKVS(1024, 2, 16)
	benchmarkKVS(1024, 4, 16)
	benchmarkKVS(1024, 8, 16)
}
