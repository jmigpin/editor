package toolbarparser

func Parse(str string) *Data {
	//return parse1_basedOnScanutilScanner(str)
	return parse2_basedOnLrparserPState(str)
	//return parse3_basedOnLrparser(str)
}
