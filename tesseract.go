package chequer

import (
  "http"
  "json"

  "github.com/zenazn/goji"
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
    for {
      part, partErr := multiReader.NextPart()
      if partErr == io.EOF {
        // TODO: finish up and return something good
      }
      if partErr != nil {
        http.Error(w, partError.Error(), http.StatusBadRequest)
        return
      }
    }

  } else {
    // They didn't send multipart data, so we'll attempt to parse as JSON
    decoder := json.NewDecoder(r.Body)
    var chequeRequest ChequeRequest
    decodeErr := decoder.Decode(&chequeRequest)

    if err != nil {
      http.Error(w, decodeErr.Error(), http.StatusBadRequest)
      return
    }
  }
}
