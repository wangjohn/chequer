package chequer

import (
  "http"
  "json"
  "os"
  "io/ioutil"
  "mime/multipart"

  "github.com/zenazn/goji"
)

const (
  prefix = "tesseract"
  tesseractLanguage = "msr"
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
  if err == nil {
    // They didn't send multipart data, so we'll attempt to parse as JSON
    decoder := json.NewDecoder(r.Body)
    var chequeRequest ChequeRequest
    decodeErr := decoder.Decode(&chequeRequest)

    if decodeErr != nil {
      http.Error(w, decodeErr.Error(), http.StatusBadRequest)
      return
    }

    return processChequeFromRequest(chequeRequest)
  } else {
    parts := make([]multipart.Part, 0)
    for {
      part, partErr := multiReader.NextPart()
      parts = append(parts, part)
      if partErr == io.EOF {
        return processMultiparts(parts)
      }
      if partErr != nil {
        http.Error(w, partError.Error(), http.StatusBadRequest)
        return
      }
    }
  }
}

func processChequeFromRequest(request ChequeRequest) {
  // TODO: do something
}

func processMultiparts(parts []multipart.Part) {
  // TODO: do sometihng
}

func ProcessCheque(img *os.File) (error, *ChequeResult) {
  outFile, err := ioutil.TempFile(os.TempDir(), prefix)
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
    log.Printf("Error calling tesseract: %v\n%v", err, output)
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

  return *ChequeResult{
    Account: findAccountNumber(micrLine),
    Routing: findRoutingNumber(micrLine),
  }
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

