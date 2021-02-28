package box

import (
	"bufio"
	"io"
	"strconv"
	"strings"

	"github.com/KiloProjects/kilonova"
)

// ParseMetaFile parses a specified meta file
func ParseMetaFile(r io.Reader) *kilonova.RunStats {
	if r == nil {
		return nil
	}
	var file = new(kilonova.RunStats)

	s := bufio.NewScanner(r)

	for s.Scan() {
		if !strings.Contains(s.Text(), ":") {
			continue
		}
		l := strings.SplitN(s.Text(), ":", 2)
		switch l[0] {
		case "cg-mem":
			file.Memory, _ = strconv.Atoi(l[1])
		case "exitcode":
			file.ExitCode, _ = strconv.Atoi(l[1])
		case "exitsig":
			file.ExitSignal, _ = strconv.Atoi(l[1])
		case "killed":
			file.Killed = true
		case "message":
			file.Message = l[1]
		case "status":
			file.Status = l[1]
		case "time":
			file.Time, _ = strconv.ParseFloat(l[1], 32)
		case "time-wall":
			file.WallTime, _ = strconv.ParseFloat(l[1], 32)
		default:
			continue
		}
	}

	return file
}
