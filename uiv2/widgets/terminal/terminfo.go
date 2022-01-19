package terminal

import (
	"sync"

	"github.com/gdamore/tcell/v2/terminfo"
	"github.com/gdamore/tcell/v2/terminfo/dynamic"
)

var cachedTerminfo map[string]*terminfo.Terminfo
var cachedTerminfoMutex sync.Mutex

func init() {
	cachedTerminfo = make(map[string]*terminfo.Terminfo)
}

// findTerminfo returns a terminfo struct via tcell's dynamic method first,
// then using the built-in databases. The aim is to use the terminfo database
// most likely to be correct. Maybe even better would be parsing the terminfo
// file directly using something like https://github.com/beevik/terminfo/, to
// avoid the extra process.
func findTerminfo(name string) (*terminfo.Terminfo, error) {
	cachedTerminfoMutex.Lock()
	if ti, ok := cachedTerminfo[name]; ok {
		cachedTerminfoMutex.Unlock()
		return ti, nil
	}
	ti, _, e := dynamic.LoadTerminfo(name)
	if e == nil {
		cachedTerminfo[name] = ti
		cachedTerminfoMutex.Unlock()
		return ti, nil
	}
	ti, e = terminfo.LookupTerminfo(name)
	return ti, e
}
