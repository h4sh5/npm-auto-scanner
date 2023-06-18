package main

import (
	"net/http"
	"encoding/json"
	"io/ioutil"
	"log"
	"time"
	"os/exec"
	"strings"
	"strconv"
	"bytes"
	"os"
)

type GithubIssue struct {
	// objects must be exported, since encoding/json package and similar packages ignore unexported fields.
	// that means fields must be capitalized. golang footgun

	Title string `json:"title"`
	Body string `json:"body"`
	Labels []string `json:"labels"`
}

// https://stackoverflow.com/a/41516687
// RunCMD is a simple wrapper around terminal commands
func RunCMD(path string, args []string, debug bool) (out string, err error) {

	start := time.Now()
    cmd := exec.Command(path, args...)

    var b []byte
    b, err = cmd.CombinedOutput()
    out = string(b)

    if debug {

        log.Println(strings.Join(cmd.Args[:], " "), "command finished in:",time.Now().Sub(start))
        if err != nil {
            log.Println("RunCMD ERROR")
        }
        log.Println(out)
    }

    // returns both out and err by default since its defined in func
    return
}

func create_github_issue(jsonBody []byte) {
	url := "https://api.github.com/repos/h4sh5/npm-auto-scanner/issues"
	bodyReader := bytes.NewReader(jsonBody)
	req, err := http.NewRequest(http.MethodPost, url, bodyReader)
	req.Header.Add("Accept", "application/vnd.github+json")
	req.Header.Add("Authorization", "Bearer " + string(os.Getenv("GITHUB_TOKEN")))
	req.Header.Add("X-GitHub-Api-Version", "2022-11-28")
	res, err := http.DefaultClient.Do(req)
	resBody, err := ioutil.ReadAll(res.Body)
	log.Println("github issue create resp:", string(resBody))

	if err != nil {
		log.Println("create_github_issue http req err:",err)
	}
	
	if res.StatusCode != 201 && res.StatusCode != 200 {
		log.Println("create_github_issue bad http code:", res.StatusCode)
	}

}

func raise_guarddog_issues(name string, version string, guarddog_json_out string) {
	var item map[string]interface {}
	err := json.Unmarshal([]byte(guarddog_json_out), &item)
	if err != nil {
		log.Println("error unmarshaling guarddog res:", err)
	}
	
	// result := item["results"].(map[string]interface{})
	if item["issues"] == nil {
		log.Println("issues is nil. skipping item:", item)
		return
	}
	issue_count := item["issues"].(float64)
	log.Println(name, version, "has", issue_count, "issues")
	var labels []string
    // delete empty issues
    if  item["results"] == nil {
    	log.Println("results is nil. skipping item:", item)
		return
    }
    results := item["results"].(map[string]interface{})
    for key, val := range results {
    	if valmap, ok := val.(map[string]interface{}); ok {
    		if len(valmap) == 0 {
		    	delete(results, key)
		    }
    	}
	    
	    if _, ok := val.(string); ok {
	    	labels = append(labels, key)
	    }
	}


	resultsStr, err := json.Marshal(results)
	if err != nil {
		log.Println("error marshaling results:",err)
		return
	}
	ghissue := GithubIssue{
		Title: name + " " + version + " has " + strconv.FormatFloat(issue_count, 'f', -1, 64) + " guarddog issues",
		Labels: labels,
		Body: "```" + string(resultsStr) + "```",
	}
	// log.Println("ghissue before marshaling:", ghissue)
	issueBodyStr,err := json.Marshal(ghissue)
	if err != nil {
		log.Println("error marshaling github_issue_body:",err)
		return
	}
	if issue_count > 0 {
		log.Println("creating github issue:", string(issueBodyStr))
		create_github_issue([]byte(issueBodyStr))
	}
	

}

func process_pkg(name string, version string) {
	log.Println("process_pkg",name,version)
	bin := "guarddog"
	args := []string{"npm", "scan", "--output-format=json", "-x release_zero", "-x repository_integrity_mismatch", "-x single_python_file", "-x empty_information", name, "--version", version}
	out, _ := RunCMD(bin, args, true)
	raise_guarddog_issues(name, version, out)
}

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
    			go process_pkg(doc["name"].(string), latest_ver.(string))
    		}
    		
    		// log.Println()
    	}
	    
	}
}