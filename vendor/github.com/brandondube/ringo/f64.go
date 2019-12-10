package ringo

// CircleF64 is a ring buffer of f64 values.  It is not concurrent safe.
type CircleF64 struct {
	buf    []float64
	cursor int
	filled bool
}

// Append adds a value to the buffer
func (c *CircleF64) Append(f float64) {
	if c.cursor == cap(c.buf) {
		c.cursor = 0
		c.filled = true
	}
	c.buf[c.cursor] = f
	c.cursor++
}

// Head gets the most recent addition.  It returns zero if the buffer is empty
func (c *CircleF64) Head() float64 {
	if c.cursor == 0 && !c.filled {
		return 0
	}
	return c.buf[c.cursor]
}

// Tail gets the least recent addition.  It returns zero if the buffer is empty
func (c *CircleF64) Tail() float64 {
	if c.cursor == 0 && !c.filled {
		return 0
	} else if c.cursor == 0 && c.filled {
		return c.buf[len(c.buf)-1]
	}
	return c.buf[c.cursor-1]
}

// Contiguous gets a slice of the values in the buffer from least to most recent.
// It returns []float64{0} if the buffer is empty
func (c *CircleF64) Contiguous() []float64 {
	if c.cursor == 0 && !c.filled {
		return []float64{0}
	}
	if c.filled {
		chunk1 := c.buf[c.cursor:]
		chunk2 := c.buf[:c.cursor]
		out := append(chunk1, chunk2...)
		return out
	}
	return c.buf[:c.cursor]
}

// Init creates a new slice of zeros and resets the internal state of the buffer.
// It may be called multiple times
func (c *CircleF64) Init(size int) {
	c.buf = make([]float64, size)
	c.filled = false
	c.cursor = 0
}
