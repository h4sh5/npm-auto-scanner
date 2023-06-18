package main

import (
	"net/http"
	"encoding/json"
	"io/ioutil"
	"log"
	"time"
)



func main() {
	// follow changes in longpoll mode, include docs so full details are retrieved
	log.Println("started")
	dburl := "https://replicate.npmjs.com/_changes?since=now&feed=longpoll&include_docs=true"
	for { // ever and ever..

		client := http.Client{
		    Timeout: 60 * time.Second,
		}

		// get new changes (since now)
		res,err := client.Get(dburl)
		if err != nil {
			log.Println("Error making GET req:", err)
		}

		// https://irshadhasmat.medium.com/golang-simple-json-parsing-using-empty-interface-and-without-struct-in-go-language-e56d0e69968
		var respData map[string]interface{}

		log.Printf("status code: %d\n", res.StatusCode)

		defer res.Body.Close()

		resBody, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Println("Error reading HTTP response:", err)
		}
		// log.Println("response data:", resBody)

		// decode json
	    err = json.Unmarshal([]byte(resBody), &respData)
	    if err != nil {
	        log.Println("Error unmarshaling JSON:", err)
	    }
	    
	    // log.Println("data unmarshaled:", respData) // prints out json structure

	    // use type assertion to cast
	    // results is a JSON array
	    // if results, ok := respData["results"].([]map[string]interface{}); ok {
	    results := respData["results"].([]interface {})
    	log.Println("num results:", len(results))
    	for _, r := range results {
    		result := r.(map[string]interface{})
    		if result["deleted"] != true {
    			doc := result["doc"].(map[string]interface{})
    			dist_tags := doc["dist-tags"].(map[string]interface{})
    			latest_ver := dist_tags["latest"]
    			log.Println(doc["name"], latest_ver)
    		}
    		
    		// log.Println()
    	}
	    
	}
}