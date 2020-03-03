package mock

type Jobqueue struct {
	PutFunc func(uri string) error
}

func (jq Jobqueue) Put(uri string) error {
	return jq.PutFunc(uri)
}
