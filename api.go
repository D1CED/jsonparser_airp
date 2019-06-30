package airp

func Marshal(v interface{}) ([]byte, error) {
	n, err := NewJSONGo(v)
	if err != nil {
		return nil, err
	}
	return n.MarshalJSON()
}

func Unmarshal(data []byte, v interface{}) (err error) {
	n, err := NewJSON(data)
	if err != nil {
		return err
	}
	return n.JSON2Go(v)
}
