# go-concurrency-example

Study exercises of using concurrency patterns in Golang.

## User story

Console app, which gets URLs list from command-line argument and prints lines count on each site.

Example:

```cmd
> myApp "https://ya.ru, https://google.com, https://mts.ru"
> processing...
> https://ya.ru: 234
> https://google.com: 123
> https://mts.ru: 345
```

## Implementation requirements

- all requests must run concurrently
- on Ctrl+C pressed app must stop processing and prints `cancel` for incomplete requests
- in case of an error receiving the page, the http response code should be printed
- fan-in implementation must prints lines count in the same order as in the command-line parameter
- worker pool implementation can print results in any order
