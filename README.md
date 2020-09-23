#Web Crawler Cli


##Description
The webcrawler is a golang based program which does not use any external WebCrawler Crawler Modules avaialble.
The crawler returns a yaml output of the Host it crawlled the links found and any static resources it may have discoverd. 

## Prerequisites
- Golang 1.15 


## How to Build
To build the crawler for your local machine you will have to have the Prerequisites installed before continuing. 

Run the following command in the folder where the main.go can be found.
```bash
go build .
```

Once complete an executable should be found in the current directory named `Go-WebCrawler` 


## Usage
The crawler provides 2 Flags which can be passed in to change the behaviour of the crawler.
```
Usage of ./Go-WebCrawler:
  -depth int
    	How deep the crawler should search thru the site (default 2)
  -url string
    	URL to start Crawling (default "https://golang.org")
```

To run the Crawler with the defaults all that is required is to run

```
./Go-WebCrawler
```

To use any of the options they can be used like below, the flags have no particular order and either can be set or left
as there default
```
./Go-WebCrawler -url https://golang.org -depth 3
```
