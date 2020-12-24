package box

import (
	"bufio"
	"io"
	"log"
	"strconv"
	"strings"
)

// MetaFile holds all data of a meta file
// These are arranged in the order of the manual: https://www.ucw.cz/moe/isolate.1.html
type MetaFile struct {
	CgEnabled   bool `json:"cg-enabled"`
	CgMem       int  `json:"cg-mem"`
	CgOOMKilled bool `json:"cg-oom-killed"`

	ForcedCSWs    int `json:"csw-forced"`
	VoluntaryCSWs int `json:"csw-voluntary"`

	ExitCode   int `json:"exitcode"`
	ExitSignal int `json:"exitsig"`

	Killed bool `json:"killed"`

	MaxRSS int `json:"max-rss"`

	Message string `json:"message"`
	Status  string `json:"status"`

	Time     float64 `json:"time"`
	WallTime float64 `json:"time-wall"`
}

// ParseMetaFile parses a specified meta file
func ParseMetaFile(r io.Reader) *MetaFile {
	var file = new(MetaFile)

	s := bufio.NewScanner(r)

	for s.Scan() {
		if !strings.Contains(s.Text(), ":") {
			continue
		}
		l := strings.SplitN(s.Text(), ":", 2)
		switch l[0] {
		case "cg-mem":
			file.CgMem, _ = strconv.Atoi(l[1])
		case "cg-oom-killed":
			file.CgOOMKilled = true
		case "csw-forced":
			file.ForcedCSWs, _ = strconv.Atoi(l[1])
		case "csw-voluntary":
			file.VoluntaryCSWs, _ = strconv.Atoi(l[1])
		case "exitcode":
			file.ExitCode, _ = strconv.Atoi(l[1])
		case "exitsig":
			file.ExitSignal, _ = strconv.Atoi(l[1])
		case "killed":
			file.Killed = true
		case "max-rss":
			file.MaxRSS, _ = strconv.Atoi(l[1])
		case "message":
			file.Message = l[1]
		case "status":
			file.Status = l[1]
		case "time":
			file.Time, _ = strconv.ParseFloat(l[1], 32)
		case "time-wall":
			file.WallTime, _ = strconv.ParseFloat(l[1], 32)
		case "cg-enabled":
			if l[1] == "true" {
				file.CgEnabled = true
			}
		default:
			log.Printf("Unknown meta variable %s with value %s\n", l[0], l[1])
		}
	}

	return file
}
