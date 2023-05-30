package ui

type stackPool struct {
	objects         [][]*Element
	capacity        int
	maxCapacity     int
	baseCapacity    int
	resizeThreshold int
	constructor     func() []*Element
}

func newStackPool(baseCapacity, resizeThreshold int, constructor func() []*Element) *stackPool {
	objects := make([][]*Element, 0, baseCapacity)
	return &stackPool{
		objects:         objects,
		capacity:        baseCapacity,
		maxCapacity:     baseCapacity,
		baseCapacity:    baseCapacity,
		resizeThreshold: resizeThreshold,
		constructor:    constructor,
	}
}

func (p *stackPool) Get() []*Element {
	if len(p.objects) == 0 {
		return p.constructor()
	}

	lastIndex := len(p.objects) - 1
	obj := p.objects[lastIndex]
	p.objects = p.objects[:lastIndex]
	return obj
}

func (p *stackPool) Put(elements []*Element) {
	p.objects = append(p.objects, elements)

	if len(p.objects) <= p.capacity-p.resizeThreshold {
		p.AdjustCapacity(p.capacity - p.resizeThreshold)
	} else if len(p.objects) >= p.capacity+p.resizeThreshold {
		p.AdjustCapacity(p.capacity + p.resizeThreshold)
	}
}

func (p *stackPool) AdjustCapacity(newCapacity int) {
	if newCapacity < p.baseCapacity {
		newCapacity = p.baseCapacity
	} else if newCapacity > p.maxCapacity {
		newCapacity = p.maxCapacity
	}

	// Downsizing
	if newCapacity < p.capacity {
		excess := p.objects[newCapacity:p.capacity]
		for i := range excess {
			p.objects[i] = nil
		}
		p.objects = p.objects[:newCapacity]
	}

	p.capacity = newCapacity
}

func (p *stackPool) ResizeThreshold() int {
	return p.resizeThreshold
}

func newElementConstructor() []*Element {
	return make([]*Element, 0,128)
}


// finalizerPool

type finalizerPool struct {
	objects         [][]func()
	capacity        int
	maxCapacity     int
	baseCapacity    int
	resizeThreshold int
	constructor     func() []func()
}

func newFinalizersPool(baseCapacity, resizeThreshold int, constructor func() []func()) *finalizerPool {
	objects := make([][]func(), 0, baseCapacity)
	return &finalizerPool{
		objects:         objects,
		capacity:        baseCapacity,
		maxCapacity:     baseCapacity,
		baseCapacity:    baseCapacity,
		resizeThreshold: resizeThreshold,
		constructor:    constructor,
	}
}

func (p *finalizerPool) Get() []func() {
	if len(p.objects) == 0 {
		return p.constructor()
	}

	lastIndex := len(p.objects) - 1
	obj := p.objects[lastIndex]
	p.objects = p.objects[:lastIndex]
	return obj
}

func (p *finalizerPool) Put(elements []func()) {
	p.objects = append(p.objects, elements)

	if len(p.objects) <= p.capacity-p.resizeThreshold {
		p.adjustCapacity(p.capacity - p.resizeThreshold)
	} else if len(p.objects) >= p.capacity+p.resizeThreshold {
		p.adjustCapacity(p.capacity + p.resizeThreshold)
	}
}

func (p *finalizerPool) adjustCapacity(newCapacity int) {
	if newCapacity < p.baseCapacity {
		newCapacity = p.baseCapacity
	} else if newCapacity > p.maxCapacity {
		newCapacity = p.maxCapacity
	}

	// Downsizing
	if newCapacity < p.capacity {
		excess := p.objects[newCapacity:p.capacity]
		for i := range excess {
			p.objects[i] = nil
		}
		p.objects = p.objects[:newCapacity]
	}

	p.capacity = newCapacity
}

func (p *finalizerPool) ResizeThreshold() int {
	return p.resizeThreshold
}

func newFinalizersConstructor() []func() {
	return make([]func(), 0, 512)
}

// Object pool

type objectPool struct {
	objects         []Object
	capacity        int
	maxCapacity     int
	baseCapacity    int
	resizeThreshold int
	constructor     func() Object
}

func newObjectsPool(baseCapacity, resizeThreshold int, constructor func() []Object) *objectPool {
	objects := make([]Object, 0, baseCapacity)
	return &objectPool{
		objects:         objects,
		capacity:        baseCapacity,
		maxCapacity:     baseCapacity,
		baseCapacity:    baseCapacity,
		resizeThreshold: resizeThreshold,
		constructor:    func() Object{return Object{newobject(),false,2}},
	}
}

func (p *objectPool) Get() Object {
	if len(p.objects) == 0 {
		return p.constructor()
	}

	lastIndex := len(p.objects) - 1
	obj := p.objects[lastIndex]
	p.objects = p.objects[:lastIndex]
	return obj
}

func (p *objectPool) Put(elements Object) {
	p.objects = append(p.objects, elements)

	if len(p.objects) <= p.capacity-p.resizeThreshold {
		p.adjustCapacity(p.capacity - p.resizeThreshold)
	} else if len(p.objects) >= p.capacity+p.resizeThreshold {
		p.adjustCapacity(p.capacity + p.resizeThreshold)
	}
}

func (p *objectPool) adjustCapacity(newCapacity int) {
	if newCapacity < p.baseCapacity {
		newCapacity = p.baseCapacity
	} else if newCapacity > p.maxCapacity {
		newCapacity = p.maxCapacity
	}

	// Downsizing
	if newCapacity < p.capacity {
		excess := p.objects[newCapacity:p.capacity]
		for i := range excess {
			p.objects[i] = Object{nil, false, 2}
		}
		p.objects = p.objects[:newCapacity]
	}

	p.capacity = newCapacity
}

func (p *objectPool) ResizeThreshold() int {
	return p.resizeThreshold
}

func newObjectsConstructor() []Object {
	return make([]Object, 0, 512)
}


// List pool

type listPool struct {
	objects         [][]func()
	capacity        int
	maxCapacity     int
	baseCapacity    int
	resizeThreshold int
	constructor     func() []func()
}

func newListsPool(baseCapacity, resizeThreshold int, constructor func() []func()) *listPool {
	objects := make([][]func(), 0, baseCapacity)
	return &listPool{
		objects:         objects,
		capacity:        baseCapacity,
		maxCapacity:     baseCapacity,
		baseCapacity:    baseCapacity,
		resizeThreshold: resizeThreshold,
		constructor:    constructor,
	}
}

func (p *listPool) Get() []func() {
	if len(p.objects) == 0 {
		return p.constructor()
	}

	lastIndex := len(p.objects) - 1
	obj := p.objects[lastIndex]
	p.objects = p.objects[:lastIndex]
	return obj
}

func (p *listPool) Put(elements []func()) {
	p.objects = append(p.objects, elements)

	if len(p.objects) <= p.capacity-p.resizeThreshold {
		p.adjustCapacity(p.capacity - p.resizeThreshold)
	} else if len(p.objects) >= p.capacity+p.resizeThreshold {
		p.adjustCapacity(p.capacity + p.resizeThreshold)
	}
}

func (p *listPool) adjustCapacity(newCapacity int) {
	if newCapacity < p.baseCapacity {
		newCapacity = p.baseCapacity
	} else if newCapacity > p.maxCapacity {
		newCapacity = p.maxCapacity
	}

	// Downsizing
	if newCapacity < p.capacity {
		excess := p.objects[newCapacity:p.capacity]
		for i := range excess {
			p.objects[i] = nil
		}
		p.objects = p.objects[:newCapacity]
	}

	p.capacity = newCapacity
}

func (p *listPool) ResizeThreshold() int {
	return p.resizeThreshold
}

func newListsConstructor() []func() {
	return make([]func(), 0, 512)
}


var StackPool = newStackPool(128, 64, newElementConstructor)
var finalizersPool = newFinalizersPool(128, 64, newFinalizersConstructor)
var objectsPool = newObjectsPool(128, 64, newObjectsConstructor)
var listsPool = newListsPool(128, 64, newListsConstructor)