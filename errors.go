package args

type notFoundErr interface {
	IsNotFound() bool
}

// Returned by Store implementations when a value is not found
type NotFoundErr struct {
	msg string
}

func (s *NotFoundErr) Error() string {
	return s.msg
}

// Return true if the error implements IsNotFound()
func IsNotFoundErr(err error) bool {
	obj, ok := err.(notFoundErr)
	if !ok {
		return false
	}
	return obj.IsNotFound()
}
