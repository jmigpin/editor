package gosource

import "testing"

func ccTest(t *testing.T, filename string, src interface{}, index int) {
	t.Helper()

	//LogDebug()

	err := CodeCompletion(filename, src, index)
	if err != nil {
		t.Fatal(err)
	}
}

func ccTestSrc(t *testing.T, src interface{}, index int) {
	filename := "t000/src.go"
	ccTest(t, filename, src, index)
}

func TestCC1(t *testing.T) {
	src := ` 
		package pack1
		import(
			"fmt"			
		)
		func func1() {
			fmt.Prin
		}
	`
	ccTestSrc(t, src, 68) // Prin
}
