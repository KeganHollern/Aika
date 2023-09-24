package voice

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Lucas voice example
// s3://voice-cloning-zero-shot/993f93ee-27f8-42a9-9415-2f316e7a5a5f/luc/manifest.json

type TTSRequest struct {
	Text  string `json:"text"`
	Voice string `json:"voice"`
}
type TTSResponse struct {
	ID      string    `json:"id"`
	Created time.Time `json:"created"`
	Input   struct {
		OutputFormat string `json:"output_format"`
		Quality      string `json:"quality"`
		SampleRate   int    `json:"sample_rate"`
		Seed         any    `json:"seed"`
		Speed        int    `json:"speed"`
		Temperature  any    `json:"temperature"`
		Text         string `json:"text"`
		Voice        string `json:"voice"`
	} `json:"input"`
	Output *struct {
		Duration float64 `json:"duration"`
		Size     int     `json:"size"`
		URL      string  `json:"url"`
	} `json:"output"`
	Links []struct {
		ContentType string `json:"contentType"`
		Description string `json:"description"`
		Href        string `json:"href"`
		Method      string `json:"method"`
		Rel         string `json:"rel"`
	} `json:"_links"`
}

type PlayHT struct {
	User   string
	Secret string
}

func (player PlayHT) TTS(request TTSRequest) (TTSResponse, error) {
	var response TTSResponse
	url := "https://play.ht/api/v2/tts"

	payload, err := json.Marshal(request)
	if err != nil {
		return response, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return response, err
	}

	req.Header.Set("AUTHORIZATION", "Bearer "+player.Secret)
	req.Header.Set("X-USER-ID", player.User)
	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return response, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return response, err
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return response, err
	}

	for response.Output == nil {
		response, err = player.pollJob(response.ID)
		if err != nil {
			return response, err
		}
	}

	return response, nil
}

func (player PlayHT) pollJob(id string) (TTSResponse, error) {
	var response TTSResponse
	url := "https://play.ht/api/v2/tts/" + id

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return response, err
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("AUTHORIZATION", "Bearer "+player.Secret)
	req.Header.Set("X-USER-ID", player.User)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return response, err
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return response, err
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return response, err
	}

	return response, nil
}

// DownloadMP3 downloads an MP3 from the provided URL and saves it to the specified directory with the given fileName.
func DownloadMP3(url, directory, fileName string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download: %v", resp.Status)
	}

	// Create the directory if it doesn't exist
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		os.MkdirAll(directory, 0755)
	}

	// Open a file for writing
	out, err := os.Create(filepath.Join(directory, fileName))
	if err != nil {
		return err
	}
	defer out.Close()

	// Copy data from HTTP response to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
