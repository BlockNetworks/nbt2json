package main

import (
	"C"
	"fmt"
	"unsafe"

	"github.com/midnightfreddie/nbt2json"
)

// HelloDll is here as a test while I work out parameter passing
// Any functions or vars exposed in the shared lib must be capitalized (Go rule)
// The export comment is needed to have the "C" package make the item available
//   in the shared library. Note there must be no space between the // and
//   'export'
//export HelloDll
func HelloDll() {
	fmt.Println("Hello from the libnbt2json dll!")
}

// NOTE: Functions don't do anything yet; I'm just trying to figure out how
//   to pass C-native values to/from Go
// Oh cool, these comments are in the .h file when no blank lines separate them
// The NBT data must be in a byte array. Pass a pointer to the array and the
//   length of the array
// Temporarily hard-codeed for Bedrock / little-endian only
//export Nbt2Json
func Nbt2Json(byteArray unsafe.Pointer, length C.int) *C.char {
	var goByteArray = C.GoBytes(byteArray, length)
	fmt.Print("The first byte in the byte array is ")
	fmt.Println(goByteArray[0])
	jsonData, err := nbt2json.Nbt2Json(goByteArray, nbt2json.Bedrock, "")
	if err != nil {
		panic(err)
	}
	// var tempString = "Hello from a Go string"
	return C.CString(string(jsonData))
}

// NOTE: Functions don't do anything yet; I'm just trying to figure out how
//   to pass C-native values to/from Go
//export Json2Nbt
func Json2Nbt(cString *C.char) {
	var s string
	s = C.GoString(cString)
	fmt.Println(s)
	// return []byte(s)
}

func main() {}
