package errorsx

// Compact returns the first error in the set, if any.
func Compact(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}

	return nil
}
