package ui

// BasicElement describes the basic UI tree Element. Alongside ViewElement, they
// are the two types of object that can be found in a UI tree.
type BasicElement struct {
	Raw *Element
}

func (e BasicElement) watchable()          {}
func (e BasicElement) AsElement() *Element { return e.Raw }
func (e BasicElement) AsBasicElement() BasicElement { return e }

func (e BasicElement) RemoveChildren() BasicElement {
	e.AsElement().removeChildren()
	return e
}

func (e BasicElement) SetChildren(elements ...AnyElement) BasicElement {
	e.RemoveChildren()
	for _, el := range elements {
		e.AsElement().AppendChild(el)
	}
	return e
}
