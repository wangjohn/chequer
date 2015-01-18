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
  if err != nil {
    // They didn't send multipart data, so we'll attempt to parse as JSON
    decoder := json.NewDecoder(r.Body)
    var chequeRequest ChequeRequest
    decodeErr := decoder.Decode(&chequeRequest)

    if decodeErr != nil {
      http.Error(w, decodeErr.Error(), http.StatusBadRequest)
      return
    }

    processChequeFromRequest(chequeRequest)
  } else {
    parts := make([][]byte, 0)
    for {
      part, partErr := multiReader.NextPart()
      if partErr == io.EOF {
        processMultiparts(parts)
      } else {
        p, err := ioutil.ReadAll(part)
        if err != nil {
          http.Error(w, err.Error(), http.StatusBadRequest)
          return
        }
        parts = append(parts, p)
      }
      if partErr != nil {
        http.Error(w, partErr.Error(), http.StatusBadRequest)
        return
      }
    }
  }
}

func processChequeFromRequest(request ChequeRequest) {
  // TODO: do something
}

func processMultiparts(parts [][]byte) (error, *ChequeResult) {
  for _, p := range parts {
    tmpFile, err := ioutil.TempFile(os.TempDir(), tmpPrefix)
    if err != nil {
      return err, nil
    }
    tmpFile.Close()

    ioutil.WriteFile(tmpFile.Name(), p, 0644)

    return ProcessCheque(tmpFile)
  }
  return nil, nil
}

func ProcessCheque(img *os.File) (error, *ChequeResult) {
  outFile, err := ioutil.TempFile(os.TempDir(), resultPrefix)
  if err != nil {
    return err, nil
  }

  options := []string{
    img.Name(),
    outFile.Name(),
    "-l",
    tesseractLanguage,
  }

  err = outFile.Close()
  if err != nil {
    return err, nil
  }

  cmd := exec.Command("tesseract", options...)
  output, err := cmd.CombinedOutput()
  if err != nil {
    log.Printf("Error calling tesseract: %v\n%s", err, output)
    return err, nil
  }

  outFile, err = os.Open(outFile.Name() + ".txt")
  if err != nil {
    return err, nil
  }

  return ProcessTesseractOutput(outFile)
}

func ProcessTesseractOutput(outFile *os.File) (error, *ChequeResult) {
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
  return nil, &result
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
  Account string
  Routing string
}

