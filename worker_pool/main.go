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

func worker(jobs <-chan string, results chan<- task, wg *sync.WaitGroup) {
	defer wg.Done()
	for URL := range jobs {
		res := processURL(URL)
		results <- task{URL: URL, result: res}
	}

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

func handleResults(ctx context.Context, results <-chan task, URLlist []string, done chan<- struct{}) {
	processedURLs := make(map[string]bool)
loop:
	for {
		select {
		case res, ok := <-results:
			if !ok {
				break loop
			}
			fmt.Printf("%s: %d\n", res.URL, res.result)
			processedURLs[res.URL] = true
		case <-ctx.Done():
			fmt.Println("stopped")
			break loop
		}
	}

	for _, URL := range URLlist {
		if _, ok := processedURLs[URL]; !ok {
			fmt.Printf("%s: cancel\n", URL)
		}
	}
	done <- struct{}{}
}

func main() {
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)

	if len(os.Args) < 2 {
		fmt.Println("URL list is empty. Must be specified as command-line argument:\nfoo.exe \"https://ya.ru, https://google.com, https://mts.ru\"")
		return
	}

	fmt.Println("processing...")
	URLlist := strings.Split(os.Args[1], ", ")

	const workersCount = 4
	jobs := make(chan string)
	results := make(chan task)
	done := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(workersCount)

	// run our workers
	for i := 0; i < workersCount; i++ {
		go worker(jobs, results, &wg)
	}

	// process results asynchronously
	go handleResults(ctx, results, URLlist, done)

	for _, URL := range URLlist {
		jobs <- URL
	}

	close(jobs)
	// wait until all workers done their job
	wg.Wait()

	close(results)
	// wait for tasks results to be processed
	<-done
}
