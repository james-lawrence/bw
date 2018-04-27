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

// String useful wrapper for string constants as errors.
type String string

func (t String) Error() string {
	return string(t)
}
