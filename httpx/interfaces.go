package httpx

type WaitGroup interface {
	Add(delta int)
	Wait()
	Done()
}

type Sender interface {
	Send(interface{}) interface{}
}
