package main

type FlagError struct {
	Name   string
	Value  string
	Reason string
}

func (e FlagError) Error() string {
	return e.Name + ": " + e.Reason
}

type UsageError struct {
	Message string
}

func (e UsageError) Error() string {
	return e.Message
}

type FileReadError struct {
	Path string
	Err  error
}

func (e FileReadError) Error() string {
	if e.Err == nil {
		return e.Path
	}
	return e.Path + ": " + e.Err.Error()
}

func (e FileReadError) Unwrap() error {
	return e.Err
}
