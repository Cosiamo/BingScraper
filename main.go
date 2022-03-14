package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// [string]string indicates there's a string on the left side and the right side
// key value pairs are like objects in JavaScript
// syntax is like JSON
// list of all the Bing domains that exist
var bingDomains = map[string]string {
	"com": "",
	"uk": "&cc=GB",
	"us": "&cc=US",
	"tr": "&cc=TR",
	"tw": "&cc=TW",
	"ch": "&cc=CH",
	"se": "&cc=SE",
	"es": "&cc=ES",
	"za": "&cc=ZA",
	"sa": "&cc=SA",
	"ru": "&cc=RU",
	"ph": "&cc=PH",
	"pt": "&cc=PT",
	"pl": "&cc=PL",
	"cn": "&cc=CN",
	"no": "&cc=NO",
	"nz": "&cc=NZ",
	"nl": "&cc=NL",
	"mx": "&cc=MX",
	"my": "&cc=MY",
	"kr": "&cc=KR",
	"jp": "&cc=JP",
	"it": "&cc=IT",
	"id": "&cc=ID",
	"in": "&cc=IN",
	"hk": "&cc=HK",
	"de": "&cc=DE",
	"fr": "&cc=FR",
	"fi": "&cc=FI",
	"dk": "&cc=DK",
	"cl": "&cc=CL",
	"ca": "&cc=CA",
	"br": "&cc=BR",
	"be": "&cc=BE",
	"at": "&cc=AT",
	"au": "&cc=AU",
	"ar": "&cc=AR",
}

type SearchResult struct {
	ResultRank int
	ResultURL string
	ResultTitle string
	ResultDesc string
}

// slices are like arrays
// userAgents are browser engines
var userAgents = []string {
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36",
	"Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/11.0 Safari/604.1.38",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:56.0) Gecko/20100101 Firefox/56.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/11.0 Safari/604.1.38",
}

// need to randomize the user agents so that the search doesn't hit any rate limits
func randomUserAgent() string {
	rand.Seed(time.Now().Unix())
	// take the length of userAgents and create a random number from one of them
	// if the number is 2 it'll select the third value in the list of userAgents
	randNum := rand.Int()%len(userAgents)
	return userAgents[randNum]
}

// builds the URL
func buildBingUrls(searchTerm, country string, pages, count int)([]string, error) {
	// toScrape is what is being returned from this function
	toScrape := []string{}
	searchTerm = strings.Trim(searchTerm, " ")
	searchTerm = strings.Replace(searchTerm, " ", "+", -1)
	if countryCode, found := bingDomains[country]; found {
		for i := 0; i < pages; i++ {
			first := firstParameter(i, count);
			scrapeURL  := fmt.Sprintf("https://bing.com/search?q=%s&first=%d&count=%d%s", searchTerm, first, count, countryCode)
			toScrape = append(toScrape, scrapeURL)
		}
	} else {
		err := fmt.Errorf("Country(%s)is currently not supported", country)
		return nil, err
	}
	return toScrape, nil
}

// number is 'i' from the for loop in buildBindUrls
func firstParameter(number, count int) int {
	// need to add 1 because 'i' starts at 0 which Bing does not understand
	if number == 0 {
		return number +1
	}
	return number*count +1
}

func getScrapeClient(proxyString interface{}) *http.Client {
	switch V := proxyString.(type) {
	case string:
		// parse the proxyString
		proxyUrl, _ := url.Parse(V)
		// proxyUrl becomes the Proxy value which becomes the Transport value that becomes the client
		return &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)}}
	default:
		return &http.Client{}
	}
}

// make a request to the link from buildBingUrls and get something in return
func scrapeClientRequest(searchURL string, proxyString interface{})(*http.Response, error) {
	baseClient := getScrapeClient(proxyString)

	// searchURL comes from buildBingUrls
	req, _ := http.NewRequest("GET", searchURL, nil)
	req.Header.Set("User-Agent", randomUserAgent())

	res, err := baseClient.Do(req)
	if res.StatusCode != 200 {
		err := fmt.Errorf("scraper received a non-200 status code suggesting a ban")
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	return res, nil
}

// receive the results from scrapeClientRequest - unstructured data
// creates the struct for the search result with the SearchResult parameters
func bingResultParser(response *http.Response, rank int)([]SearchResult, error) {
	// takes response and creates a document from that response
	doc, err := goquery.NewDocumentFromResponse(response)
	if err != nil {
		return nil, err
	}

	results := []SearchResult{}
	sel := doc.Find("li.b_algo")
	rank++

	// ranging over all of the node in sel
	for i := range sel.Nodes{
		item := sel.Eq(i)
		linkTag := item.Find("a")
		link, _ := linkTag.Attr("href")
		titleTag := item.Find("h2")
		descTag := item.Find("div.b_caption p")
		desc := descTag.Text()
		title := titleTag.Text()
		link = strings.Trim(link, " ")
		if link != "" && link != "#" && !strings.HasPrefix(link, "/") {
			result := SearchResult{
				// ResultRank
				rank,
				// ResultURL
				link,
				// ResultTitle
				title,
				// ResultDesc
				desc,
			}
			results = append(results, result)
			rank ++
		}
	}
	return results, err
}

// talks to func main
// assembles the results from all the other functions
func BingScrape(searchTerm, country string, proxyString interface{}, pages, count, backoff int)([]SearchResult, error) {
	results := []SearchResult{}

	bingPages, err := buildBingUrls(searchTerm, country, pages, count)

	if err != nil {
		return nil, err
	}

	for _, page := range bingPages{
		rank := len(results)
		res, err := scrapeClientRequest(page, proxyString)
		if err != nil {
			return nil, err
		}
		data, err := bingResultParser(res, rank)
		if err != nil {
			return nil, err
		}
		// range over the data and append to results
		// create a slice of result and send it back
		for _, result := range data {
			results = append(results, result)
		}
		// optional but a best practice  when scraping something or making requests
		// ideally want to use backoff to randomize the time duration between when which you make the requests
		time.Sleep(time.Duration(backoff)*time.Second)
	}
	return results, nil
}

func main() {
	// BingScrape(searchTerm, country, proxyString, pages, count, backoff)
	res, err := BingScrape("github", "com", nil, 1, 30, 10)
	if err == nil {
		for _, res := range res {
			fmt.Println(res)
		}
	} else {
		// a lot of projects use log for errors
		fmt.Println(err)
	}
}