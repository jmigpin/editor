package flagutil

type StringFuncFlag func(string) error

func (v StringFuncFlag) String() string     { return "" }
func (v StringFuncFlag) Set(s string) error { return v(s) }

//----------

type BoolFuncFlag func(string) error

func (v BoolFuncFlag) String() string     { return "" }
func (v BoolFuncFlag) Set(s string) error { return v(s) }
func (v BoolFuncFlag) IsBoolFlag() bool   { return true }
