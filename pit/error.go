package pit

import (
	"sync"
)

type errorMap sync.Map

func (e errorMap) add(err error) {
}

func (e errorMap) get(err error) uint64 {
	return 0
}

func (e errorMap) sum() uint64 {
	return 0
}
