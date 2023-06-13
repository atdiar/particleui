package ui

import(
	"bytes"
	"encoding/base64"
	"encoding/binary"
)


// EditOps describes an operation to be performed on the DOM.
type EditOp struct {
	Operation string
	ElementID string
	Index     int
}


type point struct {
	x, y int
	op   string
}

func MyersDiff(a, b []string) []EditOp {
	if len(a) == 0 && len(b) == 0 {
        return []EditOp{}
    }

	n, m := len(a), len(b)
	max := n + m
	v := make([]int, 2*max+1)

	trace := make([][]point, max+1)

	for d := 0; d <= max; d++ {
		trace[d] = make([]point, 2*max+1)

		for k := -d; k <= d; k += 2 {
			var x int
			var op string

			if k == -d || (k != d && v[k-1+max] < v[k+1+max]) {
				x = v[k+1+max]
				op = "Insert"
			} else {
				x = v[k-1+max] + 1
				op = "Remove"
			}

			y := x - k
			trace[d][k+max] = point{x, y, op}

			for x < n && y < m && a[x] == b[y] {
				x, y = x+1, y+1
			}

			v[k+max] = x
			if x >= n && y >= m {
				return generateEditScript(trace, d, k+max, a, b)
			}
		}
	}
	return nil
}

func generateEditScript(trace [][]point, d, k int, a, b []string) []EditOp {
	var ops []EditOp
	for d >= 0 {
		pt := trace[d][k]
		if pt.op == "Remove" {
			if pt.x > 0 {
				ops = append(ops, EditOp{"Remove", a[pt.x-1], pt.x - 1})
			}
			k--
		} else if pt.op == "Insert" {
			if pt.y > 0 {
				ops = append(ops, EditOp{"Insert", b[pt.y-1], pt.y - 1})
			}
			k++
		}
		d--
	}
	return reverse(ops)
}

func reverse(ops []EditOp) []EditOp {
	for i := len(ops)/2 - 1; i >= 0; i-- {
		opp := len(ops) - 1 - i
		ops[i], ops[opp] = ops[opp], ops[i]
	}
	return ops
}

func applyEdits(e *Element, edits []EditOp, children map[string]*Element) (finalize func()){
	var finalizers = finalizersPool.Get()
	for _, edit := range edits {
		switch edit.Operation {
		case "Insert":
			c:= children[edit.ElementID]
			e.Children.Insert(c,edit.Index)
			finalize := attach(e,c)
			finalizers = append(finalizers,finalize)

		case "Remove":
			c:= children[edit.ElementID]
			finalize := detach(c)
			finalizers = append(finalizers,finalize)
			e.Children.Remove(c)
		}
	}

	return func(){
		for i, f:= range finalizers{
			f()
			finalizers[i] = nil
		}
		finalizers = finalizers[:0]
		finalizersPool.Put(finalizers)
	}
}

func EncodeEditOperations(operations []EditOp) string {
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
