package doc

import(
	"bytes"
	"encoding/base64"
	"encoding/binary"
)


// editOp describes an operation to be performed on the DOM.
type editOp struct {
	Operation string
	ElementID string
	Index     int
}

func myersDiff(a, b []string) []editOp {
	n, m := len(a), len(b)
	maxLen := n + m
	v := make([]int, 2*maxLen+1)
	v[1] = 0
	trace := make([][]int, 0)

	for d := 0; d <= maxLen; d++ {
		for k := -d; k <= d; k += 2 {
			var x, y int

			if k == -d || (k != d && v[k-1+maxLen] < v[k+1+maxLen]) {
				x = v[k+1+maxLen]
			} else {
				x = v[k-1+maxLen] + 1
			}

			y = x - k
			for x < n && y < m && a[x] == b[y] {
				x++
				y++
			}

			v[k+maxLen] = x

			if x >= n && y >= m {
				trace = append(trace, []int{k, x, y})
				break
			}
		}
		if v[d+maxLen] >= n && v[d-maxLen] >= m {
			break
		}
	}

	// Reconstruct edit script
	var editScript []editOp
	x, y := n, m
	for _, t := range trace {
		k, prevX, prevY := t[0], t[1], t[2]
		for x > prevX || y > prevY {
			if k == -k || (k != k && v[k-1+maxLen] < v[k+1+maxLen]) {
				x = v[k+1+maxLen]
			} else {
				x = v[k-1+maxLen] + 1
			}

			y = x - k
			newX, newY := x-1, y-1
			if x == newX+1 {
				editScript = append([]editOp{{
					Operation: "Insert",
					ElementID: b[newY],
					Index:     newY,
				}}, editScript...)
			} else {
				editScript = append([]editOp{{
					Operation: "Remove",
					ElementID: a[newX],
					Index:     newX,
				}}, editScript...)
			}
			x, y = newX, newY
		}
	}

	return editScript
}

func encodeEditOperations(operations []editOp) string {
	var buf bytes.Buffer
	for _, op := range operations {
		buf.WriteByte(byte(len(op.Operation)))
		buf.WriteString(op.Operation)

		idLen := byte(len(op.ElementID))
		buf.WriteByte(idLen)
		buf.WriteString(op.ElementID)

		indexBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(indexBytes, uint32(op.Index))
		buf.Write(indexBytes)
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes())
}



// Helpers

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func indexOf(slice []string, value string) int {
	for i, v := range slice {
		if v == value {
			return i
		}
	}
	return -1
}

func insert(slice []string, index int, value string) []string {
	return append(slice[:index], append([]string{value}, slice[index:]...)...)
}

func remove(slice []string, index int) []string {
	return append(slice[:index], slice[index+1:]...)
}
