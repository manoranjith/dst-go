package session

type Sample struct {
	a string
	b string
	c uint8
}

func NewSample() *Sample {
	return &Sample{
		a: "aone",
		b: "bone",
		c: 21,
	}
}
