package common

type PlaceHolder[T any] struct{}

func (p PlaceHolder[T]) UnmarshalJSON(data []byte) error {
	return nil
}
