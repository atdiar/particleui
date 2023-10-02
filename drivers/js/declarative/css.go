package functions

import(
	"github.com/atdiar/particleui"
)

func css(modifier string, property,value string) func(*ui.Element) *ui.Element{
	return func(e *ui.Element) *ui.Element{
		// Retrieve/Create a css ruleset object and store the new rule/declaration.
		var ruleset *ui.TempObject
		c,ok:= e.Get("css","ruleset")
		if !ok{
			ruleset = ui.NewObject()
		}
		ruleset = c.(ui.Object).MakeCopy()
		ruleobj,ok:= ruleset.Get(modifier)
		var rules *ui.TempObject
		if !ok{
			rules = ui.NewObject()
			rules.Set(property,ui.String(value))
			ruleset.Set(modifier,rules.Commit())
		} else{
			rules = ruleobj.(ui.Object).MakeCopy()
			rules.Set(property,ui.String(value))
			ruleset.Set(modifier,rules.Commit())
		}
		e.Set("css","ruleset",ruleset.Commit())
		return e
	}
}

type valueFn[T any] struct{
	VFn func(property, pseudoclass string) (func(*ui.Element) *ui.Element)
	CFn func(property, pseudoclass string) (func(value string) func(*ui.Element) *ui.Element)
	Typ T
}

func (v valueFn[T]) Setter(property string) (func(*ui.Element) *ui.Element){
	var pseudoclass string
	switch any(v.Typ).(type){
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

func (v valueFn[T]) CustomSetter(property string) (func(value string) func(*ui.Element) *ui.Element){
	var pseudoclass string
	switch any(v.Typ).(type){
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

func NewValueFn[T any](fn func(property, pseudoclass string) (func(*ui.Element) *ui.Element), cfn func(property, pseudoclass string) (func(value string) func(*ui.Element) *ui.Element)) valueFn[T]{
	var v T
	return valueFn[T]{fn, cfn, v}
}

func cfn(property, pseudoclass string) (func(val string) func(*ui.Element)*ui.Element){
	return func(val string) func(e *ui.Element) *ui.Element {
		return css(pseudoclass, property, val)
	}
}

func vfn(value string) func(property string, pseudoclass string) func(*ui.Element)*ui.Element{
	return func(property, pseudoclass string) func(*ui.Element) *ui.Element {
		return cfn(property,pseudoclass)(value)
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

	Hover *Container
	Active *Container
	Focus *Container
	Visited *Container
	Link *Container
	FirstChild *Container
	LastChild *Container
	Checked *Container
	Disabled *Container
	Enabled *Container
}

type Content struct {
	Style  ContentStyle
	Layout ContentLayout

	Hover *Content
	Active *Content
	Focus *Content
	Visited *Content
	Link *Content
	FirstChild *Content
	LastChild *Content
	Checked *Content
	Disabled *Content
	Enabled *Content
}

func initializeContainer[pseudoclass any]() *Container{
	c := Container{}
	c.Style = initializeContainerStyle[pseudoclass]()
	c.Layout = initializeContainerLayout[pseudoclass]()
	return &c
}

func initializeContent[pseudoclass any]() *Content{
	c := Content{}
	c.Style = initializeContentStyle[pseudoclass]()
	c.Layout = initializeContentLayout[pseudoclass]()
	return &c
}

func initializeContainerLayout[pseudoclass any]() ContainerLayout{
	// Setting the proper function for each field
	c := ContainerLayout{}
	c.BoxShadow.None = NewValueFn[pseudoclass](vfn("None"),nil).Setter("box-shadow")
	c.BoxShadow.Value = NewValueFn[pseudoclass](nil,cfn).CustomSetter("box-shadow")

	c.JustifyContent.FlexStart = NewValueFn[pseudoclass](vfn("flex-start"),nil).Setter("justify-content")
	c.JustifyContent.FlexEnd = NewValueFn[pseudoclass](vfn("flex-end"),nil).Setter("justify-content")
	c.JustifyContent.Center = NewValueFn[pseudoclass](vfn("center"),nil).Setter("justify-content")
	c.JustifyContent.SpaceBetween = NewValueFn[pseudoclass](vfn("space-between"),nil).Setter("justify-content")
	c.JustifyContent.SpaceAround = NewValueFn[pseudoclass](vfn("space-around"),nil).Setter("justify-content")
	c.JustifyContent.Value = NewValueFn[pseudoclass](nil,cfn).CustomSetter("justify-content")

	// ZIndex
    c.ZIndex.Auto = NewValueFn[pseudoclass](vfn("auto"), nil).Setter("z-index")
    c.ZIndex.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("z-index")

    // Float
    c.Float.None = NewValueFn[pseudoclass](vfn("none"), nil).Setter("float")
    c.Float.Left = NewValueFn[pseudoclass](vfn("left"), nil).Setter("float")
    c.Float.Right = NewValueFn[pseudoclass](vfn("right"), nil).Setter("float")
    c.Float.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("float")

    // Overflow
    c.Overflow.Visible = NewValueFn[pseudoclass](vfn("visible"), nil).Setter("overflow")
    c.Overflow.Hidden = NewValueFn[pseudoclass](vfn("hidden"), nil).Setter("overflow")
    c.Overflow.Scroll = NewValueFn[pseudoclass](vfn("scroll"), nil).Setter("overflow")
    c.Overflow.Auto = NewValueFn[pseudoclass](vfn("auto"), nil).Setter("overflow")
    c.Overflow.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("overflow")

    // OverflowY
    c.OverflowY.Visible = NewValueFn[pseudoclass](vfn("visible"), nil).Setter("overflow-y")
    c.OverflowY.Hidden = NewValueFn[pseudoclass](vfn("hidden"), nil).Setter("overflow-y")
    c.OverflowY.Scroll = NewValueFn[pseudoclass](vfn("scroll"), nil).Setter("overflow-y")
    c.OverflowY.Auto = NewValueFn[pseudoclass](vfn("auto"), nil).Setter("overflow-y")
    c.OverflowY.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("overflow-y")

    // Perspective
    c.Perspective.None = NewValueFn[pseudoclass](vfn("none"), nil).Setter("perspective")
    c.Perspective.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("perspective")

	// BorderCollapse
    c.BorderCollapse.Separate = NewValueFn[pseudoclass](vfn("separate"), nil).Setter("border-collapse")
    c.BorderCollapse.Collapse = NewValueFn[pseudoclass](vfn("collapse"), nil).Setter("border-collapse")
    c.BorderCollapse.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-collapse")

    // PageBreakBefore
    c.PageBreakBefore.Auto = NewValueFn[pseudoclass](vfn("auto"), nil).Setter("page-break-before")
    c.PageBreakBefore.Always = NewValueFn[pseudoclass](vfn("always"), nil).Setter("page-break-before")
    c.PageBreakBefore.Avoid = NewValueFn[pseudoclass](vfn("avoid"), nil).Setter("page-break-before")
    c.PageBreakBefore.Left = NewValueFn[pseudoclass](vfn("left"), nil).Setter("page-break-before")
    c.PageBreakBefore.Right = NewValueFn[pseudoclass](vfn("right"), nil).Setter("page-break-before")
    c.PageBreakBefore.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("page-break-before")

    // Columns
    c.Columns.Auto = NewValueFn[pseudoclass](vfn("auto"), nil).Setter("columns")
    c.Columns.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("columns")

    // ColumnCount
    c.ColumnCount.Auto = NewValueFn[pseudoclass](vfn("auto"), nil).Setter("column-count")
    c.ColumnCount.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("column-count")

    // MinHeight
    c.MinHeight.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("min-height")

    // PageBreakInside
    c.PageBreakInside.Auto = NewValueFn[pseudoclass](vfn("auto"), nil).Setter("page-break-inside")
    c.PageBreakInside.Avoid = NewValueFn[pseudoclass](vfn("avoid"), nil).Setter("page-break-inside")
    c.PageBreakInside.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("page-break-inside")

    // ColumnGap
    c.ColumnGap.Length = NewValueFn[pseudoclass](nil, cfn).CustomSetter("column-gap") 
    c.ColumnGap.Normal = NewValueFn[pseudoclass](vfn("normal"), nil).Setter("column-gap")
    c.ColumnGap.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("column-gap")

    // Clip
    c.Clip.Auto = NewValueFn[pseudoclass](vfn("auto"), nil).Setter("clip")
    c.Clip.Shape = NewValueFn[pseudoclass](vfn("shape"), nil).Setter("clip")
    c.Clip.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("clip")

	// FlexDirection
	c.FlexDirection.Row = NewValueFn[pseudoclass](vfn("row"), nil).Setter("flex-direction")
	c.FlexDirection.RowReverse = NewValueFn[pseudoclass](vfn("row-reverse"), nil).Setter("flex-direction")
	c.FlexDirection.Column = NewValueFn[pseudoclass](vfn("column"), nil).Setter("flex-direction")
	c.FlexDirection.ColumnReverse = NewValueFn[pseudoclass](vfn("column-reverse"), nil).Setter("flex-direction")
	c.FlexDirection.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("flex-direction")

	// PageBreakAfter
	c.PageBreakAfter.Auto = NewValueFn[pseudoclass](vfn("auto"), nil).Setter("page-break-after")
	c.PageBreakAfter.Always = NewValueFn[pseudoclass](vfn("always"), nil).Setter("page-break-after")
	c.PageBreakAfter.Avoid = NewValueFn[pseudoclass](vfn("avoid"), nil).Setter("page-break-after")
	c.PageBreakAfter.Left = NewValueFn[pseudoclass](vfn("left"), nil).Setter("page-break-after")
	c.PageBreakAfter.Right = NewValueFn[pseudoclass](vfn("right"), nil).Setter("page-break-after")
	c.PageBreakAfter.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("page-break-after")

	// Top
	c.Top.Auto = NewValueFn[pseudoclass](vfn("auto"), nil).Setter("top")
	c.Top.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("top")

	// CounterIncrement
	c.CounterIncrement.None = NewValueFn[pseudoclass](vfn("none"), nil).Setter("counter-increment")
	c.CounterIncrement.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("counter-increment")

	// Height
	c.Height.Auto = NewValueFn[pseudoclass](vfn("auto"), nil).Setter("height")
	c.Height.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("height")

	// TransformStyle
	c.TransformStyle.Flat = NewValueFn[pseudoclass](vfn("flat"), nil).Setter("transform-style")
	c.TransformStyle.Preserve3d = NewValueFn[pseudoclass](vfn("preserve-3d"), nil).Setter("transform-style")
	c.TransformStyle.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("transform-style")

	// OverflowX
	c.OverflowX.Visible = NewValueFn[pseudoclass](vfn("visible"), nil).Setter("overflow-x")
	c.OverflowX.Hidden = NewValueFn[pseudoclass](vfn("hidden"), nil).Setter("overflow-x")
	c.OverflowX.Scroll = NewValueFn[pseudoclass](vfn("scroll"), nil).Setter("overflow-x")
	c.OverflowX.Auto = NewValueFn[pseudoclass](vfn("auto"), nil).Setter("overflow-x")
	c.OverflowX.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("overflow-x")

	// FlexWrap
	c.FlexWrap.Nowrap = NewValueFn[pseudoclass](vfn("nowrap"), nil).Setter("flex-wrap")
	c.FlexWrap.Wrap = NewValueFn[pseudoclass](vfn("wrap"), nil).Setter("flex-wrap")
	c.FlexWrap.WrapReverse = NewValueFn[pseudoclass](vfn("wrap-reverse"), nil).Setter("flex-wrap")
	c.FlexWrap.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("flex-wrap")

	// MaxWidth
	c.MaxWidth.None = NewValueFn[pseudoclass](vfn("none"), nil).Setter("max-width")
	c.MaxWidth.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("max-width")

	// Bottom
	c.Bottom.Auto = NewValueFn[pseudoclass](vfn("auto"), nil).Setter("bottom")
	c.Bottom.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("bottom")

	// CounterReset
	c.CounterReset.None = NewValueFn[pseudoclass](vfn("none"), nil).Setter("counter-reset")
	c.CounterReset.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("counter-reset")

	// Right
	c.Right.Auto = NewValueFn[pseudoclass](vfn("auto"), nil).Setter("right")
	c.Right.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("right")

	// BoxSizing
	c.BoxSizing.ContentBox = NewValueFn[pseudoclass](vfn("content-box"), nil).Setter("box-sizing")
	c.BoxSizing.BorderBox = NewValueFn[pseudoclass](vfn("border-box"), nil).Setter("box-sizing")
	c.BoxSizing.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("box-sizing")

	// Position
	c.Position.Static = NewValueFn[pseudoclass](vfn("static"), nil).Setter("position")
	c.Position.Absolute = NewValueFn[pseudoclass](vfn("absolute"), nil).Setter("position")
	c.Position.Fixed = NewValueFn[pseudoclass](vfn("fixed"), nil).Setter("position")
	c.Position.Relative = NewValueFn[pseudoclass](vfn("relative"), nil).Setter("position")
	c.Position.Sticky = NewValueFn[pseudoclass](vfn("sticky"), nil).Setter("position")
	c.Position.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("position")

	// TableLayout
	c.TableLayout.Auto = NewValueFn[pseudoclass](vfn("auto"), nil).Setter("table-layout")
	c.TableLayout.Fixed = NewValueFn[pseudoclass](vfn("fixed"), nil).Setter("table-layout")
	c.TableLayout.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("table-layout")

	// Width
	c.Width.Auto = NewValueFn[pseudoclass](vfn("auto"), nil).Setter("width")
	c.Width.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("width")

	// MaxHeight
	c.MaxHeight.None = NewValueFn[pseudoclass](vfn("none"), nil).Setter("max-height")
	c.MaxHeight.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("max-height")

	// ColumnWidth
	c.ColumnWidth.Auto = NewValueFn[pseudoclass](vfn("auto"), nil).Setter("column-width")
	c.ColumnWidth.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("column-width")

	// MinWidth
	c.MinWidth.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("min-width")

	// VerticalAlign
	c.VerticalAlign.Baseline = NewValueFn[pseudoclass](vfn("baseline"), nil).Setter("vertical-align")
	c.VerticalAlign.Top = NewValueFn[pseudoclass](vfn("top"), nil).Setter("vertical-align")
	c.VerticalAlign.TextTop = NewValueFn[pseudoclass](vfn("text-top"), nil).Setter("vertical-align")
	c.VerticalAlign.Middle = NewValueFn[pseudoclass](vfn("middle"), nil).Setter("vertical-align")
	c.VerticalAlign.Bottom = NewValueFn[pseudoclass](vfn("bottom"), nil).Setter("vertical-align")
	c.VerticalAlign.TextBottom = NewValueFn[pseudoclass](vfn("text-bottom"), nil).Setter("vertical-align")
	c.VerticalAlign.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("vertical-align")

	// PerspectiveOrigin
	c.PerspectiveOrigin.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("perspective-origin")

	// AlignContent
	c.AlignContent.Stretch = NewValueFn[pseudoclass](vfn("stretch"), nil).Setter("align-content")
	c.AlignContent.Center = NewValueFn[pseudoclass](vfn("center"), nil).Setter("align-content")
	c.AlignContent.FlexStart = NewValueFn[pseudoclass](vfn("flex-start"), nil).Setter("align-content")
	c.AlignContent.FlexEnd = NewValueFn[pseudoclass](vfn("flex-end"), nil).Setter("align-content")
	c.AlignContent.SpaceBetween = NewValueFn[pseudoclass](vfn("space-between"), nil).Setter("align-content")
	c.AlignContent.SpaceAround = NewValueFn[pseudoclass](vfn("space-around"), nil).Setter("align-content")
	c.AlignContent.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("align-content")

	// FlexFlow
	c.FlexFlow.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("flex-flow")

	// Display
	c.Display.Inline = NewValueFn[pseudoclass](vfn("inline"), nil).Setter("display")
	c.Display.Block = NewValueFn[pseudoclass](vfn("block"), nil).Setter("display")
	c.Display.Contents = NewValueFn[pseudoclass](vfn("contents"), nil).Setter("display")
	c.Display.Flex = NewValueFn[pseudoclass](vfn("flex"), nil).Setter("display")
	c.Display.Grid = NewValueFn[pseudoclass](vfn("grid"), nil).Setter("display")
	c.Display.None = NewValueFn[pseudoclass](vfn("none"), nil).Setter("display")
	c.Display.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("display")

	// Left
	c.Left.Auto = NewValueFn[pseudoclass](vfn("auto"), nil).Setter("left")
	c.Left.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("left")

	return c

}

func initializeContainerStyle[pseudoclass any]() ContainerStyle{
	c:= ContainerStyle{}

	c.BackgroundImage.URL = NewValueFn[pseudoclass](nil, cfn).CustomSetter("background-image")
	c.BackgroundImage.None = NewValueFn[pseudoclass](vfn("none"), nil).Setter("background-image")
	c.BackgroundImage.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("background-image")

	// BorderLeftStyle
	c.BorderLeftStyle.None = NewValueFn[pseudoclass](vfn("none"), nil).Setter("border-left-style")
	c.BorderLeftStyle.Hidden = NewValueFn[pseudoclass](vfn("hidden"), nil).Setter("border-left-style")
	c.BorderLeftStyle.Dotted = NewValueFn[pseudoclass](vfn("dotted"), nil).Setter("border-left-style")
	c.BorderLeftStyle.Dashed = NewValueFn[pseudoclass](vfn("dashed"), nil).Setter("border-left-style")
	c.BorderLeftStyle.Solid = NewValueFn[pseudoclass](vfn("solid"), nil).Setter("border-left-style")
	c.BorderLeftStyle.Double = NewValueFn[pseudoclass](vfn("double"), nil).Setter("border-left-style")
	c.BorderLeftStyle.Groove = NewValueFn[pseudoclass](vfn("groove"), nil).Setter("border-left-style")
	c.BorderLeftStyle.Ridge = NewValueFn[pseudoclass](vfn("ridge"), nil).Setter("border-left-style")
	c.BorderLeftStyle.Inset = NewValueFn[pseudoclass](vfn("inset"), nil).Setter("border-left-style")
	c.BorderLeftStyle.Outset = NewValueFn[pseudoclass](vfn("outset"), nil).Setter("border-left-style")
	c.BorderLeftStyle.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-left-style")

	// BoxShadow
	c.BoxShadow.None = NewValueFn[pseudoclass](vfn("none"), nil).Setter("box-shadow")
	c.BoxShadow.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("box-shadow")

	// TransitionDelay
	c.TransitionDelay.Time = NewValueFn[pseudoclass](nil, cfn).CustomSetter("transition-delay")
	c.TransitionDelay.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("transition-delay")

	// AnimationDuration
	c.AnimationDuration.Time = NewValueFn[pseudoclass](nil, cfn).CustomSetter("animation-duration")
	c.AnimationDuration.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("animation-duration")

	// ListStyle
	c.ListStyle.ListStyleType = NewValueFn[pseudoclass](nil, cfn).CustomSetter("list-style-type")
	c.ListStyle.ListStylePosition = NewValueFn[pseudoclass](nil, cfn).CustomSetter("list-style-position")
	c.ListStyle.ListStyleImage = NewValueFn[pseudoclass](nil, cfn).CustomSetter("list-style-image")
	c.ListStyle.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("list-style")

	// OutlineWidth
	c.OutlineWidth.Medium = NewValueFn[pseudoclass](vfn("medium"), nil).Setter("outline-width")
	c.OutlineWidth.Thin = NewValueFn[pseudoclass](vfn("thin"), nil).Setter("outline-width")
	c.OutlineWidth.Thick = NewValueFn[pseudoclass](vfn("thick"), nil).Setter("outline-width")
	c.OutlineWidth.Length = NewValueFn[pseudoclass](nil, cfn).CustomSetter("outline-width")
	c.OutlineWidth.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("outline-width")
	
	// BorderTopLeftRadius
	c.BorderTopLeftRadius.Length = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-top-left-radius")
	c.BorderTopLeftRadius.Percent = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-top-left-radius")
	c.BorderTopLeftRadius.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-top-left-radius")

	// WhiteSpace
	c.WhiteSpace.Normal = NewValueFn[pseudoclass](vfn("normal"), nil).Setter("white-space")
	c.WhiteSpace.Nowrap = NewValueFn[pseudoclass](vfn("nowrap"), nil).Setter("white-space")
	c.WhiteSpace.Pre = NewValueFn[pseudoclass](vfn("pre"), nil).Setter("white-space")
	c.WhiteSpace.PreLine = NewValueFn[pseudoclass](vfn("pre-line"), nil).Setter("white-space")
	c.WhiteSpace.PreWrap = NewValueFn[pseudoclass](vfn("pre-wrap"), nil).Setter("white-space")
	c.WhiteSpace.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("white-space")

	// BorderRight
	c.BorderRight.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-right")

	// TextDecorationLine
	c.TextDecorationLine.None = NewValueFn[pseudoclass](vfn("none"), nil).Setter("text-decoration-line")
	c.TextDecorationLine.Underline = NewValueFn[pseudoclass](vfn("underline"), nil).Setter("text-decoration-line")
	c.TextDecorationLine.Overline = NewValueFn[pseudoclass](vfn("overline"), nil).Setter("text-decoration-line")
	c.TextDecorationLine.LineThrough = NewValueFn[pseudoclass](vfn("line-through"), nil).Setter("text-decoration-line")
	c.TextDecorationLine.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("text-decoration-line")

	// AnimationDelay
	c.AnimationDelay.Time = NewValueFn[pseudoclass](nil, cfn).CustomSetter("animation-delay")
	c.AnimationDelay.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("animation-delay")

	// BackgroundPosition
	c.BackgroundPosition.LeftTop = NewValueFn[pseudoclass](vfn("left top"), nil).Setter("background-position")
	c.BackgroundPosition.LeftCenter = NewValueFn[pseudoclass](vfn("left center"), nil).Setter("background-position")
	c.BackgroundPosition.LeftBottom = NewValueFn[pseudoclass](vfn("left bottom"), nil).Setter("background-position")
	c.BackgroundPosition.RightTop = NewValueFn[pseudoclass](vfn("right top"), nil).Setter("background-position")
	c.BackgroundPosition.RightCenter = NewValueFn[pseudoclass](vfn("right center"), nil).Setter("background-position")
	c.BackgroundPosition.RightBottom = NewValueFn[pseudoclass](vfn("right bottom"), nil).Setter("background-position")
	c.BackgroundPosition.CenterTop = NewValueFn[pseudoclass](vfn("center top"), nil).Setter("background-position")
	c.BackgroundPosition.CenterCenter = NewValueFn[pseudoclass](vfn("center center"), nil).Setter("background-position")
	c.BackgroundPosition.CenterBottom = NewValueFn[pseudoclass](vfn("center bottom"), nil).Setter("background-position")
	c.BackgroundPosition.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("background-position")

	// BorderImage
	c.BorderImage.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-image")

	// BorderSpacing
	c.BorderSpacing.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-spacing")

	// BorderImageOutset
	c.BorderImageOutset.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-image-outset")

	// BorderImageSlice
	c.BorderImageSlice.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-image-slice")

	// BorderLeftColor
	c.BorderLeftColor.Color = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-left-color")
	c.BorderLeftColor.Transparent = NewValueFn[pseudoclass](vfn("transparent"), nil).Setter("border-left-color")
	c.BorderLeftColor.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-left-color")

	// FontSize
	c.FontSize.Medium = NewValueFn[pseudoclass](vfn("medium"), nil).Setter("font-size")
	c.FontSize.XxSmall = NewValueFn[pseudoclass](vfn("xx-small"), nil).Setter("font-size")
	c.FontSize.XSmall = NewValueFn[pseudoclass](vfn("x-small"), nil).Setter("font-size")
	c.FontSize.Small = NewValueFn[pseudoclass](vfn("small"), nil).Setter("font-size")
	c.FontSize.Large = NewValueFn[pseudoclass](vfn("large"), nil).Setter("font-size")
	c.FontSize.XLarge = NewValueFn[pseudoclass](vfn("x-large"), nil).Setter("font-size")
	c.FontSize.XxLarge = NewValueFn[pseudoclass](vfn("xx-large"), nil).Setter("font-size")
	c.FontSize.Smaller = NewValueFn[pseudoclass](vfn("smaller"), nil).Setter("font-size")
	c.FontSize.Larger = NewValueFn[pseudoclass](vfn("larger"), nil).Setter("font-size")
	c.FontSize.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("font-size")

	// LineHeight
	c.LineHeight.Normal = NewValueFn[pseudoclass](vfn("normal"), nil).Setter("line-height")
	c.LineHeight.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("line-height")

	// TextDecorationStyle
	c.TextDecorationStyle.Solid = NewValueFn[pseudoclass](vfn("solid"), nil).Setter("text-decoration-style")
	c.TextDecorationStyle.Double = NewValueFn[pseudoclass](vfn("double"), nil).Setter("text-decoration-style")
	c.TextDecorationStyle.Dotted = NewValueFn[pseudoclass](vfn("dotted"), nil).Setter("text-decoration-style")
	c.TextDecorationStyle.Dashed = NewValueFn[pseudoclass](vfn("dashed"), nil).Setter("text-decoration-style")
	c.TextDecorationStyle.Wavy = NewValueFn[pseudoclass](vfn("wavy"), nil).Setter("text-decoration-style")
	c.TextDecorationStyle.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("text-decoration-style")

	// BackfaceVisibility
	c.BackfaceVisibility.Visible = NewValueFn[pseudoclass](vfn("visible"), nil).Setter("backface-visibility")
	c.BackfaceVisibility.Hidden = NewValueFn[pseudoclass](vfn("hidden"), nil).Setter("backface-visibility")
	c.BackfaceVisibility.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("backface-visibility")

	// BorderRightStyle
	c.BorderRightStyle.None = NewValueFn[pseudoclass](vfn("none"), nil).Setter("border-right-style")
	c.BorderRightStyle.Hidden = NewValueFn[pseudoclass](vfn("hidden"), nil).Setter("border-right-style")
	c.BorderRightStyle.Dotted = NewValueFn[pseudoclass](vfn("dotted"), nil).Setter("border-right-style")
	c.BorderRightStyle.Dashed = NewValueFn[pseudoclass](vfn("dashed"), nil).Setter("border-right-style")
	c.BorderRightStyle.Solid = NewValueFn[pseudoclass](vfn("solid"), nil).Setter("border-right-style")
	c.BorderRightStyle.Double = NewValueFn[pseudoclass](vfn("double"), nil).Setter("border-right-style")
	c.BorderRightStyle.Groove = NewValueFn[pseudoclass](vfn("groove"), nil).Setter("border-right-style")
	c.BorderRightStyle.Ridge = NewValueFn[pseudoclass](vfn("ridge"), nil).Setter("border-right-style")
	c.BorderRightStyle.Inset = NewValueFn[pseudoclass](vfn("inset"), nil).Setter("border-right-style")
	c.BorderRightStyle.Outset = NewValueFn[pseudoclass](vfn("outset"), nil).Setter("border-right-style")
	c.BorderRightStyle.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-right-style")

	// TextDecoration
	c.TextDecoration.TextDecorationLine = NewValueFn[pseudoclass](nil, cfn).CustomSetter("text-decoration-line")
	c.TextDecoration.TextDecorationColor = NewValueFn[pseudoclass](nil, cfn).CustomSetter("text-decoration-color")
	c.TextDecoration.TextDecorationStyle = NewValueFn[pseudoclass](nil, cfn).CustomSetter("text-decoration-style")
	c.TextDecoration.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("text-decoration")

	// Transition
	c.Transition.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("transition")

	// AnimationIterationCount
	c.AnimationIterationCount.Infinite = NewValueFn[pseudoclass](vfn("infinite"), nil).Setter("animation-iteration-count")
	c.AnimationIterationCount.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("animation-iteration-count")

	// BorderBottom
	c.BorderBottom.BorderBottomWidth = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-bottom-width")
	c.BorderBottom.BorderBottomStyle = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-bottom-style")
	c.BorderBottom.BorderBottomColor = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-bottom-color")
	c.BorderBottom.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-bottom")

	// AnimationTimingFunction
	c.AnimationTimingFunction.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("animation-timing-function")

	// BorderRadius
	c.BorderRadius.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-radius")

	// Quotes
	c.Quotes.None = NewValueFn[pseudoclass](vfn("none"), nil).Setter("quotes")
	c.Quotes.Auto = NewValueFn[pseudoclass](vfn("auto"), nil).Setter("quotes")
	c.Quotes.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("quotes")

	// TabSize
	c.TabSize.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("tab-size")

	// AnimationFillMode
	c.AnimationFillMode.None = NewValueFn[pseudoclass](vfn("none"), nil).Setter("animation-fill-mode")
	c.AnimationFillMode.Forwards = NewValueFn[pseudoclass](vfn("forwards"), nil).Setter("animation-fill-mode")
	c.AnimationFillMode.Backwards = NewValueFn[pseudoclass](vfn("backwards"), nil).Setter("animation-fill-mode")
	c.AnimationFillMode.Both = NewValueFn[pseudoclass](vfn("both"), nil).Setter("animation-fill-mode")
	c.AnimationFillMode.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("animation-fill-mode")

	// BackgroundSize
	c.BackgroundSize.Auto = NewValueFn[pseudoclass](vfn("auto"), nil).Setter("background-size")
	c.BackgroundSize.Cover = NewValueFn[pseudoclass](vfn("cover"), nil).Setter("background-size")
	c.BackgroundSize.Contain = NewValueFn[pseudoclass](vfn("contain"), nil).Setter("background-size")
	c.BackgroundSize.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("background-size")

	// FontSizeAdjust
	c.FontSizeAdjust.None = NewValueFn[pseudoclass](vfn("none"), nil).Setter("font-size-adjust")
	c.FontSizeAdjust.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("font-size-adjust")

	// ListStylePosition
	c.ListStylePosition.Inside = NewValueFn[pseudoclass](vfn("inside"), nil).Setter("list-style-position")
	c.ListStylePosition.Outside = NewValueFn[pseudoclass](vfn("outside"), nil).Setter("list-style-position")
	c.ListStylePosition.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("list-style-position")

	// TextAlign
	c.TextAlign.Left = NewValueFn[pseudoclass](vfn("left"), nil).Setter("text-align")
	c.TextAlign.Right = NewValueFn[pseudoclass](vfn("right"), nil).Setter("text-align")
	c.TextAlign.Center = NewValueFn[pseudoclass](vfn("center"), nil).Setter("text-align")
	c.TextAlign.Justify = NewValueFn[pseudoclass](vfn("justify"), nil).Setter("text-align")
	c.TextAlign.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("text-align")

	// TextJustify
	c.TextJustify.Auto = NewValueFn[pseudoclass](vfn("auto"), nil).Setter("text-justify")
	c.TextJustify.InterWord = NewValueFn[pseudoclass](vfn("inter-word"), nil).Setter("text-justify")
	c.TextJustify.InterCharacter = NewValueFn[pseudoclass](vfn("inter-character"), nil).Setter("text-justify")
	c.TextJustify.None = NewValueFn[pseudoclass](vfn("none"), nil).Setter("text-justify")
	c.TextJustify.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("text-justify")

	// BackgroundAttachment
	c.BackgroundAttachment.Scroll = NewValueFn[pseudoclass](vfn("scroll"), nil).Setter("background-attachment")
	c.BackgroundAttachment.Fixed = NewValueFn[pseudoclass](vfn("fixed"), nil).Setter("background-attachment")
	c.BackgroundAttachment.Local = NewValueFn[pseudoclass](vfn("local"), nil).Setter("background-attachment")
	c.BackgroundAttachment.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("background-attachment")

	// BorderRightWidth
	c.BorderRightWidth.Medium = NewValueFn[pseudoclass](vfn("medium"), nil).Setter("border-right-width")
	c.BorderRightWidth.Thin = NewValueFn[pseudoclass](vfn("thin"), nil).Setter("border-right-width")
	c.BorderRightWidth.Thick = NewValueFn[pseudoclass](vfn("thick"), nil).Setter("border-right-width")
	c.BorderRightWidth.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-right-width")

	// Font
	c.Font.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("font")

	// BorderLeft
	c.BorderLeft.BorderLeftWidth = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-left-width")
	c.BorderLeft.BorderLeftStyle = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-left-style")
	c.BorderLeft.BorderLeftColor = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-left-color")
	c.BorderLeft.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-left")

	// TransitionDuration
	c.TransitionDuration.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("transition-duration")

	// WordSpacing
	c.WordSpacing.Normal = NewValueFn[pseudoclass](vfn("normal"), nil).Setter("word-spacing")
	c.WordSpacing.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("word-spacing")

	// AnimationName
	c.AnimationName.None = NewValueFn[pseudoclass](vfn("none"), nil).Setter("animation-name")
	c.AnimationName.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("animation-name")

	// AnimationPlayState
	c.AnimationPlayState.Paused = NewValueFn[pseudoclass](vfn("paused"), nil).Setter("animation-play-state")
	c.AnimationPlayState.Running = NewValueFn[pseudoclass](vfn("running"), nil).Setter("animation-play-state")
	c.AnimationPlayState.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("animation-play-state")

	// LetterSpacing
	c.LetterSpacing.Normal = NewValueFn[pseudoclass](vfn("normal"), nil).Setter("letter-spacing")
	c.LetterSpacing.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("letter-spacing")

	// BorderBottomStyle
	c.BorderBottomStyle.None = NewValueFn[pseudoclass](vfn("none"), nil).Setter("border-bottom-style")
	c.BorderBottomStyle.Hidden = NewValueFn[pseudoclass](vfn("hidden"), nil).Setter("border-bottom-style")
	c.BorderBottomStyle.Dotted = NewValueFn[pseudoclass](vfn("dotted"), nil).Setter("border-bottom-style")
	c.BorderBottomStyle.Dashed = NewValueFn[pseudoclass](vfn("dashed"), nil).Setter("border-bottom-style")
	c.BorderBottomStyle.Solid = NewValueFn[pseudoclass](vfn("solid"), nil).Setter("border-bottom-style")
	c.BorderBottomStyle.Double = NewValueFn[pseudoclass](vfn("double"), nil).Setter("border-bottom-style")
	c.BorderBottomStyle.Groove = NewValueFn[pseudoclass](vfn("groove"), nil).Setter("border-bottom-style")
	c.BorderBottomStyle.Ridge = NewValueFn[pseudoclass](vfn("ridge"), nil).Setter("border-bottom-style")
	c.BorderBottomStyle.Inset = NewValueFn[pseudoclass](vfn("inset"), nil).Setter("border-bottom-style")
	c.BorderBottomStyle.Outset = NewValueFn[pseudoclass](vfn("outset"), nil).Setter("border-bottom-style")
	c.BorderBottomStyle.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-bottom-style")

	// WordBreak
	c.WordBreak.Normal = NewValueFn[pseudoclass](vfn("normal"), nil).Setter("word-break")
	c.WordBreak.BreakAll = NewValueFn[pseudoclass](vfn("break-all"), nil).Setter("word-break")
	c.WordBreak.KeepAll = NewValueFn[pseudoclass](vfn("keep-all"), nil).Setter("word-break")
	c.WordBreak.BreakWord = NewValueFn[pseudoclass](vfn("break-word"), nil).Setter("word-break")
	c.WordBreak.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("word-break")

	// BorderBottomRightRadius
	c.BorderBottomRightRadius.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-bottom-right-radius")

	// FontStyle
	c.FontStyle.Normal = NewValueFn[pseudoclass](vfn("normal"), nil).Setter("font-style")
	c.FontStyle.Italic = NewValueFn[pseudoclass](vfn("italic"), nil).Setter("font-style")
	c.FontStyle.Oblique = NewValueFn[pseudoclass](vfn("oblique"), nil).Setter("font-style")
	c.FontStyle.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("font-style")

	// Order
	c.Order.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("order")

	// OutlineStyle
	c.OutlineStyle.None = NewValueFn[pseudoclass](vfn("none"), nil).Setter("outline-style")
	c.OutlineStyle.Hidden = NewValueFn[pseudoclass](vfn("hidden"), nil).Setter("outline-style")
	c.OutlineStyle.Dotted = NewValueFn[pseudoclass](vfn("dotted"), nil).Setter("outline-style")
	c.OutlineStyle.Dashed = NewValueFn[pseudoclass](vfn("dashed"), nil).Setter("outline-style")
	c.OutlineStyle.Solid = NewValueFn[pseudoclass](vfn("solid"), nil).Setter("outline-style")
	c.OutlineStyle.Double = NewValueFn[pseudoclass](vfn("double"), nil).Setter("outline-style")
	c.OutlineStyle.Groove = NewValueFn[pseudoclass](vfn("groove"), nil).Setter("outline-style")
	c.OutlineStyle.Ridge = NewValueFn[pseudoclass](vfn("ridge"), nil).Setter("outline-style")
	c.OutlineStyle.Inset = NewValueFn[pseudoclass](vfn("inset"), nil).Setter("outline-style")
	c.OutlineStyle.Outset = NewValueFn[pseudoclass](vfn("outset"), nil).Setter("outline-style")
	c.OutlineStyle.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("outline-style")

	// BorderBottomLeftRadius
	c.BorderBottomLeftRadius.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-bottom-left-radius")

	// BorderImageSource
	c.BorderImageSource.None = NewValueFn[pseudoclass](vfn("none"), nil).Setter("border-image-source")
	c.BorderImageSource.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-image-source")

	// TextAlignLast
	c.TextAlignLast.Auto = NewValueFn[pseudoclass](vfn("auto"), nil).Setter("text-align-last")
	c.TextAlignLast.Left = NewValueFn[pseudoclass](vfn("left"), nil).Setter("text-align-last")
	c.TextAlignLast.Right = NewValueFn[pseudoclass](vfn("right"), nil).Setter("text-align-last")
	c.TextAlignLast.Center = NewValueFn[pseudoclass](vfn("center"), nil).Setter("text-align-last")
	c.TextAlignLast.Justify = NewValueFn[pseudoclass](vfn("justify"), nil).Setter("text-align-last")
	c.TextAlignLast.Start = NewValueFn[pseudoclass](vfn("start"), nil).Setter("text-align-last")
	c.TextAlignLast.End = NewValueFn[pseudoclass](vfn("end"), nil).Setter("text-align-last")
	c.TextAlignLast.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("text-align-last")

	// BorderImageWidth
	c.BorderImageWidth.Auto = NewValueFn[pseudoclass](vfn("auto"), nil).Setter("border-image-width")
	c.BorderImageWidth.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-image-width")

	// FontWeight
	c.FontWeight.Normal = NewValueFn[pseudoclass](vfn("normal"), nil).Setter("font-weight")
	c.FontWeight.Bold = NewValueFn[pseudoclass](vfn("bold"), nil).Setter("font-weight")
	c.FontWeight.Bolder = NewValueFn[pseudoclass](vfn("bolder"), nil).Setter("font-weight")
	c.FontWeight.Lighter = NewValueFn[pseudoclass](vfn("lighter"), nil).Setter("font-weight")
	c.FontWeight.S100 = NewValueFn[pseudoclass](vfn("100"), nil).Setter("font-weight")
	c.FontWeight.S200 = NewValueFn[pseudoclass](vfn("200"), nil).Setter("font-weight")
	c.FontWeight.S300 = NewValueFn[pseudoclass](vfn("300"), nil).Setter("font-weight")
	c.FontWeight.S400 = NewValueFn[pseudoclass](vfn("400"), nil).Setter("font-weight")
	c.FontWeight.S500 = NewValueFn[pseudoclass](vfn("500"), nil).Setter("font-weight")
	c.FontWeight.S600 = NewValueFn[pseudoclass](vfn("600"), nil).Setter("font-weight")
	c.FontWeight.S700 = NewValueFn[pseudoclass](vfn("700"), nil).Setter("font-weight")
	c.FontWeight.S800 = NewValueFn[pseudoclass](vfn("800"), nil).Setter("font-weight")
	c.FontWeight.S900 = NewValueFn[pseudoclass](vfn("900"), nil).Setter("font-weight")
	c.FontWeight.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("font-weight")

	// ListStyleImage
	c.ListStyleImage.None = NewValueFn[pseudoclass](vfn("none"), nil).Setter("list-style-image")
	c.ListStyleImage.Url = NewValueFn[pseudoclass](nil, cfn).CustomSetter("list-style-image")  // Special handling might be needed for URL
	c.ListStyleImage.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("list-style-image")

	// Opacity
	c.Opacity.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("opacity")

	// Clear
	c.Clear.None = NewValueFn[pseudoclass](vfn("none"), nil).Setter("clear")
	c.Clear.Left = NewValueFn[pseudoclass](vfn("left"), nil).Setter("clear")
	c.Clear.Right = NewValueFn[pseudoclass](vfn("right"), nil).Setter("clear")
	c.Clear.Both = NewValueFn[pseudoclass](vfn("both"), nil).Setter("clear")
	c.Clear.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("clear")

	// BorderTopColor
	c.BorderTopColor.Transparent = NewValueFn[pseudoclass](vfn("transparent"), nil).Setter("border-top-color")
	c.BorderTopColor.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-top-color")

	// Border
	c.Border.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border")

	// BorderRightColor
	c.BorderRightColor.Transparent = NewValueFn[pseudoclass](vfn("transparent"), nil).Setter("border-right-color")
	c.BorderRightColor.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-right-color")

	// TransitionTimingFunction
	c.TransitionTimingFunction.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("transition-timing-function")

	// BorderBottomWidth
	c.BorderBottomWidth.Medium = NewValueFn[pseudoclass](vfn("medium"), nil).Setter("border-bottom-width")
	c.BorderBottomWidth.Thin = NewValueFn[pseudoclass](vfn("thin"), nil).Setter("border-bottom-width")
	c.BorderBottomWidth.Thick = NewValueFn[pseudoclass](vfn("thick"), nil).Setter("border-bottom-width")
	c.BorderBottomWidth.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-bottom-width")

	// BorderStyle
	c.BorderStyle.None = NewValueFn[pseudoclass](vfn("none"), nil).Setter("border-style")
	c.BorderStyle.Hidden = NewValueFn[pseudoclass](vfn("hidden"), nil).Setter("border-style")
	c.BorderStyle.Dotted = NewValueFn[pseudoclass](vfn("dotted"), nil).Setter("border-style")
	c.BorderStyle.Dashed = NewValueFn[pseudoclass](vfn("dashed"), nil).Setter("border-style")
	c.BorderStyle.Solid = NewValueFn[pseudoclass](vfn("solid"), nil).Setter("border-style")
	c.BorderStyle.Double = NewValueFn[pseudoclass](vfn("double"), nil).Setter("border-style")
	c.BorderStyle.Groove = NewValueFn[pseudoclass](vfn("groove"), nil).Setter("border-style")
	c.BorderStyle.Ridge = NewValueFn[pseudoclass](vfn("ridge"), nil).Setter("border-style")
	c.BorderStyle.Inset = NewValueFn[pseudoclass](vfn("inset"), nil).Setter("border-style")
	c.BorderStyle.Outset = NewValueFn[pseudoclass](vfn("outset"), nil).Setter("border-style")
	c.BorderStyle.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-style")

	// BorderTopRightRadius
	c.BorderTopRightRadius.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-top-right-radius")

	// CaptionSide
	c.CaptionSide.Top = NewValueFn[pseudoclass](vfn("top"), nil).Setter("caption-side")
	c.CaptionSide.Bottom = NewValueFn[pseudoclass](vfn("bottom"), nil).Setter("caption-side")
	c.CaptionSide.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("caption-side")

	// FontFamily
	c.FontFamily.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("font-family")

	// TextDecorationColor
	c.TextDecorationColor.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("text-decoration-color")

	// TransitionProperty
	c.TransitionProperty.None = NewValueFn[pseudoclass](vfn("none"), nil).Setter("transition-property")
	c.TransitionProperty.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("transition-property")

	// BackgroundOrigin
	c.BackgroundOrigin.PaddingBox = NewValueFn[pseudoclass](vfn("padding-box"), nil).Setter("background-origin")
	c.BackgroundOrigin.BorderBox = NewValueFn[pseudoclass](vfn("border-box"), nil).Setter("background-origin")
	c.BackgroundOrigin.ContentBox = NewValueFn[pseudoclass](vfn("content-box"), nil).Setter("background-origin")
	c.BackgroundOrigin.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("background-origin")

	// TextIndent
	c.TextIndent.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("text-indent")

	// Visibility
	c.Visibility.Visible = NewValueFn[pseudoclass](vfn("visible"), nil).Setter("visibility")
	c.Visibility.Hidden = NewValueFn[pseudoclass](vfn("hidden"), nil).Setter("visibility")
	c.Visibility.Collapse = NewValueFn[pseudoclass](vfn("collapse"), nil).Setter("visibility")
	c.Visibility.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("visibility")

	// BorderColor
	c.BorderColor.Transparent = NewValueFn[pseudoclass](vfn("transparent"), nil).Setter("border-color")
	c.BorderColor.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-color")

	// BorderTop
	c.BorderTop.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-top")

	// FontVariant
	c.FontVariant.Normal = NewValueFn[pseudoclass](vfn("normal"), nil).Setter("font-variant")
	c.FontVariant.SmallCaps = NewValueFn[pseudoclass](vfn("small-caps"), nil).Setter("font-variant")
	c.FontVariant.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("font-variant")

	// Outline
	c.Outline.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("outline")

	// BorderBottomColor
	c.BorderBottomColor.Transparent = NewValueFn[pseudoclass](vfn("transparent"), nil).Setter("border-bottom-color")
	c.BorderBottomColor.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-bottom-color")

	// BorderTopStyle
	c.BorderTopStyle.None = NewValueFn[pseudoclass](vfn("none"), nil).Setter("border-top-style")
	c.BorderTopStyle.Hidden = NewValueFn[pseudoclass](vfn("hidden"), nil).Setter("border-top-style")
	c.BorderTopStyle.Dotted = NewValueFn[pseudoclass](vfn("dotted"), nil).Setter("border-top-style")
	c.BorderTopStyle.Dashed = NewValueFn[pseudoclass](vfn("dashed"), nil).Setter("border-top-style")
	c.BorderTopStyle.Solid = NewValueFn[pseudoclass](vfn("solid"), nil).Setter("border-top-style")
	c.BorderTopStyle.Double = NewValueFn[pseudoclass](vfn("double"), nil).Setter("border-top-style")
	c.BorderTopStyle.Groove = NewValueFn[pseudoclass](vfn("groove"), nil).Setter("border-top-style")
	c.BorderTopStyle.Ridge = NewValueFn[pseudoclass](vfn("ridge"), nil).Setter("border-top-style")
	c.BorderTopStyle.Inset = NewValueFn[pseudoclass](vfn("inset"), nil).Setter("border-top-style")
	c.BorderTopStyle.Outset = NewValueFn[pseudoclass](vfn("outset"), nil).Setter("border-top-style")
	c.BorderTopStyle.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-top-style")

	// BorderWidth
	c.BorderWidth.Medium = NewValueFn[pseudoclass](vfn("medium"), nil).Setter("border-width")
	c.BorderWidth.Thin = NewValueFn[pseudoclass](vfn("thin"), nil).Setter("border-width")
	c.BorderWidth.Thick = NewValueFn[pseudoclass](vfn("thick"), nil).Setter("border-width")
	c.BorderWidth.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-width")

	// ListStyleType
	c.ListStyleType.Disc = NewValueFn[pseudoclass](vfn("disc"), nil).Setter("list-style-type")
	c.ListStyleType.Armenian = NewValueFn[pseudoclass](vfn("armenian"), nil).Setter("list-style-type")
	c.ListStyleType.Circle = NewValueFn[pseudoclass](vfn("circle"), nil).Setter("list-style-type")
	c.ListStyleType.CjkIdeographic = NewValueFn[pseudoclass](vfn("cjk-ideographic"), nil).Setter("list-style-type")
	c.ListStyleType.Decimal = NewValueFn[pseudoclass](vfn("decimal"), nil).Setter("list-style-type")
	c.ListStyleType.DecimalLeadingZero = NewValueFn[pseudoclass](vfn("decimal-leading-zero"), nil).Setter("list-style-type")
	c.ListStyleType.Georgian = NewValueFn[pseudoclass](vfn("georgian"), nil).Setter("list-style-type")
	c.ListStyleType.Hebrew = NewValueFn[pseudoclass](vfn("hebrew"), nil).Setter("list-style-type")
	c.ListStyleType.Hiragana = NewValueFn[pseudoclass](vfn("hiragana"), nil).Setter("list-style-type")
	c.ListStyleType.HiraganaIroha = NewValueFn[pseudoclass](vfn("hiragana-iroha"), nil).Setter("list-style-type")
	c.ListStyleType.Katakana = NewValueFn[pseudoclass](vfn("katakana"), nil).Setter("list-style-type")
	c.ListStyleType.KatakanaIroha = NewValueFn[pseudoclass](vfn("katakana-iroha"), nil).Setter("list-style-type")
	c.ListStyleType.LowerAlpha = NewValueFn[pseudoclass](vfn("lower-alpha"), nil).Setter("list-style-type")
	c.ListStyleType.LowerGreek = NewValueFn[pseudoclass](vfn("lower-greek"), nil).Setter("list-style-type")
	c.ListStyleType.LowerLatin = NewValueFn[pseudoclass](vfn("lower-latin"), nil).Setter("list-style-type")
	c.ListStyleType.LowerRoman = NewValueFn[pseudoclass](vfn("lower-roman"), nil).Setter("list-style-type")
	c.ListStyleType.None = NewValueFn[pseudoclass](vfn("none"), nil).Setter("list-style-type")
	c.ListStyleType.Square = NewValueFn[pseudoclass](vfn("square"), nil).Setter("list-style-type")
	c.ListStyleType.UpperAlpha = NewValueFn[pseudoclass](vfn("upper-alpha"), nil).Setter("list-style-type")
	c.ListStyleType.UpperGreek = NewValueFn[pseudoclass](vfn("upper-greek"), nil).Setter("list-style-type")
	c.ListStyleType.UpperLatin = NewValueFn[pseudoclass](vfn("upper-latin"), nil).Setter("list-style-type")
	c.ListStyleType.UpperRoman = NewValueFn[pseudoclass](vfn("upper-roman"), nil).Setter("list-style-type")
	c.ListStyleType.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("list-style-type")

	// OutlineOffset
	c.OutlineOffset.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("outline-offset")

	// Animation
	c.Animation.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("animation")

	// Background
	c.Background.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("background")

	// BackgroundRepeat
	c.BackgroundRepeat.Repeat = NewValueFn[pseudoclass](vfn("repeat"), nil).Setter("background-repeat")
	c.BackgroundRepeat.RepeatX = NewValueFn[pseudoclass](vfn("repeat-x"), nil).Setter("background-repeat")
	c.BackgroundRepeat.RepeatY = NewValueFn[pseudoclass](vfn("repeat-y"), nil).Setter("background-repeat")
	c.BackgroundRepeat.NoRepeat = NewValueFn[pseudoclass](vfn("no-repeat"), nil).Setter("background-repeat")
	c.BackgroundRepeat.Space = NewValueFn[pseudoclass](vfn("space"), nil).Setter("background-repeat")
	c.BackgroundRepeat.Round = NewValueFn[pseudoclass](vfn("round"), nil).Setter("background-repeat")
	c.BackgroundRepeat.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("background-repeat")

	// BorderTopWidth
	c.BorderTopWidth.Medium = NewValueFn[pseudoclass](vfn("medium"), nil).Setter("border-top-width")
	c.BorderTopWidth.Thin = NewValueFn[pseudoclass](vfn("thin"), nil).Setter("border-top-width")
	c.BorderTopWidth.Thick = NewValueFn[pseudoclass](vfn("thick"), nil).Setter("border-top-width")
	c.BorderTopWidth.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-top-width")

	// WordWrap
	c.WordWrap.Normal = NewValueFn[pseudoclass](vfn("normal"), nil).Setter("word-wrap")
	c.WordWrap.BreakWord = NewValueFn[pseudoclass](vfn("break-word"), nil).Setter("word-wrap")
	c.WordWrap.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("word-wrap")

	// BackgroundColor
	c.BackgroundColor.Transparent = NewValueFn[pseudoclass](vfn("transparent"), nil).Setter("background-color")
	c.BackgroundColor.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("background-color")

	// TextOverflow
	c.TextOverflow.Clip = NewValueFn[pseudoclass](vfn("clip"), nil).Setter("text-overflow")
	c.TextOverflow.Ellipsis = NewValueFn[pseudoclass](vfn("ellipsis"), nil).Setter("text-overflow")
	c.TextOverflow.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("text-overflow")

	// TextShadow
	c.TextShadow.None = NewValueFn[pseudoclass](vfn("none"), nil).Setter("text-shadow")
	c.TextShadow.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("text-shadow")

	// BackgroundClip
	c.BackgroundClip.BorderBox = NewValueFn[pseudoclass](vfn("border-box"), nil).Setter("background-clip")
	c.BackgroundClip.PaddingBox = NewValueFn[pseudoclass](vfn("padding-box"), nil).Setter("background-clip")
	c.BackgroundClip.ContentBox = NewValueFn[pseudoclass](vfn("content-box"), nil).Setter("background-clip")
	c.BackgroundClip.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("background-clip")

	// BorderLeftWidth
	c.BorderLeftWidth.Medium = NewValueFn[pseudoclass](vfn("medium"), nil).Setter("border-left-width")
	c.BorderLeftWidth.Thin = NewValueFn[pseudoclass](vfn("thin"), nil).Setter("border-left-width")
	c.BorderLeftWidth.Thick = NewValueFn[pseudoclass](vfn("thick"), nil).Setter("border-left-width")
	c.BorderLeftWidth.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-left-width")

	// Resize
	c.Resize.None = NewValueFn[pseudoclass](vfn("none"), nil).Setter("resize")
	c.Resize.Both = NewValueFn[pseudoclass](vfn("both"), nil).Setter("resize")
	c.Resize.Horizontal = NewValueFn[pseudoclass](vfn("horizontal"), nil).Setter("resize")
	c.Resize.Vertical = NewValueFn[pseudoclass](vfn("vertical"), nil).Setter("resize")
	c.Resize.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("resize")

	// AnimationDirection
	c.AnimationDirection.Normal = NewValueFn[pseudoclass](vfn("normal"), nil).Setter("animation-direction")
	c.AnimationDirection.Reverse = NewValueFn[pseudoclass](vfn("reverse"), nil).Setter("animation-direction")
	c.AnimationDirection.Alternate = NewValueFn[pseudoclass](vfn("alternate"), nil).Setter("animation-direction")
	c.AnimationDirection.AlternateReverse = NewValueFn[pseudoclass](vfn("alternate-reverse"), nil).Setter("animation-direction")
	c.AnimationDirection.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("animation-direction")

	// Color
	c.Color.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("color")

	// OutlineColor
	c.OutlineColor.Invert = NewValueFn[pseudoclass](vfn("invert"), nil).Setter("outline-color")
	c.OutlineColor.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("outline-color")

	// BorderImageRepeat
	c.BorderImageRepeat.Stretch = NewValueFn[pseudoclass](vfn("stretch"), nil).Setter("border-image-repeat")
	c.BorderImageRepeat.Repeat = NewValueFn[pseudoclass](vfn("repeat"), nil).Setter("border-image-repeat")
	c.BorderImageRepeat.Round = NewValueFn[pseudoclass](vfn("round"), nil).Setter("border-image-repeat")
	c.BorderImageRepeat.Space = NewValueFn[pseudoclass](vfn("space"), nil).Setter("border-image-repeat")
	c.BorderImageRepeat.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("border-image-repeat")

	// FontStretch
	c.FontStretch.UltraCondensed = NewValueFn[pseudoclass](vfn("ultra-condensed"), nil).Setter("font-stretch")
	c.FontStretch.ExtraCondensed = NewValueFn[pseudoclass](vfn("extra-condensed"), nil).Setter("font-stretch")
	c.FontStretch.Condensed = NewValueFn[pseudoclass](vfn("condensed"), nil).Setter("font-stretch")
	c.FontStretch.SemiCondensed = NewValueFn[pseudoclass](vfn("semi-condensed"), nil).Setter("font-stretch")
	c.FontStretch.Normal = NewValueFn[pseudoclass](vfn("normal"), nil).Setter("font-stretch")
	c.FontStretch.SemiExpanded = NewValueFn[pseudoclass](vfn("semi-expanded"), nil).Setter("font-stretch")
	c.FontStretch.Expanded = NewValueFn[pseudoclass](vfn("expanded"), nil).Setter("font-stretch")
	c.FontStretch.ExtraExpanded = NewValueFn[pseudoclass](vfn("extra-expanded"), nil).Setter("font-stretch")
	c.FontStretch.UltraExpanded = NewValueFn[pseudoclass](vfn("ultra-expanded"), nil).Setter("font-stretch")
	c.FontStretch.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("font-stretch")

	// TextTransform
	c.TextTransform.None = NewValueFn[pseudoclass](vfn("none"), nil).Setter("text-transform")
	c.TextTransform.Capitalize = NewValueFn[pseudoclass](vfn("capitalize"), nil).Setter("text-transform")
	c.TextTransform.Uppercase = NewValueFn[pseudoclass](vfn("uppercase"), nil).Setter("text-transform")
	c.TextTransform.Lowercase = NewValueFn[pseudoclass](vfn("lowercase"), nil).Setter("text-transform")
	c.TextTransform.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("text-transform")

	return c
}

func initializeContentLayout[pseudoclass any]() ContentLayout{
	c:= ContentLayout{}

	// FlexGrow
	c.FlexGrow.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("flex-grow")

	// AlignSelf
	c.AlignSelf.Auto = NewValueFn[pseudoclass](vfn("auto"), nil).Setter("align-self")
	c.AlignSelf.Stretch = NewValueFn[pseudoclass](vfn("stretch"), nil).Setter("align-self")
	c.AlignSelf.Center = NewValueFn[pseudoclass](vfn("center"), nil).Setter("align-self")
	c.AlignSelf.FlexStart = NewValueFn[pseudoclass](vfn("flex-start"), nil).Setter("align-self")
	c.AlignSelf.FlexEnd = NewValueFn[pseudoclass](vfn("flex-end"), nil).Setter("align-self")
	c.AlignSelf.Baseline = NewValueFn[pseudoclass](vfn("baseline"), nil).Setter("align-self")
	c.AlignSelf.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("align-self")

	// Content
	c.Content.Normal = NewValueFn[pseudoclass](vfn("normal"), nil).Setter("content")
	c.Content.None = NewValueFn[pseudoclass](vfn("none"), nil).Setter("content")
	c.Content.Counter = NewValueFn[pseudoclass](vfn("counter"), nil).Setter("content")
	c.Content.Attr = NewValueFn[pseudoclass](nil, cfn).CustomSetter("content") // Adjust for attribute-based content.
	c.Content.String = NewValueFn[pseudoclass](vfn("string"), nil).Setter("content")
	c.Content.OpenQuote = NewValueFn[pseudoclass](vfn("open-quote"), nil).Setter("content")
	c.Content.CloseQuote = NewValueFn[pseudoclass](vfn("close-quote"), nil).Setter("content")
	c.Content.NoOpenQuote = NewValueFn[pseudoclass](vfn("no-open-quote"), nil).Setter("content")
	c.Content.NoCloseQuote = NewValueFn[pseudoclass](vfn("no-close-quote"), nil).Setter("content")
	c.Content.URL = NewValueFn[pseudoclass](nil, cfn).CustomSetter("content") // Adjust for URL-based content.
	c.Content.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("content")

	// ColumnSpan
	c.ColumnSpan.None = NewValueFn[pseudoclass](vfn("none"), nil).Setter("column-span")
	c.ColumnSpan.All = NewValueFn[pseudoclass](vfn("all"), nil).Setter("column-span")
	c.ColumnSpan.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("column-span")

	// Flex
	c.Flex.FlexGrow = NewValueFn[pseudoclass](nil, cfn).CustomSetter("flex-grow")
	c.Flex.FlexShrink = NewValueFn[pseudoclass](nil, cfn).CustomSetter("flex-shrink")
	c.Flex.FlexBasis = NewValueFn[pseudoclass](nil, cfn).CustomSetter("flex-basis")
	c.Flex.Auto = NewValueFn[pseudoclass](vfn("auto"), nil).Setter("flex")
	c.Flex.Initial = NewValueFn[pseudoclass](vfn("initial"), nil).Setter("flex")
	c.Flex.None = NewValueFn[pseudoclass](vfn("none"), nil).Setter("flex")
	c.Flex.Inherit = NewValueFn[pseudoclass](vfn("inherit"), nil).Setter("flex")

	// FlexShrink
	c.FlexShrink.Number = NewValueFn[pseudoclass](nil, cfn).CustomSetter("flex-shrink")
	c.FlexShrink.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("flex-shrink")

	// Order
	c.Order.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("order")

	// FlexBasis
	c.FlexBasis.Auto = NewValueFn[pseudoclass](vfn("auto"), nil).Setter("flex-basis")
	c.FlexBasis.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("flex-basis")

	// AlignItems
	c.AlignItems.Stretch = NewValueFn[pseudoclass](vfn("stretch"), nil).Setter("align-items")
	c.AlignItems.Center = NewValueFn[pseudoclass](vfn("center"), nil).Setter("align-items")
	c.AlignItems.FlexStart = NewValueFn[pseudoclass](vfn("flex-start"), nil).Setter("align-items")
	c.AlignItems.FlexEnd = NewValueFn[pseudoclass](vfn("flex-end"), nil).Setter("align-items")
	c.AlignItems.Baseline = NewValueFn[pseudoclass](vfn("baseline"), nil).Setter("align-items")
	c.AlignItems.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("align-items")

	return c
}

func initializeContentStyle[pseudoclass any]() ContentStyle{
	c:= ContentStyle{}

	// ColumnRuleWidth initialization
	c.ColumnRuleWidth.Medium = NewValueFn[pseudoclass](vfn("medium"), nil).Setter("column-rule-width")
	c.ColumnRuleWidth.Thin = NewValueFn[pseudoclass](vfn("thin"), nil).Setter("column-rule-width")
	c.ColumnRuleWidth.Thick = NewValueFn[pseudoclass](vfn("thick"), nil).Setter("column-rule-width")
	c.ColumnRuleWidth.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("column-rule-width")

		
	// ColumnRule initialization
	c.ColumnRule.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("column-rule")

	// Direction initialization
	c.Direction.Ltr = NewValueFn[pseudoclass](vfn("ltr"), nil).Setter("direction")
	c.Direction.Rtl = NewValueFn[pseudoclass](vfn("rtl"), nil).Setter("direction")
	c.Direction.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("direction")

	// ColumnRuleStyle initialization
	c.ColumnRuleStyle.None = NewValueFn[pseudoclass](vfn("none"), nil).Setter("column-rule-style")
	c.ColumnRuleStyle.Hidden = NewValueFn[pseudoclass](vfn("hidden"), nil).Setter("column-rule-style")
	c.ColumnRuleStyle.Dotted = NewValueFn[pseudoclass](vfn("dotted"), nil).Setter("column-rule-style")
	c.ColumnRuleStyle.Dashed = NewValueFn[pseudoclass](vfn("dashed"), nil).Setter("column-rule-style")
	c.ColumnRuleStyle.Solid = NewValueFn[pseudoclass](vfn("solid"), nil).Setter("column-rule-style")
	c.ColumnRuleStyle.Double = NewValueFn[pseudoclass](vfn("double"), nil).Setter("column-rule-style")
	c.ColumnRuleStyle.Groove = NewValueFn[pseudoclass](vfn("groove"), nil).Setter("column-rule-style")
	c.ColumnRuleStyle.Ridge = NewValueFn[pseudoclass](vfn("ridge"), nil).Setter("column-rule-style")
	c.ColumnRuleStyle.Inset = NewValueFn[pseudoclass](vfn("inset"), nil).Setter("column-rule-style")
	c.ColumnRuleStyle.Outset = NewValueFn[pseudoclass](vfn("outset"), nil).Setter("column-rule-style")
	c.ColumnRuleStyle.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("column-rule-style")

	
	// ColumnRuleColor initialization
	c.ColumnRuleColor.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("column-rule-color")

	// ColumnFill initialization
	c.ColumnFill.Balance = NewValueFn[pseudoclass](vfn("balance"), nil).Setter("column-fill")
	c.ColumnFill.Auto = NewValueFn[pseudoclass](vfn("auto"), nil).Setter("column-fill")
	c.ColumnFill.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("column-fill")

	// EmptyCells initialization
	c.EmptyCells.Show = NewValueFn[pseudoclass](vfn("show"), nil).Setter("empty-cells")
	c.EmptyCells.Hide = NewValueFn[pseudoclass](vfn("hide"), nil).Setter("empty-cells")
	c.EmptyCells.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("empty-cells")
	
	// Cursor initialization
	c.Cursor.Alias = NewValueFn[pseudoclass](vfn("alias"), nil).Setter("cursor")
	c.Cursor.AllScroll = NewValueFn[pseudoclass](vfn("all-scroll"), nil).Setter("cursor")
	c.Cursor.Auto = NewValueFn[pseudoclass](vfn("auto"), nil).Setter("cursor")
	c.Cursor.Cell = NewValueFn[pseudoclass](vfn("cell"), nil).Setter("cursor")
	c.Cursor.ContextMenu = NewValueFn[pseudoclass](vfn("context-menu"), nil).Setter("cursor")
	c.Cursor.ColResize = NewValueFn[pseudoclass](vfn("col-resize"), nil).Setter("cursor")
	c.Cursor.Copy = NewValueFn[pseudoclass](vfn("copy"), nil).Setter("cursor")
	c.Cursor.Crosshair = NewValueFn[pseudoclass](vfn("crosshair"), nil).Setter("cursor")
	c.Cursor.Default = NewValueFn[pseudoclass](vfn("default"), nil).Setter("cursor")
	c.Cursor.EResize = NewValueFn[pseudoclass](vfn("e-resize"), nil).Setter("cursor")
	c.Cursor.EwResize = NewValueFn[pseudoclass](vfn("ew-resize"), nil).Setter("cursor")
	c.Cursor.Grab = NewValueFn[pseudoclass](vfn("grab"), nil).Setter("cursor")
	c.Cursor.Grabbing = NewValueFn[pseudoclass](vfn("grabbing"), nil).Setter("cursor")
	c.Cursor.Help = NewValueFn[pseudoclass](vfn("help"), nil).Setter("cursor")
	c.Cursor.Move = NewValueFn[pseudoclass](vfn("move"), nil).Setter("cursor")
	c.Cursor.NResize = NewValueFn[pseudoclass](vfn("n-resize"), nil).Setter("cursor")
	c.Cursor.NeResize = NewValueFn[pseudoclass](vfn("ne-resize"), nil).Setter("cursor")
	c.Cursor.NeswResize = NewValueFn[pseudoclass](vfn("nesw-resize"), nil).Setter("cursor")
	c.Cursor.NsResize = NewValueFn[pseudoclass](vfn("ns-resize"), nil).Setter("cursor")
	c.Cursor.NwResize = NewValueFn[pseudoclass](vfn("nw-resize"), nil).Setter("cursor")
	c.Cursor.NwseResize = NewValueFn[pseudoclass](vfn("nwse-resize"), nil).Setter("cursor")
	c.Cursor.NoDrop = NewValueFn[pseudoclass](vfn("no-drop"), nil).Setter("cursor")
	c.Cursor.None = NewValueFn[pseudoclass](vfn("none"), nil).Setter("cursor")
	c.Cursor.NotAllowed = NewValueFn[pseudoclass](vfn("not-allowed"), nil).Setter("cursor")
	c.Cursor.Pointer = NewValueFn[pseudoclass](vfn("pointer"), nil).Setter("cursor")
	c.Cursor.Progress = NewValueFn[pseudoclass](vfn("progress"), nil).Setter("cursor")
	c.Cursor.RowResize = NewValueFn[pseudoclass](vfn("row-resize"), nil).Setter("cursor")
	c.Cursor.SResize = NewValueFn[pseudoclass](vfn("s-resize"), nil).Setter("cursor")
	c.Cursor.SeResize = NewValueFn[pseudoclass](vfn("se-resize"), nil).Setter("cursor")
	c.Cursor.SwResize = NewValueFn[pseudoclass](vfn("sw-resize"), nil).Setter("cursor")
	c.Cursor.Text = NewValueFn[pseudoclass](vfn("text"), nil).Setter("cursor")
	c.Cursor.VerticalText = NewValueFn[pseudoclass](vfn("vertical-text"), nil).Setter("cursor")
	c.Cursor.WResize = NewValueFn[pseudoclass](vfn("w-resize"), nil).Setter("cursor")
	c.Cursor.Wait = NewValueFn[pseudoclass](vfn("wait"), nil).Setter("cursor")
	c.Cursor.ZoomIn = NewValueFn[pseudoclass](vfn("zoom-in"), nil).Setter("cursor")
	c.Cursor.ZoomOut = NewValueFn[pseudoclass](vfn("zoom-out"), nil).Setter("cursor")
	c.Cursor.Value = NewValueFn[pseudoclass](nil, cfn).CustomSetter("cursor")

	return c
}

type ContainerLayout struct {
	BoxShadow struct {
		None func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	JustifyContent struct {
		FlexStart func(*ui.Element) *ui.Element
		FlexEnd func(*ui.Element) *ui.Element
		Center func(*ui.Element) *ui.Element
		SpaceBetween func(*ui.Element) *ui.Element
		SpaceAround func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	ZIndex struct {
		Auto func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Float struct {
		None func(*ui.Element) *ui.Element
		Left func(*ui.Element) *ui.Element
		Right func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Overflow struct {
		Visible func(*ui.Element) *ui.Element
		Hidden func(*ui.Element) *ui.Element
		Scroll func(*ui.Element) *ui.Element
		Auto func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	OverflowY struct {
		Visible func(*ui.Element) *ui.Element
		Hidden func(*ui.Element) *ui.Element
		Scroll func(*ui.Element) *ui.Element
		Auto func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Perspective struct {
		None func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderCollapse struct {
		Separate func(*ui.Element) *ui.Element
		Collapse func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	PageBreakBefore struct {
		Auto func(*ui.Element) *ui.Element
		Always func(*ui.Element) *ui.Element
		Avoid func(*ui.Element) *ui.Element
		Left func(*ui.Element) *ui.Element
		Right func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Columns struct {
		Auto func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	ColumnCount struct {
		Auto func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	MinHeight struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	PageBreakInside struct {
		Auto func(*ui.Element) *ui.Element
		Avoid func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	ColumnGap struct {
		Length func(value string) func(*ui.Element) *ui.Element
		Normal func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Clip struct {
		Auto func(*ui.Element) *ui.Element
		Shape func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	FlexDirection struct {
		Row func(*ui.Element) *ui.Element
		RowReverse func(*ui.Element) *ui.Element
		Column func(*ui.Element) *ui.Element
		ColumnReverse func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	PageBreakAfter struct {
		Auto func(*ui.Element) *ui.Element
		Always func(*ui.Element) *ui.Element
		Avoid func(*ui.Element) *ui.Element
		Left func(*ui.Element) *ui.Element
		Right func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Top struct {
		Auto func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	CounterIncrement struct {
		None func(*ui.Element) *ui.Element
		Number func(value string) func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Height struct {
		Auto func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	TransformStyle struct {
		Flat func(*ui.Element) *ui.Element
		Preserve3d func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	OverflowX struct {
		Visible func(*ui.Element) *ui.Element
		Hidden func(*ui.Element) *ui.Element
		Scroll func(*ui.Element) *ui.Element
		Auto func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	FlexWrap struct {
		Nowrap func(*ui.Element) *ui.Element
		Wrap func(*ui.Element) *ui.Element
		WrapReverse func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	MaxWidth struct {
		None func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Bottom struct {
		Auto func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	CounterReset struct {
		None func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Right struct {
		Auto func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BoxSizing struct {
		ContentBox func(*ui.Element) *ui.Element
		BorderBox func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Position struct {
		Static func(*ui.Element) *ui.Element
		Absolute func(*ui.Element) *ui.Element
		Fixed func(*ui.Element) *ui.Element
		Relative func(*ui.Element) *ui.Element
		Sticky func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	TableLayout struct {
		Auto func(*ui.Element) *ui.Element
		Fixed func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Width struct {
		Auto func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	MaxHeight struct {
		None func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	ColumnWidth struct {
		Auto func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	MinWidth struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	VerticalAlign struct {
		Baseline func(*ui.Element) *ui.Element
		Top func(*ui.Element) *ui.Element
		TextTop func(*ui.Element) *ui.Element
		Middle func(*ui.Element) *ui.Element
		Bottom func(*ui.Element) *ui.Element
		TextBottom func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	PerspectiveOrigin struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	AlignContent struct {
		Stretch func(*ui.Element) *ui.Element
		Center func(*ui.Element) *ui.Element
		FlexStart func(*ui.Element) *ui.Element
		FlexEnd func(*ui.Element) *ui.Element
		SpaceBetween func(*ui.Element) *ui.Element
		SpaceAround func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	FlexFlow struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Display struct {
		Inline func(*ui.Element) *ui.Element
		Block func(*ui.Element) *ui.Element
		Contents func(*ui.Element) *ui.Element
		Flex func(*ui.Element) *ui.Element
		Grid func(*ui.Element) *ui.Element
		None func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Left struct {
		Auto func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
}

type ContainerStyle struct {
	BackgroundImage struct {
		URL func(url string) func(*ui.Element) *ui.Element
		None func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderLeftStyle struct {
		None func(*ui.Element) *ui.Element
		Hidden func(*ui.Element) *ui.Element
		Dotted func(*ui.Element) *ui.Element
		Dashed func(*ui.Element) *ui.Element
		Solid func(*ui.Element) *ui.Element
		Double func(*ui.Element) *ui.Element
		Groove func(*ui.Element) *ui.Element
		Ridge func(*ui.Element) *ui.Element
		Inset func(*ui.Element) *ui.Element
		Outset func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BoxShadow struct {
		None func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	TransitionDelay struct {
		Time func(duration string) func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	AnimationDuration struct {
		Time func(duration string) func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	ListStyle struct {
		ListStyleType func(value string) func(*ui.Element) *ui.Element
		ListStylePosition func(value string) func(*ui.Element) *ui.Element
		ListStyleImage func(value string) func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	OutlineWidth struct {
		Medium func(*ui.Element) *ui.Element
		Thin func(*ui.Element) *ui.Element
		Thick func(*ui.Element) *ui.Element
		Length func(value string) func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderTopLeftRadius struct {
		Length func(value string) func(*ui.Element) *ui.Element
		Percent func(value string) func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	WhiteSpace struct {
		Normal func(*ui.Element) *ui.Element
		Nowrap func(*ui.Element) *ui.Element
		Pre func(*ui.Element) *ui.Element
		PreLine func(*ui.Element) *ui.Element
		PreWrap func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderRight struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	TextDecorationLine struct {
		None func(*ui.Element) *ui.Element
		Underline func(*ui.Element) *ui.Element
		Overline func(*ui.Element) *ui.Element
		LineThrough func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	AnimationDelay struct {
		Time func(duration string) func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BackgroundPosition struct {
		LeftTop func(*ui.Element) *ui.Element
		LeftCenter func(*ui.Element) *ui.Element
		LeftBottom func(*ui.Element) *ui.Element
		RightTop func(*ui.Element) *ui.Element
		RightCenter func(*ui.Element) *ui.Element
		RightBottom func(*ui.Element) *ui.Element
		CenterTop func(*ui.Element) *ui.Element
		CenterCenter func(*ui.Element) *ui.Element
		CenterBottom func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
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
		Color func(value string) func(*ui.Element) *ui.Element
		Transparent func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	FontSize struct {
		Medium func(*ui.Element) *ui.Element
		XxSmall func(*ui.Element) *ui.Element
		XSmall func(*ui.Element) *ui.Element
		Small func(*ui.Element) *ui.Element
		Large func(*ui.Element) *ui.Element
		XLarge func(*ui.Element) *ui.Element
		XxLarge func(*ui.Element) *ui.Element
		Smaller func(*ui.Element) *ui.Element
		Larger func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	LineHeight struct {
		Normal func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	TextDecorationStyle struct {
		Solid func(*ui.Element) *ui.Element
		Double func(*ui.Element) *ui.Element
		Dotted func(*ui.Element) *ui.Element
		Dashed func(*ui.Element) *ui.Element
		Wavy func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BackfaceVisibility struct {
		Visible func(*ui.Element) *ui.Element
		Hidden func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderRightStyle struct {
		None func(*ui.Element) *ui.Element
		Hidden func(*ui.Element) *ui.Element
		Dotted func(*ui.Element) *ui.Element
		Dashed func(*ui.Element) *ui.Element
		Solid func(*ui.Element) *ui.Element
		Double func(*ui.Element) *ui.Element
		Groove func(*ui.Element) *ui.Element
		Ridge func(*ui.Element) *ui.Element
		Inset func(*ui.Element) *ui.Element
		Outset func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	TextDecoration struct {
		TextDecorationLine func(value string) func(*ui.Element) *ui.Element
		TextDecorationColor func(value string) func(*ui.Element) *ui.Element
		TextDecorationStyle func(value string) func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Transition struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	AnimationIterationCount struct {
		Infinite func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderBottom struct {
		BorderBottomWidth func(value string) func(*ui.Element) *ui.Element
		BorderBottomStyle func(value string) func(*ui.Element) *ui.Element
		BorderBottomColor func(value string) func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	AnimationTimingFunction struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderRadius struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Quotes struct {
		None func(*ui.Element) *ui.Element
		Auto func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	TabSize struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	AnimationFillMode struct {
		None func(*ui.Element) *ui.Element
		Forwards func(*ui.Element) *ui.Element
		Backwards func(*ui.Element) *ui.Element
		Both func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BackgroundSize struct {
		Auto func(*ui.Element) *ui.Element
		Cover func(*ui.Element) *ui.Element
		Contain func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	FontSizeAdjust struct {
		None func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	ListStylePosition struct {
		Inside func(*ui.Element) *ui.Element
		Outside func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	TextAlign struct {
		Left func(*ui.Element) *ui.Element
		Right func(*ui.Element) *ui.Element
		Center func(*ui.Element) *ui.Element
		Justify func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	TextJustify struct {
		Auto func(*ui.Element) *ui.Element
		InterWord func(*ui.Element) *ui.Element
		InterCharacter func(*ui.Element) *ui.Element
		None func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BackgroundAttachment struct {
		Scroll func(*ui.Element) *ui.Element
		Fixed func(*ui.Element) *ui.Element
		Local func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderRightWidth struct {
		Medium func(*ui.Element) *ui.Element
		Thin func(*ui.Element) *ui.Element
		Thick func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Font struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderLeft struct {
		BorderLeftWidth func(value string) func(*ui.Element) *ui.Element
		BorderLeftStyle func(value string) func(*ui.Element) *ui.Element
		BorderLeftColor func(value string) func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	TransitionDuration struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	WordSpacing struct {
		Normal func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	AnimationName struct {
		None func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	AnimationPlayState struct {
		Paused func(*ui.Element) *ui.Element
		Running func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	LetterSpacing struct {
		Normal func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderBottomStyle struct {
		None func(*ui.Element) *ui.Element
		Hidden func(*ui.Element) *ui.Element
		Dotted func(*ui.Element) *ui.Element
		Dashed func(*ui.Element) *ui.Element
		Solid func(*ui.Element) *ui.Element
		Double func(*ui.Element) *ui.Element
		Groove func(*ui.Element) *ui.Element
		Ridge func(*ui.Element) *ui.Element
		Inset func(*ui.Element) *ui.Element
		Outset func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	WordBreak struct {
		Normal func(*ui.Element) *ui.Element
		BreakAll func(*ui.Element) *ui.Element
		KeepAll func(*ui.Element) *ui.Element
		BreakWord func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderBottomRightRadius struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	FontStyle struct {
		Normal func(*ui.Element) *ui.Element
		Italic func(*ui.Element) *ui.Element
		Oblique func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Order struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	OutlineStyle struct {
		None func(*ui.Element) *ui.Element
		Hidden func(*ui.Element) *ui.Element
		Dotted func(*ui.Element) *ui.Element
		Dashed func(*ui.Element) *ui.Element
		Solid func(*ui.Element) *ui.Element
		Double func(*ui.Element) *ui.Element
		Groove func(*ui.Element) *ui.Element
		Ridge func(*ui.Element) *ui.Element
		Inset func(*ui.Element) *ui.Element
		Outset func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderBottomLeftRadius struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderImageSource struct {
		None func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	TextAlignLast struct {
		Auto func(*ui.Element) *ui.Element
		Left func(*ui.Element) *ui.Element
		Right func(*ui.Element) *ui.Element
		Center func(*ui.Element) *ui.Element
		Justify func(*ui.Element) *ui.Element
		Start func(*ui.Element) *ui.Element
		End func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderImageWidth struct {
		Auto func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	FontWeight struct {
		Normal func(*ui.Element) *ui.Element
		Bold func(*ui.Element) *ui.Element
		Bolder func(*ui.Element) *ui.Element
		Lighter func(*ui.Element) *ui.Element
		S100 func(*ui.Element) *ui.Element
		S200 func(*ui.Element) *ui.Element
		S300 func(*ui.Element) *ui.Element
		S400 func(*ui.Element) *ui.Element
		S500 func(*ui.Element) *ui.Element
		S600 func(*ui.Element) *ui.Element
		S700 func(*ui.Element) *ui.Element
		S800 func(*ui.Element) *ui.Element
		S900 func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	ListStyleImage struct {
		None func(*ui.Element) *ui.Element
		Url func(value string) (*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Opacity struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Clear struct {
		None func(*ui.Element) *ui.Element
		Left func(*ui.Element) *ui.Element
		Right func(*ui.Element) *ui.Element
		Both func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderTopColor struct {
		Transparent func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Border struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderRightColor struct {
		Transparent func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	TransitionTimingFunction struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderBottomWidth struct {
		Medium func(*ui.Element) *ui.Element
		Thin func(*ui.Element) *ui.Element
		Thick func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderStyle struct {
		None func(*ui.Element) *ui.Element
		Hidden func(*ui.Element) *ui.Element
		Dotted func(*ui.Element) *ui.Element
		Dashed func(*ui.Element) *ui.Element
		Solid func(*ui.Element) *ui.Element
		Double func(*ui.Element) *ui.Element
		Groove func(*ui.Element) *ui.Element
		Ridge func(*ui.Element) *ui.Element
		Inset func(*ui.Element) *ui.Element
		Outset func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderTopRightRadius struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	CaptionSide struct {
		Top func(*ui.Element) *ui.Element
		Bottom func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	FontFamily struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	TextDecorationColor struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	TransitionProperty struct {
		None func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BackgroundOrigin struct {
		PaddingBox func(*ui.Element) *ui.Element
		BorderBox func(*ui.Element) *ui.Element
		ContentBox func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	TextIndent struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Visibility struct {
		Visible func(*ui.Element) *ui.Element
		Hidden func(*ui.Element) *ui.Element
		Collapse func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderColor struct {
		Transparent func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderTop struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	FontVariant struct {
		Normal func(*ui.Element) *ui.Element
		SmallCaps func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Outline struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderBottomColor struct {
		Transparent func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderTopStyle struct {
		None func(*ui.Element) *ui.Element
		Hidden func(*ui.Element) *ui.Element
		Dotted func(*ui.Element) *ui.Element
		Dashed func(*ui.Element) *ui.Element
		Solid func(*ui.Element) *ui.Element
		Double func(*ui.Element) *ui.Element
		Groove func(*ui.Element) *ui.Element
		Ridge func(*ui.Element) *ui.Element
		Inset func(*ui.Element) *ui.Element
		Outset func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderWidth struct {
		Medium func(*ui.Element) *ui.Element
		Thin func(*ui.Element) *ui.Element
		Thick func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	ListStyleType struct {
		Disc func(*ui.Element) *ui.Element
		Armenian func(*ui.Element) *ui.Element
		Circle func(*ui.Element) *ui.Element
		CjkIdeographic func(*ui.Element) *ui.Element
		Decimal func(*ui.Element) *ui.Element
		DecimalLeadingZero func(*ui.Element) *ui.Element
		Georgian func(*ui.Element) *ui.Element
		Hebrew func(*ui.Element) *ui.Element
		Hiragana func(*ui.Element) *ui.Element
		HiraganaIroha func(*ui.Element) *ui.Element
		Katakana func(*ui.Element) *ui.Element
		KatakanaIroha func(*ui.Element) *ui.Element
		LowerAlpha func(*ui.Element) *ui.Element
		LowerGreek func(*ui.Element) *ui.Element
		LowerLatin func(*ui.Element) *ui.Element
		LowerRoman func(*ui.Element) *ui.Element
		None func(*ui.Element) *ui.Element
		Square func(*ui.Element) *ui.Element
		UpperAlpha func(*ui.Element) *ui.Element
		UpperGreek func(*ui.Element) *ui.Element
		UpperLatin func(*ui.Element) *ui.Element
		UpperRoman func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
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
		Repeat func(*ui.Element) *ui.Element
		RepeatX func(*ui.Element) *ui.Element
		RepeatY func(*ui.Element) *ui.Element
		NoRepeat func(*ui.Element) *ui.Element
		Space func(*ui.Element) *ui.Element
		Round func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderTopWidth struct {
		Medium func(*ui.Element) *ui.Element
		Thin func(*ui.Element) *ui.Element
		Thick func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	WordWrap struct {
		Normal func(*ui.Element) *ui.Element
		BreakWord func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BackgroundColor struct {
		Transparent func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	TextOverflow struct {
		Clip func(*ui.Element) *ui.Element
		Ellipsis func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	TextShadow struct {
		None func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BackgroundClip struct {
		BorderBox func(*ui.Element) *ui.Element
		PaddingBox func(*ui.Element) *ui.Element
		ContentBox func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderLeftWidth struct {
		Medium func(*ui.Element) *ui.Element
		Thin func(*ui.Element) *ui.Element
		Thick func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Resize struct {
		None func(*ui.Element) *ui.Element
		Both func(*ui.Element) *ui.Element
		Horizontal func(*ui.Element) *ui.Element
		Vertical func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	AnimationDirection struct {
		Normal func(*ui.Element) *ui.Element
		Reverse func(*ui.Element) *ui.Element
		Alternate func(*ui.Element) *ui.Element
		AlternateReverse func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Color struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	OutlineColor struct {
		Invert func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	BorderImageRepeat struct {
		Stretch func(*ui.Element) *ui.Element
		Repeat func(*ui.Element) *ui.Element
		Round func(*ui.Element) *ui.Element
		Space func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	FontStretch struct {
		UltraCondensed func(*ui.Element) *ui.Element
		ExtraCondensed func(*ui.Element) *ui.Element
		Condensed func(*ui.Element) *ui.Element
		SemiCondensed func(*ui.Element) *ui.Element
		Normal func(*ui.Element) *ui.Element
		SemiExpanded func(*ui.Element) *ui.Element
		Expanded func(*ui.Element) *ui.Element
		ExtraExpanded func(*ui.Element) *ui.Element
		UltraExpanded func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	TextTransform struct {
		None func(*ui.Element) *ui.Element
		Capitalize func(*ui.Element) *ui.Element
		Uppercase func(*ui.Element) *ui.Element
		Lowercase func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
}

type ContentLayout struct {
	FlexGrow struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	AlignSelf struct {
		Auto func(*ui.Element) *ui.Element
		Stretch func(*ui.Element) *ui.Element
		Center func(*ui.Element) *ui.Element
		FlexStart func(*ui.Element) *ui.Element
		FlexEnd func(*ui.Element) *ui.Element
		Baseline func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Content struct {
		Normal func(*ui.Element) *ui.Element
		None func(*ui.Element) *ui.Element
		Counter func(*ui.Element) *ui.Element
		Attr func(value string) func(*ui.Element) *ui.Element
		String func(*ui.Element) *ui.Element
		OpenQuote func(*ui.Element) *ui.Element
		CloseQuote func(*ui.Element) *ui.Element
		NoOpenQuote func(*ui.Element) *ui.Element
		NoCloseQuote func(*ui.Element) *ui.Element
		URL func(url string) func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	ColumnSpan struct {
		None func(*ui.Element) *ui.Element
		All func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Flex struct {
		FlexGrow func(value string) func(*ui.Element) *ui.Element
		FlexShrink func(value string) func(*ui.Element) *ui.Element
		FlexBasis func(value string) func(*ui.Element) *ui.Element
		Auto func(*ui.Element) *ui.Element
		Initial func(*ui.Element) *ui.Element
		None func(*ui.Element) *ui.Element
		Inherit func(*ui.Element) *ui.Element
	}
	FlexShrink struct {
		Number func(value string) func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Order struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	FlexBasis struct {
		Auto func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	AlignItems struct {
		Stretch func(*ui.Element) *ui.Element
		Center func(*ui.Element) *ui.Element
		FlexStart func(*ui.Element) *ui.Element
		FlexEnd func(*ui.Element) *ui.Element
		Baseline func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
}

type ContentStyle struct {
	ColumnRuleWidth struct {
		Medium func(*ui.Element) *ui.Element
		Thin func(*ui.Element) *ui.Element
		Thick func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	ColumnRule struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Direction struct {
		Ltr func(*ui.Element) *ui.Element
		Rtl func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	ColumnRuleStyle struct {
		None func(*ui.Element) *ui.Element
		Hidden func(*ui.Element) *ui.Element
		Dotted func(*ui.Element) *ui.Element
		Dashed func(*ui.Element) *ui.Element
		Solid func(*ui.Element) *ui.Element
		Double func(*ui.Element) *ui.Element
		Groove func(*ui.Element) *ui.Element
		Ridge func(*ui.Element) *ui.Element
		Inset func(*ui.Element) *ui.Element
		Outset func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	ColumnRuleColor struct {
		Value func(value string) func(*ui.Element) *ui.Element
	}
	ColumnFill struct {
		Balance func(*ui.Element) *ui.Element
		Auto func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	EmptyCells struct {
		Show func(*ui.Element) *ui.Element
		Hide func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
	Cursor struct {
		Alias func(*ui.Element) *ui.Element
		AllScroll func(*ui.Element) *ui.Element
		Auto func(*ui.Element) *ui.Element
		Cell func(*ui.Element) *ui.Element
		ContextMenu func(*ui.Element) *ui.Element
		ColResize func(*ui.Element) *ui.Element
		Copy func(*ui.Element) *ui.Element
		Crosshair func(*ui.Element) *ui.Element
		Default func(*ui.Element) *ui.Element
		EResize func(*ui.Element) *ui.Element
		EwResize func(*ui.Element) *ui.Element
		Grab func(*ui.Element) *ui.Element
		Grabbing func(*ui.Element) *ui.Element
		Help func(*ui.Element) *ui.Element
		Move func(*ui.Element) *ui.Element
		NResize func(*ui.Element) *ui.Element
		NeResize func(*ui.Element) *ui.Element
		NeswResize func(*ui.Element) *ui.Element
		NsResize func(*ui.Element) *ui.Element
		NwResize func(*ui.Element) *ui.Element
		NwseResize func(*ui.Element) *ui.Element
		NoDrop func(*ui.Element) *ui.Element
		None func(*ui.Element) *ui.Element
		NotAllowed func(*ui.Element) *ui.Element
		Pointer func(*ui.Element) *ui.Element
		Progress func(*ui.Element) *ui.Element
		RowResize func(*ui.Element) *ui.Element
		SResize func(*ui.Element) *ui.Element
		SeResize func(*ui.Element) *ui.Element
		SwResize func(*ui.Element) *ui.Element
		Text func(*ui.Element) *ui.Element
		VerticalText func(*ui.Element) *ui.Element
		WResize func(*ui.Element) *ui.Element
		Wait func(*ui.Element) *ui.Element
		ZoomIn func(*ui.Element) *ui.Element
		ZoomOut func(*ui.Element) *ui.Element
		Value func(value string) func(*ui.Element) *ui.Element
	}
}

