package handlers

import (
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"streamline/utils"
)

type FileHLSDownloadHandler struct {
	StartTime time.Time
	BaseDir   string
}

func (l *FileHLSDownloadHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	utils.GetDownloadLogger().Infof("Received download request\n")
	l.serveDownload(w, req)
}

func (l *FileHLSDownloadHandler) getSourcePath(req *http.Request) string {
	return l.BaseDir
}

func (l *FileHLSDownloadHandler) isFileUploadingDone(file string) bool {
	symlink := file + ".symlink"
	if _, err := os.Stat(symlink); err == nil {
		// exist, then segment uploading is not finished yet
		return false
	}
	// not exist
	return true
}

func (l *FileHLSDownloadHandler) serveDownload(w http.ResponseWriter, req *http.Request) {
	curFileURL := req.URL.EscapedPath()[len("/lhls"):]
	curFilePath := path.Join(l.getSourcePath(req), curFileURL)
	file, err := os.Open(curFilePath) // For read access.
	if err != nil {
		utils.GetDownloadLogger().Errorf("Failed to open file: %v \n", err)
		http.NotFound(w, req)
		return
	}
	defer file.Close()

	utils.GetDownloadLogger().Debugf("file %s was requested @ %v \n", curFileURL, time.Now().Format(time.RFC3339))

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Connection", "Keep-Alive")

	if strings.HasSuffix(curFilePath, ".m3u8") {
		w.Header().Set("Content-Type", "application/x-mpegURL")
	} else if strings.HasSuffix(curFilePath, ".mp4") || strings.HasSuffix(curFilePath, ".fmp4") {
		w.Header().Set("Content-Type", "video/MP4")
	} else {
		w.Header().Set("Content-Type", "video/MP2T")
	}

	w.WriteHeader(http.StatusOK)

	bufferSize := 20480
	buffer := make([]byte, bufferSize)
	var read_err error
	bytesread := 0
	for {
		// start chunk transfer
		for {
			bytesread, read_err = file.Read(buffer)
			if read_err != nil {
				if read_err != io.EOF { // print out if read error
					utils.GetDownloadLogger().Errorf("Failed to read file: %v \n", err)
                                        panic(read_err)
				}
			}

			if bytesread > 0 {
				utils.GetDownloadLogger().Debugf("%s read %d bytes \n", curFileURL, bytesread)
				_, errpr := w.Write(buffer[:bytesread])
				if errpr != nil {
					panic(errpr)
				}
			}

			if bytesread != bufferSize {
				break
			}
		}

		if read_err != nil {
			is_chunk_uploading_done := l.isFileUploadingDone(curFilePath)
			if read_err == io.EOF && is_chunk_uploading_done {
				// if read to end and uploading is done, time to close the downloading too
				break
			}
                        utils.GetDownloadLogger().Debugf("Read to end, but uploading is not finished yet: %v \n", err)
		}
		time.Sleep(50 * time.Millisecond)
	}
	utils.GetDownloadLogger().Debugf("file %s was downloaded @ %v \n", curFileURL, time.Now().Format(time.RFC3339))

}
