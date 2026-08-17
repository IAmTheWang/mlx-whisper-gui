package jobmanager

import "bytes"

// lineSplitter is a bufio.SplitFunc that breaks on both '\r' and '\n'. tqdm
// redraws progress bars with a bare '\r' (no '\n'); a stock bufio.Scanner
// (which only splits on '\n') would block until the whole subprocess exits
// before ever emitting a token. '\r' immediately followed by '\n' is treated
// as a single ordinary line break (lastTerm '\n'), not a progress redraw --
// only a *bare* '\r' (not immediately followed by '\n') counts as one.
//
// bufio.Scanner only exposes the token text via Text(), not which byte(s)
// terminated it, so lastTerm records that out-of-band for the caller to read
// immediately after each successful Scan().
type lineSplitter struct {
	lastTerm byte // '\r', '\n', or 0 (unterminated fragment flushed at EOF)
}

func (s *lineSplitter) split(data []byte, atEOF bool) (advance int, token []byte, err error) {
	i := bytes.IndexAny(data, "\r\n")
	if i < 0 {
		if atEOF {
			if len(data) == 0 {
				return 0, nil, nil
			}
			s.lastTerm = 0
			return len(data), data, nil
		}
		return 0, nil, nil // request more data
	}

	if data[i] == '\n' {
		s.lastTerm = '\n'
		return i + 1, data[:i], nil
	}

	// data[i] == '\r' -- peek ahead to see if it's part of a \r\n pair.
	if i+1 == len(data) {
		if atEOF {
			s.lastTerm = '\r'
			return i + 1, data[:i], nil
		}
		return 0, nil, nil // don't know yet whether '\n' follows; request more
	}
	if data[i+1] == '\n' {
		s.lastTerm = '\n'
		return i + 2, data[:i], nil
	}
	s.lastTerm = '\r'
	return i + 1, data[:i], nil
}
