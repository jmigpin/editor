package lsproto

import (
	"context"
	"fmt"
	"sync"

	"github.com/jmigpin/editor/util/ctxutil"
)

type LangManager struct {
	Reg *Registration // accessed from editor

	man *Manager
	mu  struct {
		sync.Mutex
		li             *LangInstance
		cancelInstance context.CancelFunc
	}
}

func NewLangManager(man *Manager, reg *Registration) *LangManager {
	return &LangManager{Reg: reg, man: man}
}

func (lang *LangManager) instance(reqCtx context.Context) (*LangInstance, error) {
	lang.mu.Lock()
	defer lang.mu.Unlock()

	if lang.mu.li != nil {
		return lang.mu.li, nil
	}

	// setup instance context // TODO: manager ctx
	ctx, cancel := context.WithCancel(context.Background())

	// call cancel if reqCtx is done
	clearWatching := ctxutil.WatchDone(cancel, reqCtx)
	defer clearWatching()

	li, err := NewLangInstance(ctx, lang)
	if err != nil {
		cancel()
		err = lang.WrapError(err)
		return nil, err
	}

	// handle server/client abnormal early exit
	go func() {
		defer cancel()
		if err := li.Wait(); err != nil {
			lang.PrintWrapError(err)
		}
		// ensure this instance is cleared
		lang.mu.Lock()
		defer lang.mu.Unlock()
		if lang.mu.li == li {
			lang.mu.li = nil
		}
	}()

	lang.mu.li = li
	lang.mu.cancelInstance = cancel

	return li, nil
}

// returns true if the instance was running
func (lang *LangManager) Close() (error, bool) {
	lang.mu.Lock()
	defer lang.mu.Unlock()
	if lang.mu.li != nil {
		lang.mu.cancelInstance()
		lang.mu.li = nil
		return nil, true
	}
	return nil, false
}

//----------

func (lang *LangManager) PrintWrapError(err error) {
	lang.man.Error(lang.WrapError(err))
}

func (lang *LangManager) WrapError(err error) error {
	return fmt.Errorf("lsproto(%s): %w", lang.Reg.Language, err)
}

func (lang *LangManager) WrapMsg(s string) string {
	return fmt.Sprintf("lsproto(%s): %v", lang.Reg.Language, s)
}
