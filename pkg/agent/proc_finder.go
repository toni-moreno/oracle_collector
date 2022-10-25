package agent

import (
	"regexp"

	"github.com/shirou/gopsutil/v3/process"
)

type PID int32

// ProcessFinder uses gopsutil to find processes
type ProcessFinder struct{}

// FullPattern matches on the command line when the process was executed
func (pg *ProcessFinder) FullPattern(pattern string) (map[string]*process.Process, error) {
	pids := make(map[string]*process.Process)
	regxPattern, err := regexp.Compile(pattern)
	if err != nil {
		return pids, err
	}
	procs, err := pg.FastProcessList()
	if err != nil {
		return pids, err
	}
	for _, p := range procs {
		cmd, err := p.Cmdline()
		if err != nil {
			// skip, this can be caused by the pid no longer existing
			// or you having no permissions to access it
			continue
		}
		if regxPattern.MatchString(cmd) {
			match := regxPattern.FindStringSubmatch(cmd)
			result := make(map[string]string)
			for i, name := range regxPattern.SubexpNames() {
				if i != 0 && name != "" {
					result[name] = match[i]
				}
			}
			if len(result) == 1 {
				pids[result["SID"]] = p
			}
		}
	}
	return pids, err
}

func (pg *ProcessFinder) FastProcessList() ([]*process.Process, error) {
	pids, err := process.Pids()
	if err != nil {
		return nil, err
	}

	result := make([]*process.Process, len(pids))
	for i, pid := range pids {
		result[i] = &process.Process{Pid: pid}
	}
	return result, nil
}
