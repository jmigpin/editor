package core

import "flag"

type HelpRequestError struct {
	Cmd   string
	Usage string
}

func NewHelpRequestError(cmd, usage string) *HelpRequestError {
	return &HelpRequestError{Cmd: cmd, Usage: usage}
}

func (e *HelpRequestError) Error() string {
	return flag.ErrHelp.Error() + "\n" + e.Usage
}

func (e *HelpRequestError) Display() string {
	return "Help requested; " + e.Cmd + " was not run.\n" + e.Usage
}

func (e *HelpRequestError) Unwrap() error {
	return flag.ErrHelp
}
