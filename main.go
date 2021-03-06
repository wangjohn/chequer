package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/zenazn/goji"
)

const (
	tmpPrefix         = "chequer_image"
	resultPrefix      = "tesseract_result"
	tesseractLanguage = "mcr"
)

func main() {
	serviceGojiRoutes()
}

func serviceGojiRoutes() {
	goji.Post("/cheque", PostCheque)
	goji.Serve()
}

type ChequeRequest struct {
	ImageURL string `json:"image_url"`
}

func PostCheque(w http.ResponseWriter, r *http.Request) {
	multiReader, err := r.MultipartReader()
	var chequeResult *ChequeResult
	if err != nil {
		chequeResult, err = processNonMultipartChequeRequest(r)
	} else {
		chequeResult, err = processMultipartChequeRequest(multiReader)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	responseData, err := json.Marshal(chequeResult)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(responseData)
}

func processNonMultipartChequeRequest(r *http.Request) (*ChequeResult, error) {
	decoder := json.NewDecoder(r.Body)
	var chequeRequest ChequeRequest
	decodeErr := decoder.Decode(&chequeRequest)

	if decodeErr != nil {
		return nil, decodeErr
	}

	httpResponse, err := http.Get(chequeRequest.ImageURL)
	if err != nil {
		return nil, err
	}
	defer httpResponse.Body.Close()

	tmpFile, err := ioutil.TempFile(os.TempDir(), tmpPrefix)
	if err != nil {
		return nil, err
	}
	defer removeTempFile(tmpFile)

	_, err = io.Copy(tmpFile, httpResponse.Body)
	return ProcessCheque(tmpFile)
}

func processMultipartChequeRequest(multiReader *multipart.Reader) (*ChequeResult, error) {
	var chequeResult *ChequeResult
	partCount := 0

	for {
		part, partErr := multiReader.NextPart()
		partCount++

		if partErr == io.EOF {
			return chequeResult, nil
		} else if partCount > 1 {
			return chequeResult, fmt.Errorf("Multiple parts found. Currently we " +
				"only support a single part.")
		} else if partErr != nil {
			return chequeResult, partErr
		} else {
			p, err := ioutil.ReadAll(part)
			if err != nil {
				return chequeResult, err
			}

			tmpFile, err := ioutil.TempFile(os.TempDir(), tmpPrefix)
			if err != nil {
				return nil, err
			}
			defer removeTempFile(tmpFile)

			ioutil.WriteFile(tmpFile.Name(), p, 0644)
			chequeResult, err = ProcessCheque(tmpFile)
		}
	}

	// This should never be called
	return nil, fmt.Errorf("Unable to find any multipart parts.")
}

func ProcessCheque(img *os.File) (*ChequeResult, error) {
	outFile, err := ioutil.TempFile(os.TempDir(), resultPrefix)
	if err != nil {
		return nil, err
	}

	options := []string{
		img.Name(),
		outFile.Name(),
		"-l",
		tesseractLanguage,
	}
	defer removeTempFile(outFile)

	cmd := exec.Command("tesseract", options...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error calling tesseract: %v\n%s", err, output)
		return nil, err
	}

	tessFile, err := os.Open(outFile.Name() + ".txt")
	if err != nil {
		return nil, err
	}
	defer removeTempFile(tessFile)

	return ProcessTesseractOutput(tessFile)
}

func ProcessTesseractOutput(outFile *os.File) (*ChequeResult, error) {
	scanner := bufio.NewScanner(outFile)
	var micrLine string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "@") {
			micrLine = line
		}
	}

	log.Println(micrLine)
	result := ChequeResult{
		Account: findAccountNumber(micrLine),
		Routing: findRoutingNumber(micrLine),
	}
	log.Println(result)
	return &result, nil
}

func findAccountNumber(micrLine string) string {
	regexes := []string{
		"@(\\d+)@?",
		"@?(\\d+)@",
	}
	for _, regexStr := range regexes {
		re := regexp.MustCompile(regexStr)
		match := re.FindStringSubmatch(micrLine)

		if len(match) > 0 {
			return match[1]
		}
	}

	return ""
}

func findRoutingNumber(micrLine string) string {
	regexes := []string{
		"\\[([\\d-]+)\\[?",
		"\\[?([\\d-]+)\\[",
	}
	for _, regexStr := range regexes {
		re := regexp.MustCompile(regexStr)
		match := re.FindStringSubmatch(micrLine)

		if len(match) > 0 {
			return match[1]
		}
	}

	return ""
}

func removeTempFile(tmpFile *os.File) {
	tmpFile.Close()
	os.Remove(tmpFile.Name())
}

type ChequeResult struct {
	Account string `json:"account"`
	Routing string `json:"routing"`
}
