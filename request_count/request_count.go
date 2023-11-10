package request_count

type RequestCount struct {
	ch chan uint8
}

func New() *RequestCount {
	countCh := make(chan uint8, 999999)
	return &RequestCount{
		ch: countCh,
	}
}

func (r *RequestCount) Increase() {
	go func() {
		r.ch <- 1
	}()
}

func (r *RequestCount) Decrease() {
	go func() {
		<-r.ch
	}()
}

func (r *RequestCount) Count() int {
	return len(r.ch)
}
