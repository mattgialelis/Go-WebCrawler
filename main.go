package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/html"
)

var Mutex = &sync.Mutex{} //used to stop race condition on FoundHosts when multi go routines are running

type SiteMap struct {
	Host   string
	Links  []string
	Static []string
}

func main() {
	var (
		CrawlHost string
		Depth     int
	)
	result := make(chan SiteMap)
	CrawlerWg := &sync.WaitGroup{}
	Visited := make(map[string]bool)
	flag.StringVar(&CrawlHost, "url", "https://www.golang.org/", "URL to start Crawling")
	flag.IntVar(&Depth, "depth", 2, "How deep the crawler should search thru the site")
	flag.Parse()

	CrawlerWg.Add(1)
	go Crawl(CrawlHost, Depth, result, CrawlerWg, Visited)

	go func() {
		CrawlerWg.Wait() // Wait for the Crawler Waitgroup to reach zero indicating processing is done
		close(result)    //Clean up the channel
	}()

	for s := range result { // Range over the result channel, to print the responses as they are available
		sitefinder, _ := yaml.Marshal(s)
		fmt.Println(string(sitefinder))
	}
}

func Crawl(url string, depth int, ret chan SiteMap, CrawlerWg *sync.WaitGroup, Visited map[string]bool) {
	defer CrawlerWg.Done()
	if depth == 0 {
		return
	}

	smap, err := Fetcher(url)
	if err != nil {
		return
	}

	for _, u := range smap.Links {
		Mutex.Lock() //Lock to save any race condtions when multi go routines are reading/writing to the VisitedMap
		Visited[url] = true
		if _, ok := Visited[u]; !ok {
			CrawlerWg.Add(1) //Add to wait group to be sure we wait for all Recursive Calls can return
			go Crawl(u, depth-1, ret, CrawlerWg, Visited)
		}
		Mutex.Unlock() //Unlock Mutex opened on line 66
	}

	ret <- smap //Send Sitemap struct response from Fetcher back to Main for printing as results come thru
	return
}

func Fetcher(webUrl string) (SiteMap, error) {
	var (
		fetcherwg sync.WaitGroup
		links     = make(chan string)
		smap      = SiteMap{Host: webUrl}
	)
	foundHrefs := make(map[string]bool) //Used to track which Refs have been seen in this Page already and keep Response Uniq
	fetcherwg.Add(1)

	respBody, SiteUrl, err := GetSiteBody(webUrl)
	if err != nil {
		return SiteMap{}, err
	}
	defer respBody.Close()

	defer fetcherwg.Done()

	fetcherwg.Add(1) //Add to wait group to be sure the appending to the response(smap)struct can complete
	go func() {
		defer fetcherwg.Done()
		for link := range links {
			if filepath.Ext(link) == "" {
				smap.Links = append(smap.Links, link)
				continue
			}
			smap.Static = append(smap.Static, link)
		}
	}()

	go func() {
		fetcherwg.Wait()
		close(links)
	}()

	tokens := html.NewTokenizer(respBody)
	for {
		CurToken := tokens.Next()
		switch {
		case CurToken == html.ErrorToken:
			return smap, nil
		case CurToken == html.StartTagToken:
			t := tokens.Token()
			switch t.Data {
			case "a", "link", "img", "image", "script":
				for _, a := range t.Attr {
					switch a.Key {
					case "href":
						if _, ok := foundHrefs[a.Val]; !ok {
							foundHrefs[a.Val] = true
							fetcherwg.Add(1)
							go ParseHref(a.Val, SiteUrl, links, &fetcherwg)
						}
						break
					case "src":
						if _, ok := foundHrefs[a.Val]; !ok {
							foundHrefs[a.Val] = true
							fetcherwg.Add(1)
							go ParseHref(a.Val, SiteUrl, links, &fetcherwg)
						}
						break
					}
				}
			}
		}

	}

}

func ParseHref(href string, hosturl *url.URL, links chan string, wg *sync.WaitGroup) {
	defer wg.Done()
	relURL, _ := url.Parse(href)
	absURL := hosturl.ResolveReference(relURL)
	if absURL.Host == hosturl.Host {
		links <- absURL.String()
		return
	}
	return
}

func GetSiteBody(webUrl string) (io.ReadCloser, *url.URL, error) {
	vaildUrl, err := url.ParseRequestURI(webUrl)
	if err != nil {
		return nil, nil, err
	}

	httpClient := &http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := httpClient.Get(vaildUrl.String())

	if err != nil {
		logrus.Errorf("failed to get URL %s: %v", vaildUrl.String(), err)
		return nil, nil, err
	}
	if resp.Header.Get("Content-Type") == "text/html" { // "" to allow for no header being sent
		return nil, nil, err
	}
	return resp.Body, vaildUrl, nil
}
