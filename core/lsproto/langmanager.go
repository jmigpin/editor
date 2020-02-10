package lsproto

import (
	"fmt"
	"sync"
)

type LangManager struct {
	Reg *Registration // accessed from editor
	man *Manager
	mu  struct {
		sync.Mutex
		li *LangInstance
	}
}

func NewLangManager(man *Manager, reg *Registration) *LangManager {
	return &LangManager{Reg: reg, man: man}
}

func (lang *LangManager) instance() *LangInstance {
	lang.mu.Lock()
	defer lang.mu.Unlock()
	if lang.mu.li == nil {
		lang.mu.li = NewLangInstance(lang)
	}
	return lang.mu.li
}

func (lang *LangManager) Close() error {
	lang.mu.Lock()
	defer lang.mu.Unlock()
	if lang.mu.li != nil {
		defer func() { lang.mu.li = nil }()
		err := lang.mu.li.closeFromLangManager()
		if err != nil {
			return lang.WrapError(err)
		}
	}
	return nil
}

//----------

func (lang *LangManager) ErrorAsync(err error) {
	lang.man.Error(lang.WrapError(err))
}

func (lang *LangManager) WrapError(err error) error {
	return fmt.Errorf("lsproto(%s): %w", lang.Reg.Language, err)
}

func (lang *LangManager) WrapMsg(s string) string {
	return fmt.Sprintf("lsproto(%s): %v", lang.Reg.Language, s)
}
