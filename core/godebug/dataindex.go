package godebug

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/jmigpin/editor/core/godebug/debug"
)

// Index data arriving from the server.
type DataIndex struct {
	Counter int
	afds    map[string]*debug.AnnotatorFileData
	index   []DIFile
}

type DIFile []DIDebug
type DIDebug []*DIVersion

// A version of the same msg.
type DIVersion struct {
	Counter int // counter at which the msg was received
	LineMsg *debug.LineMsg
}

func NewDataIndex() *DataIndex {
	return &DataIndex{afds: make(map[string]*debug.AnnotatorFileData)}
}

func (di *DataIndex) AnnotatorFileData(filename string) *debug.AnnotatorFileData {
	afd, ok := di.afds[filename]
	if ok {
		return afd
	}
	return nil
}

func (di *DataIndex) IndexMsg(msg interface{}) error {
	switch t := msg.(type) {
	case *debug.FilesDataMsg:
		logger.Printf("filesdatamsg: %v files", len(t.Data))

		// index files data by filename
		di.afds = make(map[string]*debug.AnnotatorFileData)
		for _, afd := range t.Data {
			logger.Printf("filename: %v", afd.Filename)
			di.afds[afd.Filename] = afd
		}
		// initialize files index
		di.index = make([]DIFile, len(di.afds))
		for _, afd := range di.afds {
			logger.Printf("file %v: %v debugs", afd.FileIndex, afd.DebugLen)
			di.index[afd.FileIndex] = make([]DIDebug, afd.DebugLen)
		}
	case *debug.LineMsg:
		// index msg
		u := &di.index[t.FileIndex][t.DebugIndex]
		*u = append(*u, &DIVersion{di.Counter, t})
		di.Counter++
	default:
		return fmt.Errorf("unexpected msg: %v", reflect.TypeOf(msg))
	}
	return nil
}

// File annotations entries at a version before counter.
func (di *DataIndex) AnnotationsEntries(filename string, counter int) []*DIVersion {
	afd := di.AnnotatorFileData(filename)
	if afd == nil {
		return nil
	}

	// build annotations entries for textarea
	debugs := di.index[afd.FileIndex]
	entries := make([]*DIVersion, len(debugs))
	for i, versions := range debugs {
		// last version that is <= counter

		// search for first version after the counter
		k := sort.Search(len(versions), func(j int) bool {
			return versions[j].Counter > counter
		})
		// first item is bigger then the counter, don't use
		if k == 0 {
			continue
		}
		k--
		entries[i] = versions[k]
	}

	return entries
}

func (di *DataIndex) LineMsgAtCounter(counter int) *debug.LineMsg {
	for _, file := range di.index {
		for _, debug := range file {
			for _, version := range debug {
				if version.Counter == counter {
					return version.LineMsg
				}
			}
		}
	}
	return nil
}

func (di *DataIndex) AnnotatorFileDataFromFileIndex(fileIndex int) *debug.AnnotatorFileData {
	for _, afd := range di.afds {
		if afd.FileIndex == fileIndex {
			return afd
		}
	}
	return nil
}

func (di *DataIndex) LineMsgsBetweenOffsets(filename string, si, ei int) []DIDebug {
	afd := di.AnnotatorFileData(filename)
	if afd == nil {
		return nil
	}

	u := []DIDebug{}
	for _, debug := range di.index[afd.FileIndex] {
		// msgs might not have arrived yet
		if len(debug) == 0 {
			continue
		}
		// any version will do
		version := debug[0]

		lm := version.LineMsg

		o := lm.Offset
		if o >= si && o < ei {
			u = append(u, debug)
		}
	}

	return u
}

func (di *DataIndex) NextDebugVersion(counter int, debug []*DIVersion) (int, bool) {
	if len(debug) == 0 {
		return 0, false
	}
	k := sort.Search(len(debug), func(j int) bool {
		return debug[j].Counter > counter
	})
	if k == len(debug) {
		//return 0, false
		k-- // use last
	}
	return debug[k].Counter, true
}

func (di *DataIndex) PreviousDebugVersion(counter int, debug []*DIVersion) (int, bool) {
	if len(debug) == 0 {
		return 0, false
	}
	k := sort.Search(len(debug), func(j int) bool {
		return debug[j].Counter >= counter
	})
	if k == 0 {
		//return 0, false
		// use first
	} else {
		// if k==len(d.versions) then every entry is smaller than the counter, use last
		k--
	}
	return debug[k].Counter, true
}
