package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

type task struct {
	URL    string
	result int
}

func worker(URL string) chan task {
	ch := make(chan task)
	go func() {
		res := processURL(URL)
		ch <- task{URL: URL, result: res}
		close(ch)
	}()
	return ch
}

func processURL(URL string) int {
	c := http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := c.Get(URL)
	if err != nil {
		fmt.Println(err)
		return http.StatusInternalServerError
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode
	}

	nl := []byte{'\n'}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return http.StatusInternalServerError
	}

	count := bytes.Count(body, nl)
	if len(body) > 0 && !bytes.HasSuffix(body, nl) {
		count++
	}

	return count
}

func fanIn(chans []chan task) chan task {
	res := make(chan task)
	var wg sync.WaitGroup
	wg.Add(len(chans))

	for _, ch := range chans {
		go func(ch chan task) {
			defer wg.Done()
			for n := range ch {
				res <- n
			}
		}(ch)
	}

	// Close results channel
	go func() {
		wg.Wait()
		close(res)
	}()
	return res
}

func main() {
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)

	if len(os.Args) < 2 {
		fmt.Println("URL list is empty. Must be specified as command-line argument:\nfoo.exe \"ya.ru, google.com, mts.ru\"")
		return
	}

	fmt.Println("processing...")
	URLlist := strings.Split(os.Args[1], ", ")
	URLcount := len(URLlist)
	results := make(map[string]int, URLcount)

	var chans []chan task
	for _, URL := range URLlist {
		ch := worker(URL)
		chans = append(chans, ch)
	}
	out := fanIn(chans)

loop:
	for {
		select {
		case res, ok := <-out:
			if !ok {
				break loop
			}
			results[res.URL] = res.result
		case <-ctx.Done():
			fmt.Println("stopped")
			break loop
		}

	}

	for _, URL := range URLlist {
		count, ok := results[URL]
		if ok {
			fmt.Printf("%s: %d\n", URL, count)
		} else {
			fmt.Printf("%s: cancel\n", URL)
		}
	}
}
