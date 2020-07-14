package controllers

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// NewTranscript will return the specific transcript in the specified language
// or optionally a translated language (tlang).
//
// GET /new?id={videoID}&lang={langCode}&tlang={langCode}
func NewTranscript(c *gin.Context) {
	lang := c.DefaultQuery("lang", "de")
	id := c.DefaultQuery("id", "dL5oGKNlR6I")
	tlang := c.DefaultQuery("tlang", "")

	var transcript Transcript
	err := transcript.getTranscript(lang, id, tlang)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, transcript)
}

// ShowList will return the langCodes of the available transcripts in JSON.
//
// GET /list?id={videoID}
func ShowList(c *gin.Context) {
	id := c.DefaultQuery("id", "dL5oGKNlR6I")
	link := fmt.Sprintf("https://video.google.com/timedtext?v=%s&type=list", id)
	var list TranscriptList
	data, err := getRawData(link)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	err = xml.Unmarshal(data, &list)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

// Line is a struct for each line of a transcript.
type Line struct {
	Text  string `xml:",chardata" json:"text"`
	Start string `xml:"start,attr" json:"start"`
	Dur   string `xml:"dur,attr" json:"dur"`
	End   string `xml:"-" json:"end"`
}

// Transcript is a struct that contains an array of Line
type Transcript struct {
	Lines []Line `xml:"text" json:"lines"`
}

// Video is a struct that has a Transcript and a TranscriptList that
// contains the available lang codes for the specified video.
type Video struct {
	Transcript     Transcript
	TranscriptList TranscriptList
}

// TranscriptList is a struct that contains the available lang codes for the
// video transcripts.
type TranscriptList struct {
	Track []struct {
		LangCode string `xml:"lang_code,attr" json:"langCode"`
	} `xml:"track" json:"track"`
}

func getRawData(link string) ([]byte, error) {
	resp, err := http.Get(link)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, errors.Errorf("Response status: %s", resp.Status)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	data = []byte(strings.ReplaceAll(string(data), "\n", " "))

	return data, nil
}

func (t *Transcript) getTranscript(lang, id, tlang string) error {
	link := fmt.Sprintf("https://video.google.com/timedtext?lang=%s&v=%s", lang, id)
	if tlang != "" {
		link = fmt.Sprintf("https://video.google.com/timedtext?lang=%s&v=%s&tlang=%s", lang, id, tlang)
	}
	data, err := getRawData(link)
	if err != nil {
		return err
	}
	var sub Transcript
	if err := xml.Unmarshal(data, &sub); err != nil {
		return err
	}
	t.Lines = sub.Lines
	for i := range t.Lines {
		tempStart, err := strconv.ParseFloat(t.Lines[i].Start, 64)
		if err != nil {
			return errors.Errorf("Unable to parse tempStart: %v", err)
		}
		tempDur, err := strconv.ParseFloat(t.Lines[i].Dur, 64)
		if err != nil {
			return errors.Errorf("Unable to parse tempStart: %v", err)
		}

		num := tempStart + tempDur
		t.Lines[i].End = strconv.FormatFloat(num, 'f', 2, 64)
	}
	return nil
}
