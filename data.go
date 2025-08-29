package eventual2go

// Data represents generic datatype.
type Data interface{}

// Outcome represents an outcome of future
type Outcome[T Data] struct {
	Result T
	Error  error
}
