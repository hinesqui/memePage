package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"io"
	"log"
	"memePage/app/conf"
	"net/http"
	"os"
	"time"
)

const (
	Today   MongoPeriod = "today"
	Week                = "week"
	Month               = "month"
	AllTime             = "all"
)

type (
	FilePathResp struct {
		Ok     bool `json:"ok"`
		Result struct {
			FileId       string `json:"file_id"`
			FileUniqueId string `json:"file_unique_id"`
			FileSize     int    `json:"file_size"`
			FilePath     string `json:"file_path"`
		} `json:"result"`
	}

	FailFilePathResp struct {
		Ok          bool   `json:"ok"`
		ErrorCode   int    `json:"error_code"`
		Description string `json:"description"`
	}

	Post struct {
		LikesCount    int
		DislikesCount int
		FileId        string
		TgPath        string
	}

	MongoPeriod string
)

func CheckFiles(posts []bson.M) {
	for _, post := range posts {
		filePath := fmt.Sprintf("%s/%s", conf.AppConf.ContentPath, post["file_id"])
		if !IsFileExist(filePath) {
			postTgUrl := make(chan string)
			go GetFilePath(post["file_id"].(string), postTgUrl)
			go DownloadTgFile(<-postTgUrl, filePath)
			close(postTgUrl)
		}
	}
}

func GetFilePath(postId string, tgUrl chan string) {
	var client http.Client

	reqUrl := fmt.Sprintf(
		"https://api.telegram.org/bot%s/getFile?file_id=%s",
		conf.Tg.Token,
		postId)

	res, err := client.Get(reqUrl)
	defer res.Body.Close()
	if err != nil {
		log.Printf("could not create request: %s\n", err)
	}

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("could not read response body: %s\n", err)
	}

	if res.StatusCode != http.StatusOK {
		parsedResp := FailFilePathResp{}
		json.Unmarshal(respBody, &parsedResp)
		log.Printf("Failed to get file path %s", parsedResp.Description)
	} else {
		parsedResp := FilePathResp{}
		json.Unmarshal(respBody, &parsedResp)

		tgUrl <- parsedResp.Result.FilePath
	}

}

func DownloadTgFile(fileUrl string, filePath string) {
	var client http.Client
	reqUrl := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", conf.Tg.Token, fileUrl)
	resp, err := client.Get(reqUrl)
	defer resp.Body.Close()
	if err != nil {
		log.Printf("could not create request: %s\n", err)
	}

	out, err := os.Create(filePath)
	defer out.Close()
	if err != nil {
		log.Printf("Failed to create file: %s\n", err)
	}

	_, err = io.Copy(out, resp.Body)
}

func IsFileExist(fileName string) bool {
	_, err := os.Stat(fileName)

	if os.IsNotExist(err) {
		return false
	} else {
		return true
	}
}

func (r MongoPeriod) GetSearchPeriodParams() (int64, primitive.DateTime, error) {
	now := time.Now()
	switch r {
	case Today:
		return 1,
			primitive.NewDateTimeFromTime(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)),
			nil
	case Week:
		return 10,
			primitive.NewDateTimeFromTime(
				time.Date(now.Year(), now.Month(), now.Day()-int(now.Weekday())+1, 0, 0, 0, 0, time.Local)),
			nil
	case Month:
		return 10,
			primitive.NewDateTimeFromTime(time.Date(now.Year(), now.Month(), 0, 0, 0, 0, 0, time.Local)),
			nil
	case AllTime:
		return 10,
			primitive.NewDateTimeFromTime(time.Date(0, 0, 0, 0, 0, 0, 0, time.Local)),
			nil
	default:
		return 0, 0, errors.New("unsupported period")
	}
}

func GetUrls(posts []primitive.M) []string {
	urls := make([]string, len(posts))

	for i, post := range posts {
		urls[i] = fmt.Sprintf("%s%s", conf.AppConf.ContentPath, post["file_id"])
	}

	return urls
}
