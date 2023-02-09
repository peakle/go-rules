package unnecessaryErrorCall

import (
	"errors"
	"fmt"
)

func foo() error {
	fmt.Println(errors.New(fmt.Sprintf("foo: %s", "bar")))            // want `use fmt.Errorf instead of nested calls`
	fmt.Println(errors.New(fmt.Sprintf("foo: %s, %s", "bar", "boo"))) // want `use fmt.Errorf instead of nested calls`

	return errors.New(fmt.Sprintf("foo: %s", "bar")) // want `use fmt.Errorf instead of nested calls`
}
