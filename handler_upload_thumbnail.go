package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"mime"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}


	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// TODO: implement the upload here
	const maxMemory = 10 << 20 // 10 MB
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return 
	}
	defer file.Close()

	mediaType := header.Header.Get("Content-Type")
	mediatype, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Content-Type", err)
		return 
	}
	if mediatype != "image/png" && mediatype != "image/jpeg" {
		respondWithError(w, http.StatusUnsupportedMediaType, "Only PNG and JPEG images are supported", nil)
		return 
	}

	// Determine extension based on validated media type
	extension := "jpg"
	if mediatype == "image/png" {
		extension = "png"
	}
	
	// Generate random filename
	randBytes := make([]byte, 32)
	if _, err := rand.Read(randBytes); err != nil{
		respondWithError(w, http.StatusInternalServerError, "Error generating random filename", err)
		return 
	}
	randomName := base64.RawURLEncoding.EncodeToString(randBytes)
	filename := randomName + "." + extension
	filePath := filepath.Join(cfg.assetsRoot, filename)
	thumbnailURL := "/assets/" + filename


	outFile, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating file", err)
		return 
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, file); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error saving file", err)
		return 
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Video not found", err)
		return 
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "User not authorized", nil)
		return 
	}

	video.ThumbnailURL = &thumbnailURL

	if err := cfg.db.UpdateVideo(video); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error updating video", err)
		return 
	}

	respondWithJSON(w, http.StatusOK, video)
}
