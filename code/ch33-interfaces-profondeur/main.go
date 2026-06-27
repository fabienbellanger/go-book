package main

import (
	"fmt"
	"unsafe"
)

func main() {
	var empty any = 42
	var shape Shape = Circle{R: 2}
	fmt.Printf("Sizeof(any)=%d, Sizeof(Shape)=%d (2 mots chacun : type + données)\n",
		unsafe.Sizeof(empty), unsafe.Sizeof(shape))

	shapes := []Shape{Circle{R: 1}, Rectangle{W: 2, H: 3}, Circle{R: 2}}
	fmt.Printf("TotalArea = %.4f\n", TotalArea(shapes))
	for _, s := range shapes {
		fmt.Printf("  %s -> aire %.4f\n", Describe(s), s.Area())
	}

	// Le piège interface-nil non-nil.
	fmt.Printf("FailBuggy(true)   == nil ? %v  (PIEGE : on attendait true)\n", FailBuggy(true) == nil)
	fmt.Printf("FailCorrect(true) == nil ? %v  (correct)\n", FailCorrect(true) == nil)

	// reflect.TypeAssert.
	if c, ok := AsCircle(Circle{R: 5}); ok {
		fmt.Printf("AsCircle -> rayon %g\n", c.R)
	}
}
