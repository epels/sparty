package mock

type Jobqueue struct {
	PutFunc func(url string) error
}

func (jq Jobqueue) Put(url string) error {
	return jq.PutFunc(url)
}
