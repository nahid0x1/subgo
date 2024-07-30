package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func main() {
	domain := flag.String("d", "", "input domain")
	outputFile := flag.String("o", "", "output file")
	flag.Parse()

	if *domain == "" {
		log.Fatal("Please provide a domain using the -d flag")
	}

	if *outputFile == "" {
		log.Fatal("Please provide an output file using the -o flag")
	}

	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(*outputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatal(err)
	}

	// Get subdomains from jldc.me
	jldcURL := fmt.Sprintf("https://jldc.me/anubis/subdomains/%s", *domain)
	var jldcSubdomains []string
	for retry := 0; retry < 3; retry++ {
		resp, err := http.Get(jldcURL)
		if err != nil {
			log.Printf("Error getting jldc subdomains: %s", err)
			time.Sleep(time.Second)
			continue
		}
		defer resp.Body.Close()

		jldcBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading jldc response: %s", err)
			continue
		}

		jldcRegex := regexp.MustCompile(`https?://[^/]+|www\.[^/]+|[^/]+\.com`)
		jldcSubdomains = jldcRegex.FindAllString(string(jldcBody), -1)
		break
	}

	// Get subdomains from crt.sh
	crtURL := fmt.Sprintf("https://crt.sh/?q=%%25.%s&output=json", *domain)
	var crtData []map[string]interface{}
	for retry := 0; retry < 3; retry++ {
		resp, err := http.Get(crtURL)
		if err != nil {
			log.Printf("Error getting crt subdomains: %s", err)
			time.Sleep(time.Second)
			continue
		}
		defer resp.Body.Close()

		crtBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading crt response: %s", err)
			continue
		}

		err = json.Unmarshal(crtBody, &crtData)
		if err != nil {
			log.Printf("Error unmarshaling crt response: %s", err)
			continue
		}
		break
	}

	crtSubdomains := make([]string, 0)
	for _, entry := range crtData {
		nameValue, ok := entry["name_value"].(string)
		if ok {
			nameValue = strings.Replace(nameValue, "*.", "", 1)
			nameValue = filterUnwantedChars(nameValue)
			crtSubdomains = append(crtSubdomains, nameValue)
		}
	}

	// Combine subdomains and write to output file
	allSubdomains := append(jldcSubdomains, crtSubdomains...)
	allSubdomains = removeDuplicates(allSubdomains)

	outputFileHandle, err := os.Create(*outputFile)
	if err != nil {
		log.Fatal(err)
	}
	defer outputFileHandle.Close()

	for _, subdomain := range allSubdomains {
		fmt.Fprintln(outputFileHandle, subdomain)
	}
}

func filterUnwantedChars(s string) string {
	// Remove unwanted characters
	s = strings.Replace(s, "\341\205\232\341\205\232\341\205\232\341\205\232\341\205\232\341\205\232\341\205\232\341\205\232\341\205\232\341\205\232\341\205\232\341\205\232\341\205\232\341\205\232\341\205\232\341\205\232\341\205\232\341\205\232\341\205\232\341\205\232\341\205\232\341\205\232\341\205\232\341\205\232\341\205\232\341\205\232\341\205\232\341\205\232\341\205\232", "", -1)
	s = strings.Replace(s, "*.", "", -1)
	return s
}

func removeDuplicates(s []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)
	for _, str := range s {
		if !seen[str] {
			seen[str] = true
			result = append(result, str)
		}
	}
	return result
}
