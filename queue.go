package restreamer

type Queue chan<- interface{}

func NewQueue(size int) *Queue {
	return &make(Queue, size)
}

func (q *Queue) Push(msg interface{}) error {
	select {
		case q <- msg:
			return nil
		default:
			return NewError(myerrors.ERR_QUEUE_FULL)
	}
}

func (q *Queue) Pop() interface{}, error {
	select {
		case msg := <-q:
			return msg, nil
		default:
			return nil, NewError(myerrors.ERR_QUEUE_EMPTY)
	}
}

func (q *Queue) Wait() interface{}, error {
	msg := <- q
	return msg
}
