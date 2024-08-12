package doc

import ui "github.com/atdiar/particleui"

// Styling via CSS
//  Styling is defined per element it applies on (so by its unique ID)
//  and per pseudoclass (hover, active, focus, visited, link, first-child, last-child, checked, disabled, enabled)

type styleFn func(stylesheetID string, stylefns ...func(*ui.Element) *ui.Element) func(*ui.Element) *ui.Element

func (s styleFn) ForAllStylesheets(stylefns ...func(*ui.Element) *ui.Element) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		d := GetDocument(e)
		for stylesheetID := range d.StyleSheets {
			s(stylesheetID, stylefns...)(e)
		}
		return e
	}
}

// Style allows to define a style for an element, namespaced by stylesheet ID.
// Each call to style clears any previous style, allowing to specify a new one for
// that specific element, for a given stylesheet.
// That means that calling Style multiple times on the same element will only replace the previous style.
// The categorisation in different stylesheet could allow for instance to
// differentiate styles for light and dark mode.
// Style are functions and as such are composable.
// This means that a modfied version of a style can merely be created by composition, using the
// base style as the first stylefns argument.
// It has a ForAllStylesheets method that applies the style to all stylesheets of the document.
var Style styleFn = func(stylesheetID string, stylefns ...func(*ui.Element) *ui.Element) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		if styled(stylesheetID, e) {
			panic("Element already styled for this stylesheet")
		}

		clearStyle(e)
		for _, fn := range stylefns {
			e = fn(e)
		}
		document := GetDocument(e)
		s, ok := document.GetStyleSheet(stylesheetID)
		if !ok {
			s = document.NewStyleSheet(stylesheetID)
		}
		defer s.Update()
		// TODO retrieve css ruleset and "translert" (translate + insert) into stylesheet
		rls, ok := e.Get("css", "ruleset")
		if !ok {
			return e
		}
		ruleset := rls.(ui.Object)
		ruleset.Range(func(pseudoclass string, rulelist ui.Value) bool {
			rulelist.(ui.List).Range(func(i int, rule ui.Value) bool {
				ruleobj := rule.(ui.Object)
				property := ruleobj.MustGetString("property")
				value := ruleobj.MustGetString("value")
				// Let's add the rule to the stylesheet for this element
				selector := "#" + e.ID + pseudoclass
				rulestr := property.String() + ":" + value.String() + ";"
				s.InsertRule(selector, rulestr)
				return false
			})
			return false
		})
		return e
	}
}

// CSS holds a list of style modifying functions organized by
// the type of element they apply to. (container or content) and what they do (change of Style or of layout).
// For instance, to change the background color of a container to red,
// one would use CSS.Container.Style.BackgroundColor.Value("red").
// To center the content of an element, one would use CSS.Content.Layout.AlignItems.Center (for Flex items)
// Now we can center a div?
var CSS = struct {
	Container
	Content
}{*newContainer(), *newContent()}

func css(pseudoclass string, property, value string) func(*ui.Element) *ui.Element {
	return func(e *ui.Element) *ui.Element {
		// Retrieve/Create a css ruleset object and store the new rule/declaration.
		var ruleset *ui.TempObject
		c, ok := e.Get("css", "ruleset")
		if !ok {
			ruleset = ui.NewObject()
		}
		ruleset = c.(ui.Object).MakeCopy()
		ruleobj, ok := ruleset.Get(pseudoclass)
		if !ok {
			rules := ui.NewObject().
				Set("property", ui.String(property)).
				Set("value", ui.String(value)).
				Commit()
			ruleset.Set(pseudoclass, ui.NewList(rules).Commit())

		} else {

			rulelist := ruleobj.(ui.List).MakeCopy()
			rules := ui.NewObject().
				Set("property", ui.String(property)).
				Set("value", ui.String(value)).
				Commit()
			rulelist.Append(rules)
			ruleset.Set(pseudoclass, rulelist.Commit())
		}
		e.Set("css", "ruleset", ruleset.Commit())
		return e
	}
}

/*
func StyleRemoveAll(e *ui.Element) *ui.Element{
	clearStyle(e)
	e.Set("internals", "css-styles-list", ui.NewList().Commit())
	return e
}

func StyleRemoveFor(stylesheetid string) func(*ui.Element) *ui.Element{
	return func(e *ui.Element) *ui.Element{
		stylesheets,ok:= e.Get("internals","css-styles-list")
		if !ok{
			return clearStyle(e)
		}
		s:= stylesheets.(ui.List).Filter(func(v ui.Value)bool{
			return v.(ui.String).String() != stylesheetid
		})
		e.Set("internals","css-styles-list",s)

		return clearStyle(e)
	}

}
*/

func clearStyle(e *ui.Element) *ui.Element {
	e.Set("css", "ruleset", ui.NewObject().Commit())
	return e
}

func styled(stylesheetid string, e *ui.Element) bool {
	styles, ok := e.Get("internals", "css-styles-list")
	if !ok {
		return false
	}
	return styles.(ui.List).Contains(ui.String(stylesheetid))
}

type valueFn[T any] struct {
	VFn func(property, pseudoclass string) func(*ui.Element) *ui.Element
	CFn func(property, pseudoclass string) func(value string) func(*ui.Element) *ui.Element
	Typ T
}

func (v valueFn[T]) Setter(property string) func(*ui.Element) *ui.Element {
	var pseudoclass string
	switch any(v.Typ).(type) {
	case Hover:
		pseudoclass = ":hover"
	case Active:
		pseudoclass = ":active"
	case Focus:
		pseudoclass = ":focus"
	case Visited:
		pseudoclass = ":visited"
	case FirstChild:
		pseudoclass = ":first-child"
	case LastChild:
		pseudoclass = ":last-child"
	case Checked:
		pseudoclass = ":checked"
	case Disabled:
		pseudoclass = ":disabled"
	case Enabled:
		pseudoclass = ":enabled"
	default:
		pseudoclass = ""
	}
	return v.VFn(property, pseudoclass)
}

func (v valueFn[T]) CustomSetter(property string) func(value string) func(*ui.Element) *ui.Element {
	var pseudoclass string
	switch any(v.Typ).(type) {
	case Hover:
		pseudoclass = ":hover"
	case Active:
		pseudoclass = ":active"
	case Focus:
		pseudoclass = ":focus"
	case Visited:
		pseudoclass = ":visited"
	case FirstChild:
		pseudoclass = ":first-child"
	case LastChild:
		pseudoclass = ":last-child"
	case Checked:
		pseudoclass = ":checked"
	case Disabled:
		pseudoclass = ":disabled"
	case Enabled:
		pseudoclass = ":enabled"
	default:
		pseudoclass = ""
	}
	return v.CFn(property, pseudoclass)
}

func newValueFn[T any](fn func(property, pseudoclass string) func(*ui.Element) *ui.Element, cfn func(property, pseudoclass string) func(value string) func(*ui.Element) *ui.Element) valueFn[T] {
	var v T
	return valueFn[T]{fn, cfn, v}
}

func cfn(property, pseudoclass string) func(val string) func(*ui.Element) *ui.Element {
	return func(val string) func(e *ui.Element) *ui.Element {
		return css(pseudoclass, property, val)
	}
}

func vfn(value string) func(property string, pseudoclass string) func(*ui.Element) *ui.Element {
	return func(property, pseudoclass string) func(*ui.Element) *ui.Element {
		return cfn(property, pseudoclass)(value)
	}
}

type PseudoClass struct{}
type None PseudoClass
type Hover PseudoClass
type Active PseudoClass
type Focus PseudoClass
type Visited PseudoClass
type Link PseudoClass
type FirstChild PseudoClass
type LastChild PseudoClass
type Checked PseudoClass
type Disabled PseudoClass
type Enabled PseudoClass

type Container struct {
	Style  ContainerStyle
	Layout ContainerLayout

	Hover      *Container
	Active     *Container
	Focus      *Container
	Visited    *Container
	Link       *Container
	FirstChild *Container
	LastChild  *Container
	Checked    *Container
	Disabled   *Container
	Enabled    *Container
}

type Content struct {
	Style  ContentStyle
	Layout ContentLayout

	Hover      *Content
	Active     *Content
	Focus      *Content
	Visited    *Content
	Link       *Content
	FirstChild *Content
	LastChild  *Content
	Checked    *Content
	Disabled   *Content
	Enabled    *Content
}

func newContainer() *Container {
	c := initializeContainer[None]()
	c.Hover = initializeContainer[Hover]()
	c.Active = initializeContainer[Active]()
	c.Focus = initializeContainer[Focus]()
	c.Visited = initializeContainer[Visited]()
	c.Link = initializeContainer[Link]()
	c.FirstChild = initializeContainer[FirstChild]()
	c.LastChild = initializeContainer[LastChild]()
	c.Checked = initializeContainer[Checked]()
	c.Disabled = initializeContainer[Disabled]()
	c.Enabled = initializeContainer[Enabled]()
	return c
}

func newContent() *Content {
	c := initializeContent[None]()
	c.Hover = initializeContent[Hover]()
	c.Active = initializeContent[Active]()
	c.Focus = initializeContent[Focus]()
	c.Visited = initializeContent[Visited]()
	c.Link = initializeContent[Link]()
	c.FirstChild = initializeContent[FirstChild]()
	c.LastChild = initializeContent[LastChild]()
	c.Checked = initializeContent[Checked]()
	c.Disabled = initializeContent[Disabled]()
	c.Enabled = initializeContent[Enabled]()
	return c
}

type CSSStyles struct {
	Container
	Content
}

func NewCSSStyles() CSSStyles {
	c := CSSStyles{}
	c.Container = *newContainer()
	c.Content = *newContent()
	return c
}

func initializeContainer[pseudoclass any]() *Container {
	c := Container{}
	c.Style = initializeContainerStyle[pseudoclass]()
	c.Layout = initializeContainerLayout[pseudoclass]()
	return &c
}

func initializeContent[pseudoclass any]() *Content {
	c := Content{}
	c.Style = initializeContentStyle[pseudoclass]()
	c.Layout = initializeContentLayout[pseudoclass]()
	return &c
}

func initializeContainerLayout[pseudoclass any]() ContainerLayout {
	// Setting the proper function for each field
	c := ContainerLayout{}
	c.BoxShadow.None = newValueFn[pseudoclass](vfn("None"), nil).Setter("box-shadow")
	c.BoxShadow.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("box-shadow")

	c.JustifyContent.FlexStart = newValueFn[pseudoclass](vfn("flex-start"), nil).Setter("justify-content")
	c.JustifyContent.FlexEnd = newValueFn[pseudoclass](vfn("flex-end"), nil).Setter("justify-content")
	c.JustifyContent.Center = newValueFn[pseudoclass](vfn("center"), nil).Setter("justify-content")
	c.JustifyContent.SpaceBetween = newValueFn[pseudoclass](vfn("space-between"), nil).Setter("justify-content")
	c.JustifyContent.SpaceAround = newValueFn[pseudoclass](vfn("space-around"), nil).Setter("justify-content")
	c.JustifyContent.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("justify-content")

	// ZIndex
	c.ZIndex.Auto = newValueFn[pseudoclass](vfn("auto"), nil).Setter("z-index")
	c.ZIndex.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("z-index")

	// Float
	c.Float.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("float")
	c.Float.Left = newValueFn[pseudoclass](vfn("left"), nil).Setter("float")
	c.Float.Right = newValueFn[pseudoclass](vfn("right"), nil).Setter("float")
	c.Float.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("float")

	// Overflow
	c.Overflow.Visible = newValueFn[pseudoclass](vfn("visible"), nil).Setter("overflow")
	c.Overflow.Hidden = newValueFn[pseudoclass](vfn("hidden"), nil).Setter("overflow")
	c.Overflow.Scroll = newValueFn[pseudoclass](vfn("scroll"), nil).Setter("overflow")
	c.Overflow.Auto = newValueFn[pseudoclass](vfn("auto"), nil).Setter("overflow")
	c.Overflow.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("overflow")

	// OverflowY
	c.OverflowY.Visible = newValueFn[pseudoclass](vfn("visible"), nil).Setter("overflow-y")
	c.OverflowY.Hidden = newValueFn[pseudoclass](vfn("hidden"), nil).Setter("overflow-y")
	c.OverflowY.Scroll = newValueFn[pseudoclass](vfn("scroll"), nil).Setter("overflow-y")
	c.OverflowY.Auto = newValueFn[pseudoclass](vfn("auto"), nil).Setter("overflow-y")
	c.OverflowY.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("overflow-y")

	// Perspective
	c.Perspective.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("perspective")
	c.Perspective.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("perspective")

	// BorderCollapse
	c.BorderCollapse.Separate = newValueFn[pseudoclass](vfn("separate"), nil).Setter("border-collapse")
	c.BorderCollapse.Collapse = newValueFn[pseudoclass](vfn("collapse"), nil).Setter("border-collapse")
	c.BorderCollapse.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-collapse")

	// PageBreakBefore
	c.PageBreakBefore.Auto = newValueFn[pseudoclass](vfn("auto"), nil).Setter("page-break-before")
	c.PageBreakBefore.Always = newValueFn[pseudoclass](vfn("always"), nil).Setter("page-break-before")
	c.PageBreakBefore.Avoid = newValueFn[pseudoclass](vfn("avoid"), nil).Setter("page-break-before")
	c.PageBreakBefore.Left = newValueFn[pseudoclass](vfn("left"), nil).Setter("page-break-before")
	c.PageBreakBefore.Right = newValueFn[pseudoclass](vfn("right"), nil).Setter("page-break-before")
	c.PageBreakBefore.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("page-break-before")

	// Columns
	c.Columns.Auto = newValueFn[pseudoclass](vfn("auto"), nil).Setter("columns")
	c.Columns.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("columns")

	// ColumnCount
	c.ColumnCount.Auto = newValueFn[pseudoclass](vfn("auto"), nil).Setter("column-count")
	c.ColumnCount.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("column-count")

	// MinHeight
	c.MinHeight.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("min-height")

	// PageBreakInside
	c.PageBreakInside.Auto = newValueFn[pseudoclass](vfn("auto"), nil).Setter("page-break-inside")
	c.PageBreakInside.Avoid = newValueFn[pseudoclass](vfn("avoid"), nil).Setter("page-break-inside")
	c.PageBreakInside.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("page-break-inside")

	// ColumnGap
	c.ColumnGap.Length = newValueFn[pseudoclass](nil, cfn).CustomSetter("column-gap")
	c.ColumnGap.Normal = newValueFn[pseudoclass](vfn("normal"), nil).Setter("column-gap")
	c.ColumnGap.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("column-gap")

	// Clip
	c.Clip.Auto = newValueFn[pseudoclass](vfn("auto"), nil).Setter("clip")
	c.Clip.Shape = newValueFn[pseudoclass](vfn("shape"), nil).Setter("clip")
	c.Clip.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("clip")

	// FlexDirection
	c.FlexDirection.Row = newValueFn[pseudoclass](vfn("row"), nil).Setter("flex-direction")
	c.FlexDirection.RowReverse = newValueFn[pseudoclass](vfn("row-reverse"), nil).Setter("flex-direction")
	c.FlexDirection.Column = newValueFn[pseudoclass](vfn("column"), nil).Setter("flex-direction")
	c.FlexDirection.ColumnReverse = newValueFn[pseudoclass](vfn("column-reverse"), nil).Setter("flex-direction")
	c.FlexDirection.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("flex-direction")

	// PageBreakAfter
	c.PageBreakAfter.Auto = newValueFn[pseudoclass](vfn("auto"), nil).Setter("page-break-after")
	c.PageBreakAfter.Always = newValueFn[pseudoclass](vfn("always"), nil).Setter("page-break-after")
	c.PageBreakAfter.Avoid = newValueFn[pseudoclass](vfn("avoid"), nil).Setter("page-break-after")
	c.PageBreakAfter.Left = newValueFn[pseudoclass](vfn("left"), nil).Setter("page-break-after")
	c.PageBreakAfter.Right = newValueFn[pseudoclass](vfn("right"), nil).Setter("page-break-after")
	c.PageBreakAfter.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("page-break-after")

	// Top
	c.Top.Auto = newValueFn[pseudoclass](vfn("auto"), nil).Setter("top")
	c.Top.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("top")

	// CounterIncrement
	c.CounterIncrement.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("counter-increment")
	c.CounterIncrement.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("counter-increment")

	// Height
	c.Height.Auto = newValueFn[pseudoclass](vfn("auto"), nil).Setter("height")
	c.Height.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("height")

	// TransformStyle
	c.TransformStyle.Flat = newValueFn[pseudoclass](vfn("flat"), nil).Setter("transform-style")
	c.TransformStyle.Preserve3d = newValueFn[pseudoclass](vfn("preserve-3d"), nil).Setter("transform-style")
	c.TransformStyle.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("transform-style")

	// OverflowX
	c.OverflowX.Visible = newValueFn[pseudoclass](vfn("visible"), nil).Setter("overflow-x")
	c.OverflowX.Hidden = newValueFn[pseudoclass](vfn("hidden"), nil).Setter("overflow-x")
	c.OverflowX.Scroll = newValueFn[pseudoclass](vfn("scroll"), nil).Setter("overflow-x")
	c.OverflowX.Auto = newValueFn[pseudoclass](vfn("auto"), nil).Setter("overflow-x")
	c.OverflowX.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("overflow-x")

	// FlexWrap
	c.FlexWrap.Nowrap = newValueFn[pseudoclass](vfn("nowrap"), nil).Setter("flex-wrap")
	c.FlexWrap.Wrap = newValueFn[pseudoclass](vfn("wrap"), nil).Setter("flex-wrap")
	c.FlexWrap.WrapReverse = newValueFn[pseudoclass](vfn("wrap-reverse"), nil).Setter("flex-wrap")
	c.FlexWrap.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("flex-wrap")

	// MaxWidth
	c.MaxWidth.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("max-width")
	c.MaxWidth.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("max-width")

	// Bottom
	c.Bottom.Auto = newValueFn[pseudoclass](vfn("auto"), nil).Setter("bottom")
	c.Bottom.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("bottom")

	// CounterReset
	c.CounterReset.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("counter-reset")
	c.CounterReset.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("counter-reset")

	// Right
	c.Right.Auto = newValueFn[pseudoclass](vfn("auto"), nil).Setter("right")
	c.Right.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("right")

	// BoxSizing
	c.BoxSizing.ContentBox = newValueFn[pseudoclass](vfn("content-box"), nil).Setter("box-sizing")
	c.BoxSizing.BorderBox = newValueFn[pseudoclass](vfn("border-box"), nil).Setter("box-sizing")
	c.BoxSizing.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("box-sizing")

	// Position
	c.Position.Static = newValueFn[pseudoclass](vfn("static"), nil).Setter("position")
	c.Position.Absolute = newValueFn[pseudoclass](vfn("absolute"), nil).Setter("position")
	c.Position.Fixed = newValueFn[pseudoclass](vfn("fixed"), nil).Setter("position")
	c.Position.Relative = newValueFn[pseudoclass](vfn("relative"), nil).Setter("position")
	c.Position.Sticky = newValueFn[pseudoclass](vfn("sticky"), nil).Setter("position")
	c.Position.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("position")

	// TableLayout
	c.TableLayout.Auto = newValueFn[pseudoclass](vfn("auto"), nil).Setter("table-layout")
	c.TableLayout.Fixed = newValueFn[pseudoclass](vfn("fixed"), nil).Setter("table-layout")
	c.TableLayout.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("table-layout")

	// Width
	c.Width.Auto = newValueFn[pseudoclass](vfn("auto"), nil).Setter("width")
	c.Width.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("width")

	// MaxHeight
	c.MaxHeight.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("max-height")
	c.MaxHeight.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("max-height")

	// ColumnWidth
	c.ColumnWidth.Auto = newValueFn[pseudoclass](vfn("auto"), nil).Setter("column-width")
	c.ColumnWidth.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("column-width")

	// MinWidth
	c.MinWidth.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("min-width")

	// VerticalAlign
	c.VerticalAlign.Baseline = newValueFn[pseudoclass](vfn("baseline"), nil).Setter("vertical-align")
	c.VerticalAlign.Top = newValueFn[pseudoclass](vfn("top"), nil).Setter("vertical-align")
	c.VerticalAlign.TextTop = newValueFn[pseudoclass](vfn("text-top"), nil).Setter("vertical-align")
	c.VerticalAlign.Middle = newValueFn[pseudoclass](vfn("middle"), nil).Setter("vertical-align")
	c.VerticalAlign.Bottom = newValueFn[pseudoclass](vfn("bottom"), nil).Setter("vertical-align")
	c.VerticalAlign.TextBottom = newValueFn[pseudoclass](vfn("text-bottom"), nil).Setter("vertical-align")
	c.VerticalAlign.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("vertical-align")

	// PerspectiveOrigin
	c.PerspectiveOrigin.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("perspective-origin")

	// AlignContent
	c.AlignContent.Stretch = newValueFn[pseudoclass](vfn("stretch"), nil).Setter("align-content")
	c.AlignContent.Center = newValueFn[pseudoclass](vfn("center"), nil).Setter("align-content")
	c.AlignContent.FlexStart = newValueFn[pseudoclass](vfn("flex-start"), nil).Setter("align-content")
	c.AlignContent.FlexEnd = newValueFn[pseudoclass](vfn("flex-end"), nil).Setter("align-content")
	c.AlignContent.SpaceBetween = newValueFn[pseudoclass](vfn("space-between"), nil).Setter("align-content")
	c.AlignContent.SpaceAround = newValueFn[pseudoclass](vfn("space-around"), nil).Setter("align-content")
	c.AlignContent.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("align-content")

	// FlexFlow
	c.FlexFlow.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("flex-flow")

	// Display
	c.Display.Inline = newValueFn[pseudoclass](vfn("inline"), nil).Setter("display")
	c.Display.Block = newValueFn[pseudoclass](vfn("block"), nil).Setter("display")
	c.Display.Contents = newValueFn[pseudoclass](vfn("contents"), nil).Setter("display")
	c.Display.Flex = newValueFn[pseudoclass](vfn("flex"), nil).Setter("display")
	c.Display.Grid = newValueFn[pseudoclass](vfn("grid"), nil).Setter("display")
	c.Display.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("display")
	c.Display.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("display")

	// Left
	c.Left.Auto = newValueFn[pseudoclass](vfn("auto"), nil).Setter("left")
	c.Left.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("left")

	c.GridTemplateColumns.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("grid-template-columns")

	c.GridTemplateRows.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("grid-template-rows")

	c.GridColumn.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("grid-column")

	c.GridRow.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("grid-row")

	c.Gap.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("gap")

	c.ScrollBehavior.Auto = newValueFn[pseudoclass](vfn("auto"), nil).Setter("scroll-behavior")
	c.ScrollBehavior.Smooth = newValueFn[pseudoclass](vfn("smooth"), nil).Setter("scroll-behavior")
	c.ScrollBehavior.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("scroll-behavior")

	return c

}

func initializeContainerStyle[pseudoclass any]() ContainerStyle {
	c := ContainerStyle{}

	c.BackgroundImage.URL = newValueFn[pseudoclass](nil, cfn).CustomSetter("background-image")
	c.BackgroundImage.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("background-image")
	c.BackgroundImage.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("background-image")

	// BorderLeftStyle
	c.BorderLeftStyle.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("border-left-style")
	c.BorderLeftStyle.Hidden = newValueFn[pseudoclass](vfn("hidden"), nil).Setter("border-left-style")
	c.BorderLeftStyle.Dotted = newValueFn[pseudoclass](vfn("dotted"), nil).Setter("border-left-style")
	c.BorderLeftStyle.Dashed = newValueFn[pseudoclass](vfn("dashed"), nil).Setter("border-left-style")
	c.BorderLeftStyle.Solid = newValueFn[pseudoclass](vfn("solid"), nil).Setter("border-left-style")
	c.BorderLeftStyle.Double = newValueFn[pseudoclass](vfn("double"), nil).Setter("border-left-style")
	c.BorderLeftStyle.Groove = newValueFn[pseudoclass](vfn("groove"), nil).Setter("border-left-style")
	c.BorderLeftStyle.Ridge = newValueFn[pseudoclass](vfn("ridge"), nil).Setter("border-left-style")
	c.BorderLeftStyle.Inset = newValueFn[pseudoclass](vfn("inset"), nil).Setter("border-left-style")
	c.BorderLeftStyle.Outset = newValueFn[pseudoclass](vfn("outset"), nil).Setter("border-left-style")
	c.BorderLeftStyle.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-left-style")

	// BoxShadow
	c.BoxShadow.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("box-shadow")
	c.BoxShadow.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("box-shadow")

	// TransitionDelay
	c.TransitionDelay.Time = newValueFn[pseudoclass](nil, cfn).CustomSetter("transition-delay")
	c.TransitionDelay.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("transition-delay")

	// AnimationDuration
	c.AnimationDuration.Time = newValueFn[pseudoclass](nil, cfn).CustomSetter("animation-duration")
	c.AnimationDuration.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("animation-duration")

	// ListStyle
	c.ListStyle.ListStyleType = newValueFn[pseudoclass](nil, cfn).CustomSetter("list-style-type")
	c.ListStyle.ListStylePosition = newValueFn[pseudoclass](nil, cfn).CustomSetter("list-style-position")
	c.ListStyle.ListStyleImage = newValueFn[pseudoclass](nil, cfn).CustomSetter("list-style-image")
	c.ListStyle.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("list-style")

	// OutlineWidth
	c.OutlineWidth.Medium = newValueFn[pseudoclass](vfn("medium"), nil).Setter("outline-width")
	c.OutlineWidth.Thin = newValueFn[pseudoclass](vfn("thin"), nil).Setter("outline-width")
	c.OutlineWidth.Thick = newValueFn[pseudoclass](vfn("thick"), nil).Setter("outline-width")
	c.OutlineWidth.Length = newValueFn[pseudoclass](nil, cfn).CustomSetter("outline-width")
	c.OutlineWidth.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("outline-width")

	// BorderTopLeftRadius
	c.BorderTopLeftRadius.Length = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-top-left-radius")
	c.BorderTopLeftRadius.Percent = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-top-left-radius")
	c.BorderTopLeftRadius.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-top-left-radius")

	// WhiteSpace
	c.WhiteSpace.Normal = newValueFn[pseudoclass](vfn("normal"), nil).Setter("white-space")
	c.WhiteSpace.Nowrap = newValueFn[pseudoclass](vfn("nowrap"), nil).Setter("white-space")
	c.WhiteSpace.Pre = newValueFn[pseudoclass](vfn("pre"), nil).Setter("white-space")
	c.WhiteSpace.PreLine = newValueFn[pseudoclass](vfn("pre-line"), nil).Setter("white-space")
	c.WhiteSpace.PreWrap = newValueFn[pseudoclass](vfn("pre-wrap"), nil).Setter("white-space")
	c.WhiteSpace.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("white-space")

	// BorderRight
	c.BorderRight.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-right")

	// TextDecorationLine
	c.TextDecorationLine.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("text-decoration-line")
	c.TextDecorationLine.Underline = newValueFn[pseudoclass](vfn("underline"), nil).Setter("text-decoration-line")
	c.TextDecorationLine.Overline = newValueFn[pseudoclass](vfn("overline"), nil).Setter("text-decoration-line")
	c.TextDecorationLine.LineThrough = newValueFn[pseudoclass](vfn("line-through"), nil).Setter("text-decoration-line")
	c.TextDecorationLine.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("text-decoration-line")

	// AnimationDelay
	c.AnimationDelay.Time = newValueFn[pseudoclass](nil, cfn).CustomSetter("animation-delay")
	c.AnimationDelay.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("animation-delay")

	// BackgroundPosition
	c.BackgroundPosition.LeftTop = newValueFn[pseudoclass](vfn("left top"), nil).Setter("background-position")
	c.BackgroundPosition.LeftCenter = newValueFn[pseudoclass](vfn("left center"), nil).Setter("background-position")
	c.BackgroundPosition.LeftBottom = newValueFn[pseudoclass](vfn("left bottom"), nil).Setter("background-position")
	c.BackgroundPosition.RightTop = newValueFn[pseudoclass](vfn("right top"), nil).Setter("background-position")
	c.BackgroundPosition.RightCenter = newValueFn[pseudoclass](vfn("right center"), nil).Setter("background-position")
	c.BackgroundPosition.RightBottom = newValueFn[pseudoclass](vfn("right bottom"), nil).Setter("background-position")
	c.BackgroundPosition.CenterTop = newValueFn[pseudoclass](vfn("center top"), nil).Setter("background-position")
	c.BackgroundPosition.CenterCenter = newValueFn[pseudoclass](vfn("center center"), nil).Setter("background-position")
	c.BackgroundPosition.CenterBottom = newValueFn[pseudoclass](vfn("center bottom"), nil).Setter("background-position")
	c.BackgroundPosition.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("background-position")

	// BorderImage
	c.BorderImage.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-image")

	// BorderSpacing
	c.BorderSpacing.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-spacing")

	// BorderImageOutset
	c.BorderImageOutset.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-image-outset")

	// BorderImageSlice
	c.BorderImageSlice.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-image-slice")

	// BorderLeftColor
	c.BorderLeftColor.Color = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-left-color")
	c.BorderLeftColor.Transparent = newValueFn[pseudoclass](vfn("transparent"), nil).Setter("border-left-color")
	c.BorderLeftColor.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-left-color")

	// FontSize
	c.FontSize.Medium = newValueFn[pseudoclass](vfn("medium"), nil).Setter("font-size")
	c.FontSize.XxSmall = newValueFn[pseudoclass](vfn("xx-small"), nil).Setter("font-size")
	c.FontSize.XSmall = newValueFn[pseudoclass](vfn("x-small"), nil).Setter("font-size")
	c.FontSize.Small = newValueFn[pseudoclass](vfn("small"), nil).Setter("font-size")
	c.FontSize.Large = newValueFn[pseudoclass](vfn("large"), nil).Setter("font-size")
	c.FontSize.XLarge = newValueFn[pseudoclass](vfn("x-large"), nil).Setter("font-size")
	c.FontSize.XxLarge = newValueFn[pseudoclass](vfn("xx-large"), nil).Setter("font-size")
	c.FontSize.Smaller = newValueFn[pseudoclass](vfn("smaller"), nil).Setter("font-size")
	c.FontSize.Larger = newValueFn[pseudoclass](vfn("larger"), nil).Setter("font-size")
	c.FontSize.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("font-size")

	// LineHeight
	c.LineHeight.Normal = newValueFn[pseudoclass](vfn("normal"), nil).Setter("line-height")
	c.LineHeight.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("line-height")

	// TextDecorationStyle
	c.TextDecorationStyle.Solid = newValueFn[pseudoclass](vfn("solid"), nil).Setter("text-decoration-style")
	c.TextDecorationStyle.Double = newValueFn[pseudoclass](vfn("double"), nil).Setter("text-decoration-style")
	c.TextDecorationStyle.Dotted = newValueFn[pseudoclass](vfn("dotted"), nil).Setter("text-decoration-style")
	c.TextDecorationStyle.Dashed = newValueFn[pseudoclass](vfn("dashed"), nil).Setter("text-decoration-style")
	c.TextDecorationStyle.Wavy = newValueFn[pseudoclass](vfn("wavy"), nil).Setter("text-decoration-style")
	c.TextDecorationStyle.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("text-decoration-style")

	// BackfaceVisibility
	c.BackfaceVisibility.Visible = newValueFn[pseudoclass](vfn("visible"), nil).Setter("backface-visibility")
	c.BackfaceVisibility.Hidden = newValueFn[pseudoclass](vfn("hidden"), nil).Setter("backface-visibility")
	c.BackfaceVisibility.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("backface-visibility")

	// BorderRightStyle
	c.BorderRightStyle.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("border-right-style")
	c.BorderRightStyle.Hidden = newValueFn[pseudoclass](vfn("hidden"), nil).Setter("border-right-style")
	c.BorderRightStyle.Dotted = newValueFn[pseudoclass](vfn("dotted"), nil).Setter("border-right-style")
	c.BorderRightStyle.Dashed = newValueFn[pseudoclass](vfn("dashed"), nil).Setter("border-right-style")
	c.BorderRightStyle.Solid = newValueFn[pseudoclass](vfn("solid"), nil).Setter("border-right-style")
	c.BorderRightStyle.Double = newValueFn[pseudoclass](vfn("double"), nil).Setter("border-right-style")
	c.BorderRightStyle.Groove = newValueFn[pseudoclass](vfn("groove"), nil).Setter("border-right-style")
	c.BorderRightStyle.Ridge = newValueFn[pseudoclass](vfn("ridge"), nil).Setter("border-right-style")
	c.BorderRightStyle.Inset = newValueFn[pseudoclass](vfn("inset"), nil).Setter("border-right-style")
	c.BorderRightStyle.Outset = newValueFn[pseudoclass](vfn("outset"), nil).Setter("border-right-style")
	c.BorderRightStyle.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-right-style")

	// TextDecoration
	c.TextDecoration.TextDecorationLine = newValueFn[pseudoclass](nil, cfn).CustomSetter("text-decoration-line")
	c.TextDecoration.TextDecorationColor = newValueFn[pseudoclass](nil, cfn).CustomSetter("text-decoration-color")
	c.TextDecoration.TextDecorationStyle = newValueFn[pseudoclass](nil, cfn).CustomSetter("text-decoration-style")
	c.TextDecoration.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("text-decoration")

	// Transition
	c.Transition.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("transition")

	// AnimationIterationCount
	c.AnimationIterationCount.Infinite = newValueFn[pseudoclass](vfn("infinite"), nil).Setter("animation-iteration-count")
	c.AnimationIterationCount.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("animation-iteration-count")

	// BorderBottom
	c.BorderBottom.BorderBottomWidth = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-bottom-width")
	c.BorderBottom.BorderBottomStyle = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-bottom-style")
	c.BorderBottom.BorderBottomColor = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-bottom-color")
	c.BorderBottom.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-bottom")

	// AnimationTimingFunction
	c.AnimationTimingFunction.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("animation-timing-function")

	// BorderRadius
	c.BorderRadius.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-radius")

	// Quotes
	c.Quotes.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("quotes")
	c.Quotes.Auto = newValueFn[pseudoclass](vfn("auto"), nil).Setter("quotes")
	c.Quotes.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("quotes")

	// TabSize
	c.TabSize.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("tab-size")

	// AnimationFillMode
	c.AnimationFillMode.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("animation-fill-mode")
	c.AnimationFillMode.Forwards = newValueFn[pseudoclass](vfn("forwards"), nil).Setter("animation-fill-mode")
	c.AnimationFillMode.Backwards = newValueFn[pseudoclass](vfn("backwards"), nil).Setter("animation-fill-mode")
	c.AnimationFillMode.Both = newValueFn[pseudoclass](vfn("both"), nil).Setter("animation-fill-mode")
	c.AnimationFillMode.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("animation-fill-mode")

	// BackgroundSize
	c.BackgroundSize.Auto = newValueFn[pseudoclass](vfn("auto"), nil).Setter("background-size")
	c.BackgroundSize.Cover = newValueFn[pseudoclass](vfn("cover"), nil).Setter("background-size")
	c.BackgroundSize.Contain = newValueFn[pseudoclass](vfn("contain"), nil).Setter("background-size")
	c.BackgroundSize.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("background-size")

	// FontSizeAdjust
	c.FontSizeAdjust.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("font-size-adjust")
	c.FontSizeAdjust.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("font-size-adjust")

	// ListStylePosition
	c.ListStylePosition.Inside = newValueFn[pseudoclass](vfn("inside"), nil).Setter("list-style-position")
	c.ListStylePosition.Outside = newValueFn[pseudoclass](vfn("outside"), nil).Setter("list-style-position")
	c.ListStylePosition.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("list-style-position")

	// TextAlign
	c.TextAlign.Left = newValueFn[pseudoclass](vfn("left"), nil).Setter("text-align")
	c.TextAlign.Right = newValueFn[pseudoclass](vfn("right"), nil).Setter("text-align")
	c.TextAlign.Center = newValueFn[pseudoclass](vfn("center"), nil).Setter("text-align")
	c.TextAlign.Justify = newValueFn[pseudoclass](vfn("justify"), nil).Setter("text-align")
	c.TextAlign.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("text-align")

	// TextJustify
	c.TextJustify.Auto = newValueFn[pseudoclass](vfn("auto"), nil).Setter("text-justify")
	c.TextJustify.InterWord = newValueFn[pseudoclass](vfn("inter-word"), nil).Setter("text-justify")
	c.TextJustify.InterCharacter = newValueFn[pseudoclass](vfn("inter-character"), nil).Setter("text-justify")
	c.TextJustify.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("text-justify")
	c.TextJustify.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("text-justify")

	// BackgroundAttachment
	c.BackgroundAttachment.Scroll = newValueFn[pseudoclass](vfn("scroll"), nil).Setter("background-attachment")
	c.BackgroundAttachment.Fixed = newValueFn[pseudoclass](vfn("fixed"), nil).Setter("background-attachment")
	c.BackgroundAttachment.Local = newValueFn[pseudoclass](vfn("local"), nil).Setter("background-attachment")
	c.BackgroundAttachment.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("background-attachment")

	// BorderRightWidth
	c.BorderRightWidth.Medium = newValueFn[pseudoclass](vfn("medium"), nil).Setter("border-right-width")
	c.BorderRightWidth.Thin = newValueFn[pseudoclass](vfn("thin"), nil).Setter("border-right-width")
	c.BorderRightWidth.Thick = newValueFn[pseudoclass](vfn("thick"), nil).Setter("border-right-width")
	c.BorderRightWidth.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-right-width")

	// Font
	c.Font.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("font")

	// BorderLeft
	c.BorderLeft.BorderLeftWidth = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-left-width")
	c.BorderLeft.BorderLeftStyle = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-left-style")
	c.BorderLeft.BorderLeftColor = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-left-color")
	c.BorderLeft.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-left")

	// TransitionDuration
	c.TransitionDuration.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("transition-duration")

	// WordSpacing
	c.WordSpacing.Normal = newValueFn[pseudoclass](vfn("normal"), nil).Setter("word-spacing")
	c.WordSpacing.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("word-spacing")

	// AnimationName
	c.AnimationName.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("animation-name")
	c.AnimationName.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("animation-name")

	// AnimationPlayState
	c.AnimationPlayState.Paused = newValueFn[pseudoclass](vfn("paused"), nil).Setter("animation-play-state")
	c.AnimationPlayState.Running = newValueFn[pseudoclass](vfn("running"), nil).Setter("animation-play-state")
	c.AnimationPlayState.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("animation-play-state")

	// LetterSpacing
	c.LetterSpacing.Normal = newValueFn[pseudoclass](vfn("normal"), nil).Setter("letter-spacing")
	c.LetterSpacing.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("letter-spacing")

	// BorderBottomStyle
	c.BorderBottomStyle.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("border-bottom-style")
	c.BorderBottomStyle.Hidden = newValueFn[pseudoclass](vfn("hidden"), nil).Setter("border-bottom-style")
	c.BorderBottomStyle.Dotted = newValueFn[pseudoclass](vfn("dotted"), nil).Setter("border-bottom-style")
	c.BorderBottomStyle.Dashed = newValueFn[pseudoclass](vfn("dashed"), nil).Setter("border-bottom-style")
	c.BorderBottomStyle.Solid = newValueFn[pseudoclass](vfn("solid"), nil).Setter("border-bottom-style")
	c.BorderBottomStyle.Double = newValueFn[pseudoclass](vfn("double"), nil).Setter("border-bottom-style")
	c.BorderBottomStyle.Groove = newValueFn[pseudoclass](vfn("groove"), nil).Setter("border-bottom-style")
	c.BorderBottomStyle.Ridge = newValueFn[pseudoclass](vfn("ridge"), nil).Setter("border-bottom-style")
	c.BorderBottomStyle.Inset = newValueFn[pseudoclass](vfn("inset"), nil).Setter("border-bottom-style")
	c.BorderBottomStyle.Outset = newValueFn[pseudoclass](vfn("outset"), nil).Setter("border-bottom-style")
	c.BorderBottomStyle.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-bottom-style")

	// WordBreak
	c.WordBreak.Normal = newValueFn[pseudoclass](vfn("normal"), nil).Setter("word-break")
	c.WordBreak.BreakAll = newValueFn[pseudoclass](vfn("break-all"), nil).Setter("word-break")
	c.WordBreak.KeepAll = newValueFn[pseudoclass](vfn("keep-all"), nil).Setter("word-break")
	c.WordBreak.BreakWord = newValueFn[pseudoclass](vfn("break-word"), nil).Setter("word-break")
	c.WordBreak.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("word-break")

	// BorderBottomRightRadius
	c.BorderBottomRightRadius.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-bottom-right-radius")

	// FontStyle
	c.FontStyle.Normal = newValueFn[pseudoclass](vfn("normal"), nil).Setter("font-style")
	c.FontStyle.Italic = newValueFn[pseudoclass](vfn("italic"), nil).Setter("font-style")
	c.FontStyle.Oblique = newValueFn[pseudoclass](vfn("oblique"), nil).Setter("font-style")
	c.FontStyle.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("font-style")

	// Order
	c.Order.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("order")

	// OutlineStyle
	c.OutlineStyle.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("outline-style")
	c.OutlineStyle.Hidden = newValueFn[pseudoclass](vfn("hidden"), nil).Setter("outline-style")
	c.OutlineStyle.Dotted = newValueFn[pseudoclass](vfn("dotted"), nil).Setter("outline-style")
	c.OutlineStyle.Dashed = newValueFn[pseudoclass](vfn("dashed"), nil).Setter("outline-style")
	c.OutlineStyle.Solid = newValueFn[pseudoclass](vfn("solid"), nil).Setter("outline-style")
	c.OutlineStyle.Double = newValueFn[pseudoclass](vfn("double"), nil).Setter("outline-style")
	c.OutlineStyle.Groove = newValueFn[pseudoclass](vfn("groove"), nil).Setter("outline-style")
	c.OutlineStyle.Ridge = newValueFn[pseudoclass](vfn("ridge"), nil).Setter("outline-style")
	c.OutlineStyle.Inset = newValueFn[pseudoclass](vfn("inset"), nil).Setter("outline-style")
	c.OutlineStyle.Outset = newValueFn[pseudoclass](vfn("outset"), nil).Setter("outline-style")
	c.OutlineStyle.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("outline-style")

	// BorderBottomLeftRadius
	c.BorderBottomLeftRadius.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-bottom-left-radius")

	// BorderImageSource
	c.BorderImageSource.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("border-image-source")
	c.BorderImageSource.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-image-source")

	// TextAlignLast
	c.TextAlignLast.Auto = newValueFn[pseudoclass](vfn("auto"), nil).Setter("text-align-last")
	c.TextAlignLast.Left = newValueFn[pseudoclass](vfn("left"), nil).Setter("text-align-last")
	c.TextAlignLast.Right = newValueFn[pseudoclass](vfn("right"), nil).Setter("text-align-last")
	c.TextAlignLast.Center = newValueFn[pseudoclass](vfn("center"), nil).Setter("text-align-last")
	c.TextAlignLast.Justify = newValueFn[pseudoclass](vfn("justify"), nil).Setter("text-align-last")
	c.TextAlignLast.Start = newValueFn[pseudoclass](vfn("start"), nil).Setter("text-align-last")
	c.TextAlignLast.End = newValueFn[pseudoclass](vfn("end"), nil).Setter("text-align-last")
	c.TextAlignLast.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("text-align-last")

	// BorderImageWidth
	c.BorderImageWidth.Auto = newValueFn[pseudoclass](vfn("auto"), nil).Setter("border-image-width")
	c.BorderImageWidth.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-image-width")

	// FontWeight
	c.FontWeight.Normal = newValueFn[pseudoclass](vfn("normal"), nil).Setter("font-weight")
	c.FontWeight.Bold = newValueFn[pseudoclass](vfn("bold"), nil).Setter("font-weight")
	c.FontWeight.Bolder = newValueFn[pseudoclass](vfn("bolder"), nil).Setter("font-weight")
	c.FontWeight.Lighter = newValueFn[pseudoclass](vfn("lighter"), nil).Setter("font-weight")
	c.FontWeight.S100 = newValueFn[pseudoclass](vfn("100"), nil).Setter("font-weight")
	c.FontWeight.S200 = newValueFn[pseudoclass](vfn("200"), nil).Setter("font-weight")
	c.FontWeight.S300 = newValueFn[pseudoclass](vfn("300"), nil).Setter("font-weight")
	c.FontWeight.S400 = newValueFn[pseudoclass](vfn("400"), nil).Setter("font-weight")
	c.FontWeight.S500 = newValueFn[pseudoclass](vfn("500"), nil).Setter("font-weight")
	c.FontWeight.S600 = newValueFn[pseudoclass](vfn("600"), nil).Setter("font-weight")
	c.FontWeight.S700 = newValueFn[pseudoclass](vfn("700"), nil).Setter("font-weight")
	c.FontWeight.S800 = newValueFn[pseudoclass](vfn("800"), nil).Setter("font-weight")
	c.FontWeight.S900 = newValueFn[pseudoclass](vfn("900"), nil).Setter("font-weight")
	c.FontWeight.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("font-weight")

	// ListStyleImage
	c.ListStyleImage.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("list-style-image")
	c.ListStyleImage.Url = newValueFn[pseudoclass](nil, cfn).CustomSetter("list-style-image") // Special handling might be needed for URL
	c.ListStyleImage.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("list-style-image")

	// Opacity
	c.Opacity.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("opacity")

	// Clear
	c.Clear.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("clear")
	c.Clear.Left = newValueFn[pseudoclass](vfn("left"), nil).Setter("clear")
	c.Clear.Right = newValueFn[pseudoclass](vfn("right"), nil).Setter("clear")
	c.Clear.Both = newValueFn[pseudoclass](vfn("both"), nil).Setter("clear")
	c.Clear.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("clear")

	// BorderTopColor
	c.BorderTopColor.Transparent = newValueFn[pseudoclass](vfn("transparent"), nil).Setter("border-top-color")
	c.BorderTopColor.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-top-color")

	// Border
	c.Border.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("border")

	// BorderRightColor
	c.BorderRightColor.Transparent = newValueFn[pseudoclass](vfn("transparent"), nil).Setter("border-right-color")
	c.BorderRightColor.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-right-color")

	// TransitionTimingFunction
	c.TransitionTimingFunction.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("transition-timing-function")

	// BorderBottomWidth
	c.BorderBottomWidth.Medium = newValueFn[pseudoclass](vfn("medium"), nil).Setter("border-bottom-width")
	c.BorderBottomWidth.Thin = newValueFn[pseudoclass](vfn("thin"), nil).Setter("border-bottom-width")
	c.BorderBottomWidth.Thick = newValueFn[pseudoclass](vfn("thick"), nil).Setter("border-bottom-width")
	c.BorderBottomWidth.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-bottom-width")

	// BorderStyle
	c.BorderStyle.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("border-style")
	c.BorderStyle.Hidden = newValueFn[pseudoclass](vfn("hidden"), nil).Setter("border-style")
	c.BorderStyle.Dotted = newValueFn[pseudoclass](vfn("dotted"), nil).Setter("border-style")
	c.BorderStyle.Dashed = newValueFn[pseudoclass](vfn("dashed"), nil).Setter("border-style")
	c.BorderStyle.Solid = newValueFn[pseudoclass](vfn("solid"), nil).Setter("border-style")
	c.BorderStyle.Double = newValueFn[pseudoclass](vfn("double"), nil).Setter("border-style")
	c.BorderStyle.Groove = newValueFn[pseudoclass](vfn("groove"), nil).Setter("border-style")
	c.BorderStyle.Ridge = newValueFn[pseudoclass](vfn("ridge"), nil).Setter("border-style")
	c.BorderStyle.Inset = newValueFn[pseudoclass](vfn("inset"), nil).Setter("border-style")
	c.BorderStyle.Outset = newValueFn[pseudoclass](vfn("outset"), nil).Setter("border-style")
	c.BorderStyle.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-style")

	// BorderTopRightRadius
	c.BorderTopRightRadius.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-top-right-radius")

	// CaptionSide
	c.CaptionSide.Top = newValueFn[pseudoclass](vfn("top"), nil).Setter("caption-side")
	c.CaptionSide.Bottom = newValueFn[pseudoclass](vfn("bottom"), nil).Setter("caption-side")
	c.CaptionSide.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("caption-side")

	// FontFamily
	c.FontFamily.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("font-family")

	// TextDecorationColor
	c.TextDecorationColor.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("text-decoration-color")

	// TransitionProperty
	c.TransitionProperty.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("transition-property")
	c.TransitionProperty.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("transition-property")

	// BackgroundOrigin
	c.BackgroundOrigin.PaddingBox = newValueFn[pseudoclass](vfn("padding-box"), nil).Setter("background-origin")
	c.BackgroundOrigin.BorderBox = newValueFn[pseudoclass](vfn("border-box"), nil).Setter("background-origin")
	c.BackgroundOrigin.ContentBox = newValueFn[pseudoclass](vfn("content-box"), nil).Setter("background-origin")
	c.BackgroundOrigin.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("background-origin")

	// TextIndent
	c.TextIndent.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("text-indent")

	// Visibility
	c.Visibility.Visible = newValueFn[pseudoclass](vfn("visible"), nil).Setter("visibility")
	c.Visibility.Hidden = newValueFn[pseudoclass](vfn("hidden"), nil).Setter("visibility")
	c.Visibility.Collapse = newValueFn[pseudoclass](vfn("collapse"), nil).Setter("visibility")
	c.Visibility.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("visibility")

	// BorderColor
	c.BorderColor.Transparent = newValueFn[pseudoclass](vfn("transparent"), nil).Setter("border-color")
	c.BorderColor.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-color")

	// BorderTop
	c.BorderTop.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-top")

	// FontVariant
	c.FontVariant.Normal = newValueFn[pseudoclass](vfn("normal"), nil).Setter("font-variant")
	c.FontVariant.SmallCaps = newValueFn[pseudoclass](vfn("small-caps"), nil).Setter("font-variant")
	c.FontVariant.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("font-variant")

	// Outline
	c.Outline.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("outline")

	// BorderBottomColor
	c.BorderBottomColor.Transparent = newValueFn[pseudoclass](vfn("transparent"), nil).Setter("border-bottom-color")
	c.BorderBottomColor.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-bottom-color")

	// BorderTopStyle
	c.BorderTopStyle.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("border-top-style")
	c.BorderTopStyle.Hidden = newValueFn[pseudoclass](vfn("hidden"), nil).Setter("border-top-style")
	c.BorderTopStyle.Dotted = newValueFn[pseudoclass](vfn("dotted"), nil).Setter("border-top-style")
	c.BorderTopStyle.Dashed = newValueFn[pseudoclass](vfn("dashed"), nil).Setter("border-top-style")
	c.BorderTopStyle.Solid = newValueFn[pseudoclass](vfn("solid"), nil).Setter("border-top-style")
	c.BorderTopStyle.Double = newValueFn[pseudoclass](vfn("double"), nil).Setter("border-top-style")
	c.BorderTopStyle.Groove = newValueFn[pseudoclass](vfn("groove"), nil).Setter("border-top-style")
	c.BorderTopStyle.Ridge = newValueFn[pseudoclass](vfn("ridge"), nil).Setter("border-top-style")
	c.BorderTopStyle.Inset = newValueFn[pseudoclass](vfn("inset"), nil).Setter("border-top-style")
	c.BorderTopStyle.Outset = newValueFn[pseudoclass](vfn("outset"), nil).Setter("border-top-style")
	c.BorderTopStyle.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-top-style")

	// BorderWidth
	c.BorderWidth.Medium = newValueFn[pseudoclass](vfn("medium"), nil).Setter("border-width")
	c.BorderWidth.Thin = newValueFn[pseudoclass](vfn("thin"), nil).Setter("border-width")
	c.BorderWidth.Thick = newValueFn[pseudoclass](vfn("thick"), nil).Setter("border-width")
	c.BorderWidth.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-width")

	// ListStyleType
	c.ListStyleType.Disc = newValueFn[pseudoclass](vfn("disc"), nil).Setter("list-style-type")
	c.ListStyleType.Armenian = newValueFn[pseudoclass](vfn("armenian"), nil).Setter("list-style-type")
	c.ListStyleType.Circle = newValueFn[pseudoclass](vfn("circle"), nil).Setter("list-style-type")
	c.ListStyleType.CjkIdeographic = newValueFn[pseudoclass](vfn("cjk-ideographic"), nil).Setter("list-style-type")
	c.ListStyleType.Decimal = newValueFn[pseudoclass](vfn("decimal"), nil).Setter("list-style-type")
	c.ListStyleType.DecimalLeadingZero = newValueFn[pseudoclass](vfn("decimal-leading-zero"), nil).Setter("list-style-type")
	c.ListStyleType.Georgian = newValueFn[pseudoclass](vfn("georgian"), nil).Setter("list-style-type")
	c.ListStyleType.Hebrew = newValueFn[pseudoclass](vfn("hebrew"), nil).Setter("list-style-type")
	c.ListStyleType.Hiragana = newValueFn[pseudoclass](vfn("hiragana"), nil).Setter("list-style-type")
	c.ListStyleType.HiraganaIroha = newValueFn[pseudoclass](vfn("hiragana-iroha"), nil).Setter("list-style-type")
	c.ListStyleType.Katakana = newValueFn[pseudoclass](vfn("katakana"), nil).Setter("list-style-type")
	c.ListStyleType.KatakanaIroha = newValueFn[pseudoclass](vfn("katakana-iroha"), nil).Setter("list-style-type")
	c.ListStyleType.LowerAlpha = newValueFn[pseudoclass](vfn("lower-alpha"), nil).Setter("list-style-type")
	c.ListStyleType.LowerGreek = newValueFn[pseudoclass](vfn("lower-greek"), nil).Setter("list-style-type")
	c.ListStyleType.LowerLatin = newValueFn[pseudoclass](vfn("lower-latin"), nil).Setter("list-style-type")
	c.ListStyleType.LowerRoman = newValueFn[pseudoclass](vfn("lower-roman"), nil).Setter("list-style-type")
	c.ListStyleType.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("list-style-type")
	c.ListStyleType.Square = newValueFn[pseudoclass](vfn("square"), nil).Setter("list-style-type")
	c.ListStyleType.UpperAlpha = newValueFn[pseudoclass](vfn("upper-alpha"), nil).Setter("list-style-type")
	c.ListStyleType.UpperGreek = newValueFn[pseudoclass](vfn("upper-greek"), nil).Setter("list-style-type")
	c.ListStyleType.UpperLatin = newValueFn[pseudoclass](vfn("upper-latin"), nil).Setter("list-style-type")
	c.ListStyleType.UpperRoman = newValueFn[pseudoclass](vfn("upper-roman"), nil).Setter("list-style-type")
	c.ListStyleType.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("list-style-type")

	// OutlineOffset
	c.OutlineOffset.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("outline-offset")

	// Animation
	c.Animation.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("animation")

	// Background
	c.Background.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("background")

	// BackgroundRepeat
	c.BackgroundRepeat.Repeat = newValueFn[pseudoclass](vfn("repeat"), nil).Setter("background-repeat")
	c.BackgroundRepeat.RepeatX = newValueFn[pseudoclass](vfn("repeat-x"), nil).Setter("background-repeat")
	c.BackgroundRepeat.RepeatY = newValueFn[pseudoclass](vfn("repeat-y"), nil).Setter("background-repeat")
	c.BackgroundRepeat.NoRepeat = newValueFn[pseudoclass](vfn("no-repeat"), nil).Setter("background-repeat")
	c.BackgroundRepeat.Space = newValueFn[pseudoclass](vfn("space"), nil).Setter("background-repeat")
	c.BackgroundRepeat.Round = newValueFn[pseudoclass](vfn("round"), nil).Setter("background-repeat")
	c.BackgroundRepeat.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("background-repeat")

	// BorderTopWidth
	c.BorderTopWidth.Medium = newValueFn[pseudoclass](vfn("medium"), nil).Setter("border-top-width")
	c.BorderTopWidth.Thin = newValueFn[pseudoclass](vfn("thin"), nil).Setter("border-top-width")
	c.BorderTopWidth.Thick = newValueFn[pseudoclass](vfn("thick"), nil).Setter("border-top-width")
	c.BorderTopWidth.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-top-width")

	// WordWrap
	c.WordWrap.Normal = newValueFn[pseudoclass](vfn("normal"), nil).Setter("word-wrap")
	c.WordWrap.BreakWord = newValueFn[pseudoclass](vfn("break-word"), nil).Setter("word-wrap")
	c.WordWrap.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("word-wrap")

	// BackgroundColor
	c.BackgroundColor.Transparent = newValueFn[pseudoclass](vfn("transparent"), nil).Setter("background-color")
	c.BackgroundColor.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("background-color")

	// TextOverflow
	c.TextOverflow.Clip = newValueFn[pseudoclass](vfn("clip"), nil).Setter("text-overflow")
	c.TextOverflow.Ellipsis = newValueFn[pseudoclass](vfn("ellipsis"), nil).Setter("text-overflow")
	c.TextOverflow.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("text-overflow")

	// TextShadow
	c.TextShadow.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("text-shadow")
	c.TextShadow.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("text-shadow")

	// BackgroundClip
	c.BackgroundClip.BorderBox = newValueFn[pseudoclass](vfn("border-box"), nil).Setter("background-clip")
	c.BackgroundClip.PaddingBox = newValueFn[pseudoclass](vfn("padding-box"), nil).Setter("background-clip")
	c.BackgroundClip.ContentBox = newValueFn[pseudoclass](vfn("content-box"), nil).Setter("background-clip")
	c.BackgroundClip.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("background-clip")

	// BorderLeftWidth
	c.BorderLeftWidth.Medium = newValueFn[pseudoclass](vfn("medium"), nil).Setter("border-left-width")
	c.BorderLeftWidth.Thin = newValueFn[pseudoclass](vfn("thin"), nil).Setter("border-left-width")
	c.BorderLeftWidth.Thick = newValueFn[pseudoclass](vfn("thick"), nil).Setter("border-left-width")
	c.BorderLeftWidth.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-left-width")

	// Resize
	c.Resize.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("resize")
	c.Resize.Both = newValueFn[pseudoclass](vfn("both"), nil).Setter("resize")
	c.Resize.Horizontal = newValueFn[pseudoclass](vfn("horizontal"), nil).Setter("resize")
	c.Resize.Vertical = newValueFn[pseudoclass](vfn("vertical"), nil).Setter("resize")
	c.Resize.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("resize")

	// AnimationDirection
	c.AnimationDirection.Normal = newValueFn[pseudoclass](vfn("normal"), nil).Setter("animation-direction")
	c.AnimationDirection.Reverse = newValueFn[pseudoclass](vfn("reverse"), nil).Setter("animation-direction")
	c.AnimationDirection.Alternate = newValueFn[pseudoclass](vfn("alternate"), nil).Setter("animation-direction")
	c.AnimationDirection.AlternateReverse = newValueFn[pseudoclass](vfn("alternate-reverse"), nil).Setter("animation-direction")
	c.AnimationDirection.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("animation-direction")

	// Color
	c.Color.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("color")

	// OutlineColor
	c.OutlineColor.Invert = newValueFn[pseudoclass](vfn("invert"), nil).Setter("outline-color")
	c.OutlineColor.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("outline-color")

	// BorderImageRepeat
	c.BorderImageRepeat.Stretch = newValueFn[pseudoclass](vfn("stretch"), nil).Setter("border-image-repeat")
	c.BorderImageRepeat.Repeat = newValueFn[pseudoclass](vfn("repeat"), nil).Setter("border-image-repeat")
	c.BorderImageRepeat.Round = newValueFn[pseudoclass](vfn("round"), nil).Setter("border-image-repeat")
	c.BorderImageRepeat.Space = newValueFn[pseudoclass](vfn("space"), nil).Setter("border-image-repeat")
	c.BorderImageRepeat.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("border-image-repeat")

	// FontStretch
	c.FontStretch.UltraCondensed = newValueFn[pseudoclass](vfn("ultra-condensed"), nil).Setter("font-stretch")
	c.FontStretch.ExtraCondensed = newValueFn[pseudoclass](vfn("extra-condensed"), nil).Setter("font-stretch")
	c.FontStretch.Condensed = newValueFn[pseudoclass](vfn("condensed"), nil).Setter("font-stretch")
	c.FontStretch.SemiCondensed = newValueFn[pseudoclass](vfn("semi-condensed"), nil).Setter("font-stretch")
	c.FontStretch.Normal = newValueFn[pseudoclass](vfn("normal"), nil).Setter("font-stretch")
	c.FontStretch.SemiExpanded = newValueFn[pseudoclass](vfn("semi-expanded"), nil).Setter("font-stretch")
	c.FontStretch.Expanded = newValueFn[pseudoclass](vfn("expanded"), nil).Setter("font-stretch")
	c.FontStretch.ExtraExpanded = newValueFn[pseudoclass](vfn("extra-expanded"), nil).Setter("font-stretch")
	c.FontStretch.UltraExpanded = newValueFn[pseudoclass](vfn("ultra-expanded"), nil).Setter("font-stretch")
	c.FontStretch.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("font-stretch")

	// TextTransform
	c.TextTransform.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("text-transform")
	c.TextTransform.Capitalize = newValueFn[pseudoclass](vfn("capitalize"), nil).Setter("text-transform")
	c.TextTransform.Uppercase = newValueFn[pseudoclass](vfn("uppercase"), nil).Setter("text-transform")
	c.TextTransform.Lowercase = newValueFn[pseudoclass](vfn("lowercase"), nil).Setter("text-transform")
	c.TextTransform.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("text-transform")

	c.Margin.Top = newValueFn[pseudoclass](nil, cfn).CustomSetter("margin-top")
	c.Margin.Right = newValueFn[pseudoclass](nil, cfn).CustomSetter("margin-right")
	c.Margin.Bottom = newValueFn[pseudoclass](nil, cfn).CustomSetter("margin-bottom")
	c.Margin.Left = newValueFn[pseudoclass](nil, cfn).CustomSetter("margin-left")
	c.Margin.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("margin")

	c.Padding.Top = newValueFn[pseudoclass](nil, cfn).CustomSetter("padding-top")
	c.Padding.Right = newValueFn[pseudoclass](nil, cfn).CustomSetter("padding-right")
	c.Padding.Bottom = newValueFn[pseudoclass](nil, cfn).CustomSetter("padding-bottom")
	c.Padding.Left = newValueFn[pseudoclass](nil, cfn).CustomSetter("padding-left")
	c.Padding.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("padding")

	c.Transform.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("transform")

	c.PointerEvents.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("pointer-events")
	c.PointerEvents.Auto = newValueFn[pseudoclass](vfn("auto"), nil).Setter("pointer-events")
	c.PointerEvents.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("pointer-events")

	c.UserSelect.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("user-select")
	c.UserSelect.Text = newValueFn[pseudoclass](vfn("text"), nil).Setter("user-select")
	c.UserSelect.Auto = newValueFn[pseudoclass](vfn("auto"), nil).Setter("user-select")
	c.UserSelect.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("user-select")
	c.UserSelect.All = newValueFn[pseudoclass](vfn("all"), nil).Setter("user-select")
	c.UserSelect.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("user-select")

	c.BackdropFilter.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("backdrop-filter")

	c.ObjectFit.Fill = newValueFn[pseudoclass](vfn("fill"), nil).Setter("object-fit")
	c.ObjectFit.Contain = newValueFn[pseudoclass](vfn("contain"), nil).Setter("object-fit")
	c.ObjectFit.Cover = newValueFn[pseudoclass](vfn("cover"), nil).Setter("object-fit")
	c.ObjectFit.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("object-fit")
	c.ObjectFit.ScaleDown = newValueFn[pseudoclass](vfn("scaledown"), nil).Setter("object-fit")
	c.ObjectFit.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("object-fit")

	c.ObjectPosition.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("object-position")

	return c
}

func initializeContentLayout[pseudoclass any]() ContentLayout {
	c := ContentLayout{}

	// FlexGrow
	c.FlexGrow.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("flex-grow")

	// AlignSelf
	c.AlignSelf.Auto = newValueFn[pseudoclass](vfn("auto"), nil).Setter("align-self")
	c.AlignSelf.Stretch = newValueFn[pseudoclass](vfn("stretch"), nil).Setter("align-self")
	c.AlignSelf.Center = newValueFn[pseudoclass](vfn("center"), nil).Setter("align-self")
	c.AlignSelf.FlexStart = newValueFn[pseudoclass](vfn("flex-start"), nil).Setter("align-self")
	c.AlignSelf.FlexEnd = newValueFn[pseudoclass](vfn("flex-end"), nil).Setter("align-self")
	c.AlignSelf.Baseline = newValueFn[pseudoclass](vfn("baseline"), nil).Setter("align-self")
	c.AlignSelf.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("align-self")

	// Content
	c.Content.Normal = newValueFn[pseudoclass](vfn("normal"), nil).Setter("content")
	c.Content.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("content")
	c.Content.Counter = newValueFn[pseudoclass](vfn("counter"), nil).Setter("content")
	c.Content.Attr = newValueFn[pseudoclass](nil, cfn).CustomSetter("content") // Adjust for attribute-based content.
	c.Content.String = newValueFn[pseudoclass](vfn("string"), nil).Setter("content")
	c.Content.OpenQuote = newValueFn[pseudoclass](vfn("open-quote"), nil).Setter("content")
	c.Content.CloseQuote = newValueFn[pseudoclass](vfn("close-quote"), nil).Setter("content")
	c.Content.NoOpenQuote = newValueFn[pseudoclass](vfn("no-open-quote"), nil).Setter("content")
	c.Content.NoCloseQuote = newValueFn[pseudoclass](vfn("no-close-quote"), nil).Setter("content")
	c.Content.URL = newValueFn[pseudoclass](nil, cfn).CustomSetter("content") // Adjust for URL-based content.
	c.Content.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("content")

	// ColumnSpan
	c.ColumnSpan.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("column-span")
	c.ColumnSpan.All = newValueFn[pseudoclass](vfn("all"), nil).Setter("column-span")
	c.ColumnSpan.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("column-span")

	// Flex
	c.Flex.FlexGrow = newValueFn[pseudoclass](nil, cfn).CustomSetter("flex-grow")
	c.Flex.FlexShrink = newValueFn[pseudoclass](nil, cfn).CustomSetter("flex-shrink")
	c.Flex.FlexBasis = newValueFn[pseudoclass](nil, cfn).CustomSetter("flex-basis")
	c.Flex.Auto = newValueFn[pseudoclass](vfn("auto"), nil).Setter("flex")
	c.Flex.Initial = newValueFn[pseudoclass](vfn("initial"), nil).Setter("flex")
	c.Flex.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("flex")
	c.Flex.Inherit = newValueFn[pseudoclass](vfn("inherit"), nil).Setter("flex")

	// FlexShrink
	c.FlexShrink.Number = newValueFn[pseudoclass](nil, cfn).CustomSetter("flex-shrink")
	c.FlexShrink.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("flex-shrink")

	// Order
	c.Order.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("order")

	// FlexBasis
	c.FlexBasis.Auto = newValueFn[pseudoclass](vfn("auto"), nil).Setter("flex-basis")
	c.FlexBasis.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("flex-basis")

	// AlignItems
	c.AlignItems.Stretch = newValueFn[pseudoclass](vfn("stretch"), nil).Setter("align-items")
	c.AlignItems.Center = newValueFn[pseudoclass](vfn("center"), nil).Setter("align-items")
	c.AlignItems.FlexStart = newValueFn[pseudoclass](vfn("flex-start"), nil).Setter("align-items")
	c.AlignItems.FlexEnd = newValueFn[pseudoclass](vfn("flex-end"), nil).Setter("align-items")
	c.AlignItems.Baseline = newValueFn[pseudoclass](vfn("baseline"), nil).Setter("align-items")
	c.AlignItems.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("align-items")

	return c
}

func initializeContentStyle[pseudoclass any]() ContentStyle {
	c := ContentStyle{}

	// ColumnRuleWidth initialization
	c.ColumnRuleWidth.Medium = newValueFn[pseudoclass](vfn("medium"), nil).Setter("column-rule-width")
	c.ColumnRuleWidth.Thin = newValueFn[pseudoclass](vfn("thin"), nil).Setter("column-rule-width")
	c.ColumnRuleWidth.Thick = newValueFn[pseudoclass](vfn("thick"), nil).Setter("column-rule-width")
	c.ColumnRuleWidth.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("column-rule-width")

	// ColumnRule initialization
	c.ColumnRule.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("column-rule")

	// Direction initialization
	c.Direction.Ltr = newValueFn[pseudoclass](vfn("ltr"), nil).Setter("direction")
	c.Direction.Rtl = newValueFn[pseudoclass](vfn("rtl"), nil).Setter("direction")
	c.Direction.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("direction")

	// ColumnRuleStyle initialization
	c.ColumnRuleStyle.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("column-rule-style")
	c.ColumnRuleStyle.Hidden = newValueFn[pseudoclass](vfn("hidden"), nil).Setter("column-rule-style")
	c.ColumnRuleStyle.Dotted = newValueFn[pseudoclass](vfn("dotted"), nil).Setter("column-rule-style")
	c.ColumnRuleStyle.Dashed = newValueFn[pseudoclass](vfn("dashed"), nil).Setter("column-rule-style")
	c.ColumnRuleStyle.Solid = newValueFn[pseudoclass](vfn("solid"), nil).Setter("column-rule-style")
	c.ColumnRuleStyle.Double = newValueFn[pseudoclass](vfn("double"), nil).Setter("column-rule-style")
	c.ColumnRuleStyle.Groove = newValueFn[pseudoclass](vfn("groove"), nil).Setter("column-rule-style")
	c.ColumnRuleStyle.Ridge = newValueFn[pseudoclass](vfn("ridge"), nil).Setter("column-rule-style")
	c.ColumnRuleStyle.Inset = newValueFn[pseudoclass](vfn("inset"), nil).Setter("column-rule-style")
	c.ColumnRuleStyle.Outset = newValueFn[pseudoclass](vfn("outset"), nil).Setter("column-rule-style")
	c.ColumnRuleStyle.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("column-rule-style")

	// ColumnRuleColor initialization
	c.ColumnRuleColor.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("column-rule-color")

	// ColumnFill initialization
	c.ColumnFill.Balance = newValueFn[pseudoclass](vfn("balance"), nil).Setter("column-fill")
	c.ColumnFill.Auto = newValueFn[pseudoclass](vfn("auto"), nil).Setter("column-fill")
	c.ColumnFill.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("column-fill")

	// EmptyCells initialization
	c.EmptyCells.Show = newValueFn[pseudoclass](vfn("show"), nil).Setter("empty-cells")
	c.EmptyCells.Hide = newValueFn[pseudoclass](vfn("hide"), nil).Setter("empty-cells")
	c.EmptyCells.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("empty-cells")

	// Cursor initialization
	c.Cursor.Alias = newValueFn[pseudoclass](vfn("alias"), nil).Setter("cursor")
	c.Cursor.AllScroll = newValueFn[pseudoclass](vfn("all-scroll"), nil).Setter("cursor")
	c.Cursor.Auto = newValueFn[pseudoclass](vfn("auto"), nil).Setter("cursor")
	c.Cursor.Cell = newValueFn[pseudoclass](vfn("cell"), nil).Setter("cursor")
	c.Cursor.ContextMenu = newValueFn[pseudoclass](vfn("context-menu"), nil).Setter("cursor")
	c.Cursor.ColResize = newValueFn[pseudoclass](vfn("col-resize"), nil).Setter("cursor")
	c.Cursor.Copy = newValueFn[pseudoclass](vfn("copy"), nil).Setter("cursor")
	c.Cursor.Crosshair = newValueFn[pseudoclass](vfn("crosshair"), nil).Setter("cursor")
	c.Cursor.Default = newValueFn[pseudoclass](vfn("default"), nil).Setter("cursor")
	c.Cursor.EResize = newValueFn[pseudoclass](vfn("e-resize"), nil).Setter("cursor")
	c.Cursor.EwResize = newValueFn[pseudoclass](vfn("ew-resize"), nil).Setter("cursor")
	c.Cursor.Grab = newValueFn[pseudoclass](vfn("grab"), nil).Setter("cursor")
	c.Cursor.Grabbing = newValueFn[pseudoclass](vfn("grabbing"), nil).Setter("cursor")
	c.Cursor.Help = newValueFn[pseudoclass](vfn("help"), nil).Setter("cursor")
	c.Cursor.Move = newValueFn[pseudoclass](vfn("move"), nil).Setter("cursor")
	c.Cursor.NResize = newValueFn[pseudoclass](vfn("n-resize"), nil).Setter("cursor")
	c.Cursor.NeResize = newValueFn[pseudoclass](vfn("ne-resize"), nil).Setter("cursor")
	c.Cursor.NeswResize = newValueFn[pseudoclass](vfn("nesw-resize"), nil).Setter("cursor")
	c.Cursor.NsResize = newValueFn[pseudoclass](vfn("ns-resize"), nil).Setter("cursor")
	c.Cursor.NwResize = newValueFn[pseudoclass](vfn("nw-resize"), nil).Setter("cursor")
	c.Cursor.NwseResize = newValueFn[pseudoclass](vfn("nwse-resize"), nil).Setter("cursor")
	c.Cursor.NoDrop = newValueFn[pseudoclass](vfn("no-drop"), nil).Setter("cursor")
	c.Cursor.None = newValueFn[pseudoclass](vfn("none"), nil).Setter("cursor")
	c.Cursor.NotAllowed = newValueFn[pseudoclass](vfn("not-allowed"), nil).Setter("cursor")
	c.Cursor.Pointer = newValueFn[pseudoclass](vfn("pointer"), nil).Setter("cursor")
	c.Cursor.Progress = newValueFn[pseudoclass](vfn("progress"), nil).Setter("cursor")
	c.Cursor.RowResize = newValueFn[pseudoclass](vfn("row-resize"), nil).Setter("cursor")
	c.Cursor.SResize = newValueFn[pseudoclass](vfn("s-resize"), nil).Setter("cursor")
	c.Cursor.SeResize = newValueFn[pseudoclass](vfn("se-resize"), nil).Setter("cursor")
	c.Cursor.SwResize = newValueFn[pseudoclass](vfn("sw-resize"), nil).Setter("cursor")
	c.Cursor.Text = newValueFn[pseudoclass](vfn("text"), nil).Setter("cursor")
	c.Cursor.VerticalText = newValueFn[pseudoclass](vfn("vertical-text"), nil).Setter("cursor")
	c.Cursor.WResize = newValueFn[pseudoclass](vfn("w-resize"), nil).Setter("cursor")
	c.Cursor.Wait = newValueFn[pseudoclass](vfn("wait"), nil).Setter("cursor")
	c.Cursor.ZoomIn = newValueFn[pseudoclass](vfn("zoom-in"), nil).Setter("cursor")
	c.Cursor.ZoomOut = newValueFn[pseudoclass](vfn("zoom-out"), nil).Setter("cursor")
	c.Cursor.Value = newValueFn[pseudoclass](nil, cfn).CustomSetter("cursor")

	return c
}

type ContainerLayout struct {
	BoxShadow struct {
		None  func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	JustifyContent struct {
		FlexStart    func(*ui.Element) *ui.Element
		FlexEnd      func(*ui.Element) *ui.Element
		Center       func(*ui.Element) *ui.Element
		SpaceBetween func(*ui.Element) *ui.Element
		SpaceAround  func(*ui.Element) *ui.Element
		Value        func(value string) func(*ui.Element) *ui.Element
	}
	ZIndex struct {
		Auto  func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Float struct {
		None  func(*ui.Element) *ui.Element
		Left  func(*ui.Element) *ui.Element
		Right func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Overflow struct {
		Visible func(*ui.Element) *ui.Element
		Hidden  func(*ui.Element) *ui.Element
		Scroll  func(*ui.Element) *ui.Element
		Auto    func(*ui.Element) *ui.Element
		Value   func(value string) func(*ui.Element) *ui.Element
	}
	OverflowY struct {
		Visible func(*ui.Element) *ui.Element
		Hidden  func(*ui.Element) *ui.Element
		Scroll  func(*ui.Element) *ui.Element
		Auto    func(*ui.Element) *ui.Element
		Value   func(value string) func(*ui.Element) *ui.Element
	}
	Perspective struct {
		None  func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderCollapse struct {
		Separate func(*ui.Element) *ui.Element
		Collapse func(*ui.Element) *ui.Element
		Value    func(value string) func(*ui.Element) *ui.Element
	}
	PageBreakBefore struct {
		Auto   func(*ui.Element) *ui.Element
		Always func(*ui.Element) *ui.Element
		Avoid  func(*ui.Element) *ui.Element
		Left   func(*ui.Element) *ui.Element
		Right  func(*ui.Element) *ui.Element
		Value  func(value string) func(*ui.Element) *ui.Element
	}
	Columns struct {
		Auto  func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	ColumnCount struct {
		Auto  func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	MinHeight struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	PageBreakInside struct {
		Auto  func(*ui.Element) *ui.Element
		Avoid func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	ColumnGap struct {
		Length func(value string) func(*ui.Element) *ui.Element
		Normal func(*ui.Element) *ui.Element
		Value  func(value string) func(*ui.Element) *ui.Element
	}
	Clip struct {
		Auto  func(*ui.Element) *ui.Element
		Shape func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	FlexDirection struct {
		Row           func(*ui.Element) *ui.Element
		RowReverse    func(*ui.Element) *ui.Element
		Column        func(*ui.Element) *ui.Element
		ColumnReverse func(*ui.Element) *ui.Element
		Value         func(value string) func(*ui.Element) *ui.Element
	}
	PageBreakAfter struct {
		Auto   func(*ui.Element) *ui.Element
		Always func(*ui.Element) *ui.Element
		Avoid  func(*ui.Element) *ui.Element
		Left   func(*ui.Element) *ui.Element
		Right  func(*ui.Element) *ui.Element
		Value  func(value string) func(*ui.Element) *ui.Element
	}
	Top struct {
		Auto  func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	CounterIncrement struct {
		None   func(*ui.Element) *ui.Element
		Number func(value string) func(*ui.Element) *ui.Element
		Value  func(value string) func(*ui.Element) *ui.Element
	}
	Height struct {
		Auto  func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	TransformStyle struct {
		Flat       func(*ui.Element) *ui.Element
		Preserve3d func(*ui.Element) *ui.Element
		Value      func(value string) func(*ui.Element) *ui.Element
	}
	OverflowX struct {
		Visible func(*ui.Element) *ui.Element
		Hidden  func(*ui.Element) *ui.Element
		Scroll  func(*ui.Element) *ui.Element
		Auto    func(*ui.Element) *ui.Element
		Value   func(value string) func(*ui.Element) *ui.Element
	}
	FlexWrap struct {
		Nowrap      func(*ui.Element) *ui.Element
		Wrap        func(*ui.Element) *ui.Element
		WrapReverse func(*ui.Element) *ui.Element
		Value       func(value string) func(*ui.Element) *ui.Element
	}
	MaxWidth struct {
		None  func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Bottom struct {
		Auto  func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	CounterReset struct {
		None  func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Right struct {
		Auto  func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BoxSizing struct {
		ContentBox func(*ui.Element) *ui.Element
		BorderBox  func(*ui.Element) *ui.Element
		Value      func(value string) func(*ui.Element) *ui.Element
	}
	Position struct {
		Static   func(*ui.Element) *ui.Element
		Absolute func(*ui.Element) *ui.Element
		Fixed    func(*ui.Element) *ui.Element
		Relative func(*ui.Element) *ui.Element
		Sticky   func(*ui.Element) *ui.Element
		Value    func(value string) func(*ui.Element) *ui.Element
	}
	TableLayout struct {
		Auto  func(*ui.Element) *ui.Element
		Fixed func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Width struct {
		Auto  func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	MaxHeight struct {
		None  func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	ColumnWidth struct {
		Auto  func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	MinWidth struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	VerticalAlign struct {
		Baseline   func(*ui.Element) *ui.Element
		Top        func(*ui.Element) *ui.Element
		TextTop    func(*ui.Element) *ui.Element
		Middle     func(*ui.Element) *ui.Element
		Bottom     func(*ui.Element) *ui.Element
		TextBottom func(*ui.Element) *ui.Element
		Value      func(value string) func(*ui.Element) *ui.Element
	}
	PerspectiveOrigin struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	AlignContent struct {
		Stretch      func(*ui.Element) *ui.Element
		Center       func(*ui.Element) *ui.Element
		FlexStart    func(*ui.Element) *ui.Element
		FlexEnd      func(*ui.Element) *ui.Element
		SpaceBetween func(*ui.Element) *ui.Element
		SpaceAround  func(*ui.Element) *ui.Element
		Value        func(value string) func(*ui.Element) *ui.Element
	}
	FlexFlow struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Display struct {
		Inline   func(*ui.Element) *ui.Element
		Block    func(*ui.Element) *ui.Element
		Contents func(*ui.Element) *ui.Element
		Flex     func(*ui.Element) *ui.Element
		Grid     func(*ui.Element) *ui.Element
		None     func(*ui.Element) *ui.Element
		Value    func(value string) func(*ui.Element) *ui.Element
	}
	Left struct {
		Auto  func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	GridTemplateColumns struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	GridTemplateRows struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	GridColumn struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	GridRow struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Gap struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	ScrollBehavior struct {
		Auto   func(*ui.Element) *ui.Element
		Smooth func(*ui.Element) *ui.Element
		Value  func(value string) func(*ui.Element) *ui.Element
	}
}

type ContainerStyle struct {
	BackgroundImage struct {
		URL   func(url string) func(*ui.Element) *ui.Element
		None  func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderLeftStyle struct {
		None   func(*ui.Element) *ui.Element
		Hidden func(*ui.Element) *ui.Element
		Dotted func(*ui.Element) *ui.Element
		Dashed func(*ui.Element) *ui.Element
		Solid  func(*ui.Element) *ui.Element
		Double func(*ui.Element) *ui.Element
		Groove func(*ui.Element) *ui.Element
		Ridge  func(*ui.Element) *ui.Element
		Inset  func(*ui.Element) *ui.Element
		Outset func(*ui.Element) *ui.Element
		Value  func(value string) func(*ui.Element) *ui.Element
	}
	BoxShadow struct {
		None  func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	TransitionDelay struct {
		Time  func(duration string) func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	AnimationDuration struct {
		Time  func(duration string) func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	ListStyle struct {
		ListStyleType     func(value string) func(*ui.Element) *ui.Element
		ListStylePosition func(value string) func(*ui.Element) *ui.Element
		ListStyleImage    func(value string) func(*ui.Element) *ui.Element
		Value             func(value string) func(*ui.Element) *ui.Element
	}
	OutlineWidth struct {
		Medium func(*ui.Element) *ui.Element
		Thin   func(*ui.Element) *ui.Element
		Thick  func(*ui.Element) *ui.Element
		Length func(value string) func(*ui.Element) *ui.Element
		Value  func(value string) func(*ui.Element) *ui.Element
	}
	BorderTopLeftRadius struct {
		Length  func(value string) func(*ui.Element) *ui.Element
		Percent func(value string) func(*ui.Element) *ui.Element
		Value   func(value string) func(*ui.Element) *ui.Element
	}
	WhiteSpace struct {
		Normal  func(*ui.Element) *ui.Element
		Nowrap  func(*ui.Element) *ui.Element
		Pre     func(*ui.Element) *ui.Element
		PreLine func(*ui.Element) *ui.Element
		PreWrap func(*ui.Element) *ui.Element
		Value   func(value string) func(*ui.Element) *ui.Element
	}
	BorderRight struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	TextDecorationLine struct {
		None        func(*ui.Element) *ui.Element
		Underline   func(*ui.Element) *ui.Element
		Overline    func(*ui.Element) *ui.Element
		LineThrough func(*ui.Element) *ui.Element
		Value       func(value string) func(*ui.Element) *ui.Element
	}
	AnimationDelay struct {
		Time  func(duration string) func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BackgroundPosition struct {
		LeftTop      func(*ui.Element) *ui.Element
		LeftCenter   func(*ui.Element) *ui.Element
		LeftBottom   func(*ui.Element) *ui.Element
		RightTop     func(*ui.Element) *ui.Element
		RightCenter  func(*ui.Element) *ui.Element
		RightBottom  func(*ui.Element) *ui.Element
		CenterTop    func(*ui.Element) *ui.Element
		CenterCenter func(*ui.Element) *ui.Element
		CenterBottom func(*ui.Element) *ui.Element
		Value        func(value string) func(*ui.Element) *ui.Element
	}
	BorderImage struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderSpacing struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderImageOutset struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderImageSlice struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderLeftColor struct {
		Color       func(value string) func(*ui.Element) *ui.Element
		Transparent func(*ui.Element) *ui.Element
		Value       func(value string) func(*ui.Element) *ui.Element
	}
	FontSize struct {
		Medium  func(*ui.Element) *ui.Element
		XxSmall func(*ui.Element) *ui.Element
		XSmall  func(*ui.Element) *ui.Element
		Small   func(*ui.Element) *ui.Element
		Large   func(*ui.Element) *ui.Element
		XLarge  func(*ui.Element) *ui.Element
		XxLarge func(*ui.Element) *ui.Element
		Smaller func(*ui.Element) *ui.Element
		Larger  func(*ui.Element) *ui.Element
		Value   func(value string) func(*ui.Element) *ui.Element
	}
	LineHeight struct {
		Normal func(*ui.Element) *ui.Element
		Value  func(value string) func(*ui.Element) *ui.Element
	}
	TextDecorationStyle struct {
		Solid  func(*ui.Element) *ui.Element
		Double func(*ui.Element) *ui.Element
		Dotted func(*ui.Element) *ui.Element
		Dashed func(*ui.Element) *ui.Element
		Wavy   func(*ui.Element) *ui.Element
		Value  func(value string) func(*ui.Element) *ui.Element
	}
	BackfaceVisibility struct {
		Visible func(*ui.Element) *ui.Element
		Hidden  func(*ui.Element) *ui.Element
		Value   func(value string) func(*ui.Element) *ui.Element
	}
	BorderRightStyle struct {
		None   func(*ui.Element) *ui.Element
		Hidden func(*ui.Element) *ui.Element
		Dotted func(*ui.Element) *ui.Element
		Dashed func(*ui.Element) *ui.Element
		Solid  func(*ui.Element) *ui.Element
		Double func(*ui.Element) *ui.Element
		Groove func(*ui.Element) *ui.Element
		Ridge  func(*ui.Element) *ui.Element
		Inset  func(*ui.Element) *ui.Element
		Outset func(*ui.Element) *ui.Element
		Value  func(value string) func(*ui.Element) *ui.Element
	}
	TextDecoration struct {
		TextDecorationLine  func(value string) func(*ui.Element) *ui.Element
		TextDecorationColor func(value string) func(*ui.Element) *ui.Element
		TextDecorationStyle func(value string) func(*ui.Element) *ui.Element
		Value               func(value string) func(*ui.Element) *ui.Element
	}
	Transition struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	AnimationIterationCount struct {
		Infinite func(*ui.Element) *ui.Element
		Value    func(value string) func(*ui.Element) *ui.Element
	}
	BorderBottom struct {
		BorderBottomWidth func(value string) func(*ui.Element) *ui.Element
		BorderBottomStyle func(value string) func(*ui.Element) *ui.Element
		BorderBottomColor func(value string) func(*ui.Element) *ui.Element
		Value             func(value string) func(*ui.Element) *ui.Element
	}
	AnimationTimingFunction struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderRadius struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Quotes struct {
		None  func(*ui.Element) *ui.Element
		Auto  func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	TabSize struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	AnimationFillMode struct {
		None      func(*ui.Element) *ui.Element
		Forwards  func(*ui.Element) *ui.Element
		Backwards func(*ui.Element) *ui.Element
		Both      func(*ui.Element) *ui.Element
		Value     func(value string) func(*ui.Element) *ui.Element
	}
	BackgroundSize struct {
		Auto    func(*ui.Element) *ui.Element
		Cover   func(*ui.Element) *ui.Element
		Contain func(*ui.Element) *ui.Element
		Value   func(value string) func(*ui.Element) *ui.Element
	}
	FontSizeAdjust struct {
		None  func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	ListStylePosition struct {
		Inside  func(*ui.Element) *ui.Element
		Outside func(*ui.Element) *ui.Element
		Value   func(value string) func(*ui.Element) *ui.Element
	}
	TextAlign struct {
		Left    func(*ui.Element) *ui.Element
		Right   func(*ui.Element) *ui.Element
		Center  func(*ui.Element) *ui.Element
		Justify func(*ui.Element) *ui.Element
		Value   func(value string) func(*ui.Element) *ui.Element
	}
	TextJustify struct {
		Auto           func(*ui.Element) *ui.Element
		InterWord      func(*ui.Element) *ui.Element
		InterCharacter func(*ui.Element) *ui.Element
		None           func(*ui.Element) *ui.Element
		Value          func(value string) func(*ui.Element) *ui.Element
	}
	BackgroundAttachment struct {
		Scroll func(*ui.Element) *ui.Element
		Fixed  func(*ui.Element) *ui.Element
		Local  func(*ui.Element) *ui.Element
		Value  func(value string) func(*ui.Element) *ui.Element
	}
	BorderRightWidth struct {
		Medium func(*ui.Element) *ui.Element
		Thin   func(*ui.Element) *ui.Element
		Thick  func(*ui.Element) *ui.Element
		Value  func(value string) func(*ui.Element) *ui.Element
	}
	Font struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderLeft struct {
		BorderLeftWidth func(value string) func(*ui.Element) *ui.Element
		BorderLeftStyle func(value string) func(*ui.Element) *ui.Element
		BorderLeftColor func(value string) func(*ui.Element) *ui.Element
		Value           func(value string) func(*ui.Element) *ui.Element
	}
	TransitionDuration struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	WordSpacing struct {
		Normal func(*ui.Element) *ui.Element
		Value  func(value string) func(*ui.Element) *ui.Element
	}
	AnimationName struct {
		None  func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	AnimationPlayState struct {
		Paused  func(*ui.Element) *ui.Element
		Running func(*ui.Element) *ui.Element
		Value   func(value string) func(*ui.Element) *ui.Element
	}
	LetterSpacing struct {
		Normal func(*ui.Element) *ui.Element
		Value  func(value string) func(*ui.Element) *ui.Element
	}
	BorderBottomStyle struct {
		None   func(*ui.Element) *ui.Element
		Hidden func(*ui.Element) *ui.Element
		Dotted func(*ui.Element) *ui.Element
		Dashed func(*ui.Element) *ui.Element
		Solid  func(*ui.Element) *ui.Element
		Double func(*ui.Element) *ui.Element
		Groove func(*ui.Element) *ui.Element
		Ridge  func(*ui.Element) *ui.Element
		Inset  func(*ui.Element) *ui.Element
		Outset func(*ui.Element) *ui.Element
		Value  func(value string) func(*ui.Element) *ui.Element
	}
	WordBreak struct {
		Normal    func(*ui.Element) *ui.Element
		BreakAll  func(*ui.Element) *ui.Element
		KeepAll   func(*ui.Element) *ui.Element
		BreakWord func(*ui.Element) *ui.Element
		Value     func(value string) func(*ui.Element) *ui.Element
	}
	BorderBottomRightRadius struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	FontStyle struct {
		Normal  func(*ui.Element) *ui.Element
		Italic  func(*ui.Element) *ui.Element
		Oblique func(*ui.Element) *ui.Element
		Value   func(value string) func(*ui.Element) *ui.Element
	}
	Order struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	OutlineStyle struct {
		None   func(*ui.Element) *ui.Element
		Hidden func(*ui.Element) *ui.Element
		Dotted func(*ui.Element) *ui.Element
		Dashed func(*ui.Element) *ui.Element
		Solid  func(*ui.Element) *ui.Element
		Double func(*ui.Element) *ui.Element
		Groove func(*ui.Element) *ui.Element
		Ridge  func(*ui.Element) *ui.Element
		Inset  func(*ui.Element) *ui.Element
		Outset func(*ui.Element) *ui.Element
		Value  func(value string) func(*ui.Element) *ui.Element
	}
	BorderBottomLeftRadius struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderImageSource struct {
		None  func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	TextAlignLast struct {
		Auto    func(*ui.Element) *ui.Element
		Left    func(*ui.Element) *ui.Element
		Right   func(*ui.Element) *ui.Element
		Center  func(*ui.Element) *ui.Element
		Justify func(*ui.Element) *ui.Element
		Start   func(*ui.Element) *ui.Element
		End     func(*ui.Element) *ui.Element
		Value   func(value string) func(*ui.Element) *ui.Element
	}
	BorderImageWidth struct {
		Auto  func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	FontWeight struct {
		Normal  func(*ui.Element) *ui.Element
		Bold    func(*ui.Element) *ui.Element
		Bolder  func(*ui.Element) *ui.Element
		Lighter func(*ui.Element) *ui.Element
		S100    func(*ui.Element) *ui.Element
		S200    func(*ui.Element) *ui.Element
		S300    func(*ui.Element) *ui.Element
		S400    func(*ui.Element) *ui.Element
		S500    func(*ui.Element) *ui.Element
		S600    func(*ui.Element) *ui.Element
		S700    func(*ui.Element) *ui.Element
		S800    func(*ui.Element) *ui.Element
		S900    func(*ui.Element) *ui.Element
		Value   func(value string) func(*ui.Element) *ui.Element
	}
	ListStyleImage struct {
		None  func(*ui.Element) *ui.Element
		Url   func(value string) func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Opacity struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Clear struct {
		None  func(*ui.Element) *ui.Element
		Left  func(*ui.Element) *ui.Element
		Right func(*ui.Element) *ui.Element
		Both  func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderTopColor struct {
		Transparent func(*ui.Element) *ui.Element
		Value       func(value string) func(*ui.Element) *ui.Element
	}
	Border struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderRightColor struct {
		Transparent func(*ui.Element) *ui.Element
		Value       func(value string) func(*ui.Element) *ui.Element
	}
	TransitionTimingFunction struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderBottomWidth struct {
		Medium func(*ui.Element) *ui.Element
		Thin   func(*ui.Element) *ui.Element
		Thick  func(*ui.Element) *ui.Element
		Value  func(value string) func(*ui.Element) *ui.Element
	}
	BorderStyle struct {
		None   func(*ui.Element) *ui.Element
		Hidden func(*ui.Element) *ui.Element
		Dotted func(*ui.Element) *ui.Element
		Dashed func(*ui.Element) *ui.Element
		Solid  func(*ui.Element) *ui.Element
		Double func(*ui.Element) *ui.Element
		Groove func(*ui.Element) *ui.Element
		Ridge  func(*ui.Element) *ui.Element
		Inset  func(*ui.Element) *ui.Element
		Outset func(*ui.Element) *ui.Element
		Value  func(value string) func(*ui.Element) *ui.Element
	}
	BorderTopRightRadius struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	CaptionSide struct {
		Top    func(*ui.Element) *ui.Element
		Bottom func(*ui.Element) *ui.Element
		Value  func(value string) func(*ui.Element) *ui.Element
	}
	FontFamily struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	TextDecorationColor struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	TransitionProperty struct {
		None  func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BackgroundOrigin struct {
		PaddingBox func(*ui.Element) *ui.Element
		BorderBox  func(*ui.Element) *ui.Element
		ContentBox func(*ui.Element) *ui.Element
		Value      func(value string) func(*ui.Element) *ui.Element
	}
	TextIndent struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Visibility struct {
		Visible  func(*ui.Element) *ui.Element
		Hidden   func(*ui.Element) *ui.Element
		Collapse func(*ui.Element) *ui.Element
		Value    func(value string) func(*ui.Element) *ui.Element
	}
	BorderColor struct {
		Transparent func(*ui.Element) *ui.Element
		Value       func(value string) func(*ui.Element) *ui.Element
	}
	BorderTop struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	FontVariant struct {
		Normal    func(*ui.Element) *ui.Element
		SmallCaps func(*ui.Element) *ui.Element
		Value     func(value string) func(*ui.Element) *ui.Element
	}
	Outline struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderBottomColor struct {
		Transparent func(*ui.Element) *ui.Element
		Value       func(value string) func(*ui.Element) *ui.Element
	}
	BorderTopStyle struct {
		None   func(*ui.Element) *ui.Element
		Hidden func(*ui.Element) *ui.Element
		Dotted func(*ui.Element) *ui.Element
		Dashed func(*ui.Element) *ui.Element
		Solid  func(*ui.Element) *ui.Element
		Double func(*ui.Element) *ui.Element
		Groove func(*ui.Element) *ui.Element
		Ridge  func(*ui.Element) *ui.Element
		Inset  func(*ui.Element) *ui.Element
		Outset func(*ui.Element) *ui.Element
		Value  func(value string) func(*ui.Element) *ui.Element
	}
	BorderWidth struct {
		Medium func(*ui.Element) *ui.Element
		Thin   func(*ui.Element) *ui.Element
		Thick  func(*ui.Element) *ui.Element
		Value  func(value string) func(*ui.Element) *ui.Element
	}
	ListStyleType struct {
		Disc               func(*ui.Element) *ui.Element
		Armenian           func(*ui.Element) *ui.Element
		Circle             func(*ui.Element) *ui.Element
		CjkIdeographic     func(*ui.Element) *ui.Element
		Decimal            func(*ui.Element) *ui.Element
		DecimalLeadingZero func(*ui.Element) *ui.Element
		Georgian           func(*ui.Element) *ui.Element
		Hebrew             func(*ui.Element) *ui.Element
		Hiragana           func(*ui.Element) *ui.Element
		HiraganaIroha      func(*ui.Element) *ui.Element
		Katakana           func(*ui.Element) *ui.Element
		KatakanaIroha      func(*ui.Element) *ui.Element
		LowerAlpha         func(*ui.Element) *ui.Element
		LowerGreek         func(*ui.Element) *ui.Element
		LowerLatin         func(*ui.Element) *ui.Element
		LowerRoman         func(*ui.Element) *ui.Element
		None               func(*ui.Element) *ui.Element
		Square             func(*ui.Element) *ui.Element
		UpperAlpha         func(*ui.Element) *ui.Element
		UpperGreek         func(*ui.Element) *ui.Element
		UpperLatin         func(*ui.Element) *ui.Element
		UpperRoman         func(*ui.Element) *ui.Element
		Value              func(value string) func(*ui.Element) *ui.Element
	}
	OutlineOffset struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Animation struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Background struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BackgroundRepeat struct {
		Repeat   func(*ui.Element) *ui.Element
		RepeatX  func(*ui.Element) *ui.Element
		RepeatY  func(*ui.Element) *ui.Element
		NoRepeat func(*ui.Element) *ui.Element
		Space    func(*ui.Element) *ui.Element
		Round    func(*ui.Element) *ui.Element
		Value    func(value string) func(*ui.Element) *ui.Element
	}
	BorderTopWidth struct {
		Medium func(*ui.Element) *ui.Element
		Thin   func(*ui.Element) *ui.Element
		Thick  func(*ui.Element) *ui.Element
		Value  func(value string) func(*ui.Element) *ui.Element
	}
	WordWrap struct {
		Normal    func(*ui.Element) *ui.Element
		BreakWord func(*ui.Element) *ui.Element
		Value     func(value string) func(*ui.Element) *ui.Element
	}
	BackgroundColor struct {
		Transparent func(*ui.Element) *ui.Element
		Value       func(value string) func(*ui.Element) *ui.Element
	}
	TextOverflow struct {
		Clip     func(*ui.Element) *ui.Element
		Ellipsis func(*ui.Element) *ui.Element
		Value    func(value string) func(*ui.Element) *ui.Element
	}
	TextShadow struct {
		None  func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BackgroundClip struct {
		BorderBox  func(*ui.Element) *ui.Element
		PaddingBox func(*ui.Element) *ui.Element
		ContentBox func(*ui.Element) *ui.Element
		Value      func(value string) func(*ui.Element) *ui.Element
	}
	BorderLeftWidth struct {
		Medium func(*ui.Element) *ui.Element
		Thin   func(*ui.Element) *ui.Element
		Thick  func(*ui.Element) *ui.Element
		Value  func(value string) func(*ui.Element) *ui.Element
	}
	Resize struct {
		None       func(*ui.Element) *ui.Element
		Both       func(*ui.Element) *ui.Element
		Horizontal func(*ui.Element) *ui.Element
		Vertical   func(*ui.Element) *ui.Element
		Value      func(value string) func(*ui.Element) *ui.Element
	}
	AnimationDirection struct {
		Normal           func(*ui.Element) *ui.Element
		Reverse          func(*ui.Element) *ui.Element
		Alternate        func(*ui.Element) *ui.Element
		AlternateReverse func(*ui.Element) *ui.Element
		Value            func(value string) func(*ui.Element) *ui.Element
	}
	Color struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	OutlineColor struct {
		Invert func(*ui.Element) *ui.Element
		Value  func(value string) func(*ui.Element) *ui.Element
	}
	BorderImageRepeat struct {
		Stretch func(*ui.Element) *ui.Element
		Repeat  func(*ui.Element) *ui.Element
		Round   func(*ui.Element) *ui.Element
		Space   func(*ui.Element) *ui.Element
		Value   func(value string) func(*ui.Element) *ui.Element
	}
	FontStretch struct {
		UltraCondensed func(*ui.Element) *ui.Element
		ExtraCondensed func(*ui.Element) *ui.Element
		Condensed      func(*ui.Element) *ui.Element
		SemiCondensed  func(*ui.Element) *ui.Element
		Normal         func(*ui.Element) *ui.Element
		SemiExpanded   func(*ui.Element) *ui.Element
		Expanded       func(*ui.Element) *ui.Element
		ExtraExpanded  func(*ui.Element) *ui.Element
		UltraExpanded  func(*ui.Element) *ui.Element
		Value          func(value string) func(*ui.Element) *ui.Element
	}
	TextTransform struct {
		None       func(*ui.Element) *ui.Element
		Capitalize func(*ui.Element) *ui.Element
		Uppercase  func(*ui.Element) *ui.Element
		Lowercase  func(*ui.Element) *ui.Element
		Value      func(value string) func(*ui.Element) *ui.Element
	}
	Margin struct {
		Top    func(value string) func(*ui.Element) *ui.Element
		Right  func(value string) func(*ui.Element) *ui.Element
		Bottom func(value string) func(*ui.Element) *ui.Element
		Left   func(value string) func(*ui.Element) *ui.Element
		Value  func(value string) func(*ui.Element) *ui.Element
	}
	Padding struct {
		Top    func(value string) func(*ui.Element) *ui.Element
		Right  func(value string) func(*ui.Element) *ui.Element
		Bottom func(value string) func(*ui.Element) *ui.Element
		Left   func(value string) func(*ui.Element) *ui.Element
		Value  func(value string) func(*ui.Element) *ui.Element
	}
	Transform struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	PointerEvents struct {
		Auto  func(*ui.Element) *ui.Element
		None  func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	UserSelect struct {
		None  func(*ui.Element) *ui.Element
		Auto  func(*ui.Element) *ui.Element
		Text  func(*ui.Element) *ui.Element
		All   func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BackdropFilter struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	ObjectFit struct {
		Contain   func(*ui.Element) *ui.Element
		Cover     func(*ui.Element) *ui.Element
		Fill      func(*ui.Element) *ui.Element
		None      func(*ui.Element) *ui.Element
		ScaleDown func(*ui.Element) *ui.Element
		Value     func(value string) func(*ui.Element) *ui.Element
	}
	ObjectPosition struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
}

type ContentLayout struct {
	FlexGrow struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	AlignSelf struct {
		Auto      func(*ui.Element) *ui.Element
		Stretch   func(*ui.Element) *ui.Element
		Center    func(*ui.Element) *ui.Element
		FlexStart func(*ui.Element) *ui.Element
		FlexEnd   func(*ui.Element) *ui.Element
		Baseline  func(*ui.Element) *ui.Element
		Value     func(value string) func(*ui.Element) *ui.Element
	}
	Content struct {
		Normal       func(*ui.Element) *ui.Element
		None         func(*ui.Element) *ui.Element
		Counter      func(*ui.Element) *ui.Element
		Attr         func(value string) func(*ui.Element) *ui.Element
		String       func(*ui.Element) *ui.Element
		OpenQuote    func(*ui.Element) *ui.Element
		CloseQuote   func(*ui.Element) *ui.Element
		NoOpenQuote  func(*ui.Element) *ui.Element
		NoCloseQuote func(*ui.Element) *ui.Element
		URL          func(url string) func(*ui.Element) *ui.Element
		Value        func(value string) func(*ui.Element) *ui.Element
	}
	ColumnSpan struct {
		None  func(*ui.Element) *ui.Element
		All   func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Flex struct {
		FlexGrow   func(value string) func(*ui.Element) *ui.Element
		FlexShrink func(value string) func(*ui.Element) *ui.Element
		FlexBasis  func(value string) func(*ui.Element) *ui.Element
		Auto       func(*ui.Element) *ui.Element
		Initial    func(*ui.Element) *ui.Element
		None       func(*ui.Element) *ui.Element
		Inherit    func(*ui.Element) *ui.Element
	}
	FlexShrink struct {
		Number func(value string) func(*ui.Element) *ui.Element
		Value  func(value string) func(*ui.Element) *ui.Element
	}
	Order struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	FlexBasis struct {
		Auto  func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	AlignItems struct {
		Stretch   func(*ui.Element) *ui.Element
		Center    func(*ui.Element) *ui.Element
		FlexStart func(*ui.Element) *ui.Element
		FlexEnd   func(*ui.Element) *ui.Element
		Baseline  func(*ui.Element) *ui.Element
		Value     func(value string) func(*ui.Element) *ui.Element
	}
}

type ContentStyle struct {
	ColumnRuleWidth struct {
		Medium func(*ui.Element) *ui.Element
		Thin   func(*ui.Element) *ui.Element
		Thick  func(*ui.Element) *ui.Element
		Value  func(value string) func(*ui.Element) *ui.Element
	}
	ColumnRule struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Direction struct {
		Ltr   func(*ui.Element) *ui.Element
		Rtl   func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	ColumnRuleStyle struct {
		None   func(*ui.Element) *ui.Element
		Hidden func(*ui.Element) *ui.Element
		Dotted func(*ui.Element) *ui.Element
		Dashed func(*ui.Element) *ui.Element
		Solid  func(*ui.Element) *ui.Element
		Double func(*ui.Element) *ui.Element
		Groove func(*ui.Element) *ui.Element
		Ridge  func(*ui.Element) *ui.Element
		Inset  func(*ui.Element) *ui.Element
		Outset func(*ui.Element) *ui.Element
		Value  func(value string) func(*ui.Element) *ui.Element
	}
	ColumnRuleColor struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	ColumnFill struct {
		Balance func(*ui.Element) *ui.Element
		Auto    func(*ui.Element) *ui.Element
		Value   func(value string) func(*ui.Element) *ui.Element
	}
	EmptyCells struct {
		Show  func(*ui.Element) *ui.Element
		Hide  func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Cursor struct {
		Alias        func(*ui.Element) *ui.Element
		AllScroll    func(*ui.Element) *ui.Element
		Auto         func(*ui.Element) *ui.Element
		Cell         func(*ui.Element) *ui.Element
		ContextMenu  func(*ui.Element) *ui.Element
		ColResize    func(*ui.Element) *ui.Element
		Copy         func(*ui.Element) *ui.Element
		Crosshair    func(*ui.Element) *ui.Element
		Default      func(*ui.Element) *ui.Element
		EResize      func(*ui.Element) *ui.Element
		EwResize     func(*ui.Element) *ui.Element
		Grab         func(*ui.Element) *ui.Element
		Grabbing     func(*ui.Element) *ui.Element
		Help         func(*ui.Element) *ui.Element
		Move         func(*ui.Element) *ui.Element
		NResize      func(*ui.Element) *ui.Element
		NeResize     func(*ui.Element) *ui.Element
		NeswResize   func(*ui.Element) *ui.Element
		NsResize     func(*ui.Element) *ui.Element
		NwResize     func(*ui.Element) *ui.Element
		NwseResize   func(*ui.Element) *ui.Element
		NoDrop       func(*ui.Element) *ui.Element
		None         func(*ui.Element) *ui.Element
		NotAllowed   func(*ui.Element) *ui.Element
		Pointer      func(*ui.Element) *ui.Element
		Progress     func(*ui.Element) *ui.Element
		RowResize    func(*ui.Element) *ui.Element
		SResize      func(*ui.Element) *ui.Element
		SeResize     func(*ui.Element) *ui.Element
		SwResize     func(*ui.Element) *ui.Element
		Text         func(*ui.Element) *ui.Element
		VerticalText func(*ui.Element) *ui.Element
		WResize      func(*ui.Element) *ui.Element
		Wait         func(*ui.Element) *ui.Element
		ZoomIn       func(*ui.Element) *ui.Element
		ZoomOut      func(*ui.Element) *ui.Element
		Value        func(value string) func(*ui.Element) *ui.Element
	}
}

func (c ContainerStyle) CustomSetter(property string) func(value string) func(*ui.Element) *ui.Element {
	return func(value string) func(*ui.Element) *ui.Element {
		return func(e *ui.Element) *ui.Element {
			return css("", property, value)(e)
		}
	}
}

func (c ContainerLayout) CustomSetter(property string) func(value string) func(*ui.Element) *ui.Element {
	return func(value string) func(*ui.Element) *ui.Element {
		return func(e *ui.Element) *ui.Element {
			return css("", property, value)(e)
		}
	}
}

func (c ContentStyle) CustomSetter(property string) func(value string) func(*ui.Element) *ui.Element {
	return func(value string) func(*ui.Element) *ui.Element {
		return func(e *ui.Element) *ui.Element {
			return css("", property, value)(e)
		}
	}
}

func (c ContentLayout) CustomSetter(property string) func(value string) func(*ui.Element) *ui.Element {
	return func(value string) func(*ui.Element) *ui.Element {
		return func(e *ui.Element) *ui.Element {
			return css("", property, value)(e)
		}
	}
}
