package main

import (
  "net/http"
  "encoding/json"
  "os"
  "io"
  "os/exec"
  "regexp"
  "log"
  "bufio"
  "strings"
  "io/ioutil"
  "mime/multipart"
  "fmt"

  "github.com/zenazn/goji"
)

const (
  tmpPrefix = "chequer_image"
  resultPrefix = "tesseract_result"
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

  return nil, nil
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
      tmpFile.Close()

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

  err = outFile.Close()
  if err != nil {
    return nil, err
  }

  cmd := exec.Command("tesseract", options...)
  output, err := cmd.CombinedOutput()
  if err != nil {
    log.Printf("Error calling tesseract: %v\n%s", err, output)
    return nil, err
  }

  outFile, err = os.Open(outFile.Name() + ".txt")
  if err != nil {
    return nil, err
  }

  return ProcessTesseractOutput(outFile)
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

  result := ChequeResult{
    Account: findAccountNumber(micrLine),
    Routing: findRoutingNumber(micrLine),
  }
  log.Println(result)
  return &result, nil
}

func findAccountNumber(micrLine string) (string) {
  re := regexp.MustCompile("@(\\d+)@")
  match := re.FindStringSubmatch(micrLine)

  if len(match) > 0 {
    return match[1]
  }

  return ""
}

func findRoutingNumber(micrLine string) (string) {
  re := regexp.MustCompile("\\[([\\d-]+)\\[?")
  match := re.FindStringSubmatch(micrLine)

  if len(match) > 0 {
    return match[1]
  }

  return ""
}

type ChequeResult struct {
  Account string `json:"account"`
  Routing string `json:"routing"`
}
