package ringo

import "time"

// CircleTime is a ring buffer of time.Time values.  It is not concurrent safe.
type CircleTime struct {
	buf    []time.Time
	cursor int
	filled bool
}

// Append adds a value to the buffer
func (c *CircleTime) Append(t time.Time) {
	if c.cursor == cap(c.buf) {
		c.cursor = 0
		c.filled = true
	}
	c.buf[c.cursor] = t
	c.cursor++
}

// Head gets the most recent addition.  It returns zero-valued time if the buffer is empty
func (c *CircleTime) Head() time.Time {
	if c.cursor == 0 && !c.filled {
		return time.Time{}
	} else if c.cursor == 0 && c.filled {
		return c.buf[len(c.buf)-1]
	}
	return c.buf[c.cursor]
}

// Tail gets the least recent addition.  It returns zero-valued time if the buffer is empty
func (c *CircleTime) Tail() time.Time {
	if c.cursor == 0 && !c.filled {
		return time.Time{}
	}
	return c.buf[c.cursor-1]
}

// Contiguous gets a slice of the values in the buffer from least to most recent.
// It returns []time.Time{time.Time{}} if the buffer is empty
func (c *CircleTime) Contiguous() []time.Time {
	if c.cursor == 0 && !c.filled {
		return []time.Time{time.Time{}}
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
func (c *CircleTime) Init(size int) {
	c.buf = make([]time.Time, size)
	c.filled = false
	c.cursor = 0
}
