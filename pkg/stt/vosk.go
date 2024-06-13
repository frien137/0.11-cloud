package stt

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	sr "github.com/kercre123/0.11-cloud/pkg/speechrequest"
	"github.com/kercre123/0.11-cloud/pkg/vars"
	vosk "github.com/kercre123/vosk-api/go"
)

var GrammerEnable bool = false

var Name string = "vosk"

var model *vosk.VoskModel
var recsmu sync.Mutex

var grmRecs []ARec
var gpRecs []ARec

var modelLoaded bool

type ARec struct {
	InUse bool
	Rec   *vosk.VoskRecognizer
}

var Grammer string

func Init() error {
	if os.Getenv("VOSK_WITH_GRAMMER") == "true" {
		fmt.Println("Initializing vosk with grammer optimizations")
		GrammerEnable = true
	}
	vosk.SetLogLevel(-1)
	if modelLoaded {
		fmt.Println("A model was already loaded, freeing all recognizers and model")
		for ind, _ := range grmRecs {
			grmRecs[ind].Rec.Free()
		}
		for ind, _ := range gpRecs {
			gpRecs[ind].Rec.Free()
		}
		gpRecs = []ARec{}
		grmRecs = []ARec{}
		model.Free()
	}
	sttLanguage := vars.Language
	if len(sttLanguage) == 0 {
		sttLanguage = "en-US"
	}
	modelPath := os.Getenv(vars.VoskModelPathEnv)
	if _, err := os.Stat(modelPath); err != nil {
		fmt.Println("Path does not exist: " + modelPath)
		return err
	}
	fmt.Println("Opening VOSK model (" + modelPath + ")")
	aModel, err := vosk.NewModel(modelPath)
	if err != nil {
		log.Fatal(err)
		return err
	}
	model = aModel

	fmt.Println("Initializing VOSK recognizers")
	gpRecognizer, err := vosk.NewRecognizer(aModel, 16000.0)
	var gprec ARec
	gprec.Rec = gpRecognizer
	gprec.InUse = false
	gpRecs = append(gpRecs, gprec)
	if err != nil {
		log.Fatal(err)
	}
	modelLoaded = true
	fmt.Println("VOSK initiated successfully")
	runTest()
	return nil
}

func runTest() {
	// make sure recognizer is all loaded into RAM
	fmt.Println("Running recognizer test")
	var withGrm bool
	if GrammerEnable {
		fmt.Println("Using grammer-optimized recognizer")
		withGrm = true
	} else {
		fmt.Println("Using general recognizer")
		withGrm = false
	}
	rec, recind := getRec(withGrm)
	sttTestPath := "./stttest.pcm"
	pcmBytes, _ := os.ReadFile(sttTestPath)
	var micData [][]byte
	cTime := time.Now()
	micData = sr.SplitVAD(pcmBytes)
	for _, sample := range micData {
		rec.AcceptWaveform(sample)
	}
	var jres map[string]interface{}
	json.Unmarshal([]byte(rec.FinalResult()), &jres)
	if withGrm {
		grmRecs[recind].InUse = false
	} else {
		gpRecs[recind].InUse = false
	}
	transcribedText := jres["text"].(string)
	tTime := time.Now().Sub(cTime)
	fmt.Println("Text (from test):", transcribedText)
	if tTime.Seconds() > 3 {
		fmt.Println("Vosk test took a while, performance may be degraded. (" + fmt.Sprint(tTime) + ")")
	}
	fmt.Println("Vosk test successful! (Took " + fmt.Sprint(tTime) + ")")
}

func getRec(withGrm bool) (*vosk.VoskRecognizer, int) {
	recsmu.Lock()
	defer recsmu.Unlock()
	if withGrm && GrammerEnable {
		for ind, rec := range grmRecs {
			if !rec.InUse {
				grmRecs[ind].InUse = true
				return grmRecs[ind].Rec, ind
			}
		}
	} else {
		for ind, rec := range gpRecs {
			if !rec.InUse {
				gpRecs[ind].InUse = true
				return gpRecs[ind].Rec, ind
			}
		}
	}
	recsmu.Unlock()
	var newrec ARec
	var newRec *vosk.VoskRecognizer
	var err error
	newrec.InUse = true
	if withGrm {
		newRec, err = vosk.NewRecognizerGrm(model, 16000.0, Grammer)
	} else {
		newRec, err = vosk.NewRecognizer(model, 16000.0)
	}
	if err != nil {
		log.Fatal(err)
	}
	newrec.Rec = newRec
	recsmu.Lock()
	if withGrm {
		grmRecs = append(grmRecs, newrec)
		return grmRecs[len(grmRecs)-1].Rec, len(grmRecs) - 1
	} else {
		gpRecs = append(gpRecs, newrec)
		return gpRecs[len(gpRecs)-1].Rec, len(gpRecs) - 1
	}
}

func STT(req sr.SpeechRequest) (string, error) {
	fmt.Println("(Bot " + req.Device + ", Vosk) Processing...")
	var withGrm bool
	rec, recind := getRec(false)
	rec.SetWords(1)
	//rec.AcceptWaveform(req.FirstReq)
	req.DetectEndOfSpeech()
	var toldUser bool
	for {
		chunk, err := req.GetNextStreamChunk()
		if req.ChunkSkips == 6 {
			if !toldUser {
				fmt.Println("past 6 skipped audio chunks")
				toldUser = true
			}
			if err != nil {
				return "", err
			}
			speechIsDone, doProcess := req.DetectEndOfSpeech()
			if doProcess {
				rec.AcceptWaveform(chunk)
			}
			if speechIsDone {
				break
			}
		} else {
			req.ChunkSkips = req.ChunkSkips + 1
		}
	}
	var jres map[string]interface{}
	json.Unmarshal([]byte(rec.FinalResult()), &jres)
	if withGrm {
		grmRecs[recind].InUse = false
	} else {
		gpRecs[recind].InUse = false
	}
	transcribedText := jres["text"].(string)
	fmt.Println("Bot " + req.Device + " Transcribed text: " + transcribedText)
	return transcribedText, nil
}
