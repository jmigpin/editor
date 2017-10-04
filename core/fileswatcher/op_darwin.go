// TODO: this version allows compilation without fileswatcher support
// +build darwin

package fileswatcher

type Op uint32

func GetCreateOp() Op {
	return Op(0)
}
func GetDeleteOp() Op {
	return Op(0)
}

func (op Op) HasDelete() bool {
	return false
}
func (op Op) HasCreate() bool {
	return false
}
func (op Op) HasModify() bool {
	return false
}
func (op Op) HasIgnored() bool {
	return false
}
func (op Op) HasIsDir() bool {
	return false
}

func (op Op) String() string {
	return "?"
}
