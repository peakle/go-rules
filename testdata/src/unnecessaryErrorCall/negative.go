package unnecessaryErrorCall

import (
	"errors"
	"fmt"
)

func bar() error {
	fmt.Print(errors.New(fmt.Sprintln(1, 2, 3)))

	return errors.New(fmt.Sprint(3, 2, 1))
}
