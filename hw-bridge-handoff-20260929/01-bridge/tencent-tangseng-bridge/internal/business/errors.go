package business

type GroupError struct {
	Step string
	Err  error
}

func (e *GroupError) Error() string { return "group " + e.Step + ": " + e.Err.Error() }
func (e *GroupError) Unwrap() error { return e.Err }
