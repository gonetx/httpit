package internal

import "errors"

type Pit struct {
	c Config
}

func NewPit(c Config) *Pit {
	return &Pit{
		c: c,
	}
}

func (p *Pit) Run() (err error) {
	return errors.New("error")
}
