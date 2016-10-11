package tautil

//"fmt"

//func TestSelectionStringIndexes0(t *testing.T) {
//text:="abcd\nabcd\nabcd"
//ta := &Textad{
//text:text,
//cursorIndex:0,
//selectionOn:false,
//selectionIndex:0,
//}
//a, b, ok := selectionStringIndexes(ta)
//if !(a==0 && b==1 && ok==true){
//t.Fatal(a,b,ok)
//}
//}
//func TestSelectionStringIndexes1(t *testing.T) {
//text:="abcd\nabcd\nabcd"
//ta := &Textad{
//text:text,
//cursorIndex:len(text),
//selectionOn:false,
//selectionIndex:0,
//}
//a, b, ok := selectionStringIndexes(ta)
//if !(ok==false){
//t.Fatal(a,b,ok)
//}
//}
//func TestSelectionStringIndexes2(t *testing.T) {
//text:="abcd\nabcd\nabcd"
//ta := &Textad{
//text:text,
//cursorIndex:len(text)-1,
//selectionOn:false,
//selectionIndex:0,
//}
//a, b, ok := selectionStringIndexes(ta)
//if !(a==len(text)-1 && b==len(text) && ok==true){
//t.Fatal(a,b,ok)
//}
//}

// selection on

//func TestSelectionStringIndexes3(t *testing.T) {
//text:="abcd\nabcd\nabcd"
//ta := &Textad{
//text:text,
//cursorIndex:0,
//selectionOn:true,
//selectionIndex:0,
//}
//a, b, ok := selectionStringIndexes(ta)
//if !(a==0 && b==1 && ok==true){
//t.Fatal(a,b,ok)
//}
//}
//func TestSelectionStringIndexes4(t *testing.T) {
//text:="abcd\nabcd\nabcd"
//ta := &Textad{
//text:text,
//cursorIndex:1,
//selectionOn:true,
//selectionIndex:0,
//}
//a, b, ok := selectionStringIndexes(ta)
//if !(a==0 && b==2 && ok==true){
//t.Fatal(a,b,ok)
//}
//}
//func TestSelectionStringIndexes5(t *testing.T) {
//text:="abcd\nabcd\nabcd"
//ta := &Textad{
//text:text,
//cursorIndex:0,
//selectionOn:true,
//selectionIndex:1,
//}
//a, b, ok := selectionStringIndexes(ta)
//if !(a==0 && b==1 && ok==true){
//t.Fatal(a,b,ok)
//}
//}
//func TestSelectionStringIndexes6(t *testing.T) {
//text:="abcd\nabcd\nabcd"
//ta := &Textad{
//text:text,
//cursorIndex:0,
//selectionOn:true,
//selectionIndex:len(text),
//}
//a, b, ok := selectionStringIndexes(ta)
//if !(a==0 && b==len(text) && ok==true){
//t.Fatal(a,b,ok)
//}
//}
//func TestSelectionStringIndexes7(t *testing.T) {
//text:="abcd\nabcd\nabcd"
//ta := &Textad{
//text:text,
//cursorIndex:len(text),
//selectionOn:true,
//selectionIndex:0,
//}
//a, b, ok := selectionStringIndexes(ta)
//if !(a==0 && b==len(text) && ok==true){
//t.Fatal(a,b,ok)
//}
//}
//func TestSelectionStringIndexes8(t *testing.T) {
//text:="abcd\nabcd\nabcd"
//ta := &Textad{
//text:text,
//cursorIndex:len(text),
//selectionOn:true,
//selectionIndex:len(text),
//}
//a, b, ok := selectionStringIndexes(ta)
//if !(ok==false){
//t.Fatal(a,b,ok)
//}
//}

// lines selection

//func TestLinesSelectionStringIndexes0(t *testing.T) {
//text:="abcd\nabcd\nabcd"
//ta := &Textad{
//text:text,
//cursorIndex:8,
//selectionOn:false,
//selectionIndex:0,
//}
//a, b, ok := linesSelectionStringIndexes(ta)
//if !(a==5 && b==10 && ok==true){
//t.Fatal(a,b,ok)
//}
//}
//func TestLinesSelectionStringIndexes1(t *testing.T) {
//text:="abcd\nabcd\nabcd"
//ta := &Textad{
//text:text,
//cursorIndex:5,
//selectionOn:false,
//selectionIndex:0,
//}
//a, b, ok := linesSelectionStringIndexes(ta)
//if !(a==5 && b==10 && ok==true){
//t.Fatal(a,b,ok)
//}
//}
//func TestLinesSelectionStringIndexes2(t *testing.T) {
//text:="abcd\n\nabcd"
//ta := &Textad{
//text:text,
//cursorIndex:5,
//selectionOn:false,
//selectionIndex:0,
//}
//a, b, ok := linesSelectionStringIndexes(ta)
//if !(a==5 && b==6 && ok==true){
//t.Fatal(a,b,ok)
//}
//}
//func TestLinesSelectionStringIndexes3(t *testing.T) {
//text:="abcd\n\n"
//ta := &Textad{
//text:text,
//cursorIndex:6,
//selectionOn:false,
//selectionIndex:0,
//}
//a, b, ok := linesSelectionStringIndexes(ta)
//if !(ok==false){
//t.Fatal(a,b,ok)
//}
//}
