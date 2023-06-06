package common

import "fmt"

func NestedCloseConnErr(err, closeErr error) error {
	if closeErr != nil {
		return fmt.Errorf("%v (another error occured while closing connection: %w)", err, closeErr)
	}
	return err
}

func WrapErr(err error) error {
	if err != nil {
		return fmt.Errorf("easytcp: %v", err)
	}
	return nil
}
