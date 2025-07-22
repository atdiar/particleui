 
const goToCSSMapping = {
    'Container.Layout.BoxShadow': {
        property: 'box-shadow',
        valueFunction: (value) => value
    },
    'Container.Layout.JustifyContent': {
        property: 'justify-content',
        valueFunction: (value) => value
    },
    'Container.Layout.ZIndex': {
        property: 'z-index',
        valueFunction: (value) => value
    },
    'Container.Layout.Float': {
        property: 'float',
        valueFunction: (value) => value
    },
    'Container.Layout.Overflow': {
        property: 'overflow',
        valueFunction: (value) => value
    },
    'Container.Layout.OverflowY': {
        property: 'overflow-y',
        valueFunction: (value) => value
    },
    'Container.Layout.Perspective': {
        property: 'perspective',
        valueFunction: (value) => value
    },
    'Container.Layout.BorderCollapse': {
        property: 'border-collapse',
        valueFunction: (value) => value
    },
    'Container.Layout.PageBreakBefore': {
        property: 'page-break-before',
        valueFunction: (value) => value
    },
    'Container.Layout.Columns': {
        property: 'columns',
        valueFunction: (value) => value
    },
    'Container.Layout.ColumnCount': {
        property: 'column-count',
        valueFunction: (value) => value
    },
    'Container.Layout.MinHeight': {
        property: 'min-height',
        valueFunction: (value) => value
    },
    'Container.Layout.PageBreakInside': {
        property: 'page-break-inside',
        valueFunction: (value) => value
    },
    'Container.Layout.ColumnGap': {
        property: 'column-gap',
        valueFunction: (value) => value
    },
    'Container.Layout.Clip': {
        property: 'clip',
        valueFunction: (value) => value
    },
    'Container.Layout.FlexDirection': {
        property: 'flex-direction',
        valueFunction: (value) => value
    },
    'Container.Layout.PageBreakAfter': {
        property: 'page-break-after',
        valueFunction: (value) => value
    },
    'Container.Layout.Top': {
        property: 'top',
        valueFunction: (value) => value
    },
    'Container.Layout.CounterIncrement': {
        property: 'counter-increment',
        valueFunction: (value) => value
    },
    'Container.Layout.Height': {
        property: 'height',
        valueFunction: (value) => value
    },
    'Container.Layout.TransformStyle': {
        property: 'transform-style',
        valueFunction: (value) => value
    },
    'Container.Layout.OverflowX': {
        property: 'overflow-x',
        valueFunction: (value) => value
    },
    'Container.Layout.FlexWrap': {
        property: 'flex-wrap',
        valueFunction: (value) => value
    },
    'Container.Layout.MaxWidth': {
        property: 'max-width',
        valueFunction: (value) => value
    },
    'Container.Layout.Bottom': {
        property: 'bottom',
        valueFunction: (value) => value
    },
    'Container.Layout.CounterReset': {
        property: 'counter-reset',
        valueFunction: (value) => value
    },
    'Container.Layout.Right': {
        property: 'right',
        valueFunction: (value) => value
    },
    'Container.Layout.BoxSizing': {
        property: 'box-sizing',
        valueFunction: (value) => value
    },
    'Container.Layout.Position': {
        property: 'position',
        valueFunction: (value) => value
    },
    'Container.Layout.TableLayout': {
        property: 'table-layout',
        valueFunction: (value) => value
    },
    'Container.Layout.Width': {
        property: 'width',
        valueFunction: (value) => value
    },
    'Container.Layout.MaxHeight': {
        property: 'max-height',
        valueFunction: (value) => value
    },
    'Container.Layout.ColumnWidth': {
        property: 'column-width',
        valueFunction: (value) => value
    },
    'Container.Layout.MinWidth': {
        property: 'min-width',
        valueFunction: (value) => value
    },
    'Content.Layout.VerticalAlign': {
        property: 'vertical-align',
        valueFunction: (value) => value
    },
    'Container.Layout.PerspectiveOrigin': {
        property: 'perspective-origin',
        valueFunction: (value) => value
    },
    'Container.Layout.AlignContent': {
        property: 'align-content',
        valueFunction: (value) => value
    },
    'Container.Layout.FlexFlow': {
        property: 'flex-flow',
        valueFunction: (value) => value
    },
    'Container.Layout.Display': {
        property: 'display',
        valueFunction: (value) => value
    },
    'Container.Layout.Left': {
        property: 'left',
        valueFunction: (value) => value
    },

    'Container.Style.BackgroundImage': {
        property: 'background-image',
        valueFunction: (value) => value
    },
    'Container.Style.BorderLeftStyle': {
        property: 'border-left-style',
        valueFunction: (value) => value
    },
    'Container.Style.BoxShadow': {
        property: 'box-shadow',
        valueFunction: (value) => value
    },
    'Container.Style.TransitionDelay': {
        property: 'transition-delay',
        valueFunction: (value) => value
    },
    'Container.Style.AnimationDuration': {
        property: 'animation-duration',
        valueFunction: (value) => value
    },
    'Content.Style.ListStyle': {
        property: 'list-style',
        valueFunction: (value) => value
    },
    'Container.Style.OutlineWidth': {
        property: 'outline-width',
        valueFunction: (value) => value
    },
    'Container.Style.BorderTopLeftRadius': {
        property: 'border-top-left-radius',
        valueFunction: (value) => value
    },
    'Content.Style.WhiteSpace': {
        property: 'white-space',
        valueFunction: (value) => value
    },
    'Container.Style.BorderRight': {
        property: 'border-right',
        valueFunction: (value) => value
    },
    'Content.Style.TextDecorationLine': {
        property: 'text-decoration-line',
        valueFunction: (value) => value
    },
    'Container.Style.AnimationDelay': {
        property: 'animation-delay',
        valueFunction: (value) => value
    },
    'Container.Style.BackgroundPosition': {
        property: 'background-position',
        valueFunction: (value) => value
    },
    'Container.Style.BorderImage': {
        property: 'border-image',
        valueFunction: (value) => value
    },
    'Container.Style.BorderSpacing': {
        property: 'border-spacing',
        valueFunction: (value) => value
    },
    'Container.Style.BorderImageOutset': {
        property: 'border-image-outset',
        valueFunction: (value) => value
    },
    'Container.Style.BorderImageSlice': {
        property: 'border-image-slice',
        valueFunction: (value) => value
    },
    'Container.Style.BorderLeftColor': {
        property: 'border-left-color',
        valueFunction: (value) => value
    },
    'Content.Style.FontSize': {
        property: 'font-size',
        valueFunction: (value) => value
    },
    'Content.Style.LineHeight': {
        property: 'line-height',
        valueFunction: (value) => value
    },
    'Content.Style.TextDecorationStyle': {
        property: 'text-decoration-style',
        valueFunction: (value) => value
    },
    'Container.Style.BackfaceVisibility': {
        property: 'backface-visibility',
        valueFunction: (value) => value
    },
    'Container.Style.BorderRightStyle': {
        property: 'border-right-style',
        valueFunction: (value) => value
    },
    'Content.Style.TextDecoration': {
        property: 'text-decoration',
        valueFunction: (value) => value
    },
    'Container.Style.Transition': {
        property: 'transition',
        valueFunction: (value) => value
    },
    'Container.Style.AnimationIterationCount': {
        property: 'animation-iteration-count',
        valueFunction: (value) => value
    },
    'Container.Style.BorderBottom': {
        property: 'border-bottom',
        valueFunction: (value) => value
    },
    'Container.Style.AnimationTimingFunction': {
        property: 'animation-timing-function',
        valueFunction: (value) => value
    },
    'Container.Style.BorderRadius': {
        property: 'border-radius',
        valueFunction: (value) => value
    },
    'Content.Style.Quotes': {
        property: 'quotes',
        valueFunction: (value) => value
    },
    'Container.Style.TabSize': {
        property: 'tab-size',
        valueFunction: (value) => value
    },
    'Container.Style.AnimationFillMode': {
        property: 'animation-fill-mode',
        valueFunction: (value) => value
    },
    'Container.Style.BackgroundSize': {
        property: 'background-size',
        valueFunction: (value) => value
    },
    'Content.Style.FontSizeAdjust': {
        property: 'font-size-adjust',
        valueFunction: (value) => value
    },
    'Content.Style.ListStylePosition': {
        property: 'list-style-position',
        valueFunction: (value) => value
    },
    'Content.Style.TextAlign': {
        property: 'text-align',
        valueFunction: (value) => value
    },
    'Content.Style.TextJustify': {
        property: 'text-justify',
        valueFunction: (value) => value
    },
    'Container.Style.BackgroundAttachment': {
        property: 'background-attachment',
        valueFunction: (value) => value
    },
    'Container.Style.BorderRightWidth': {
        property: 'border-right-width',
        valueFunction: (value) => value
    },
    'Content.Style.Font': {
        property: 'font',
        valueFunction: (value) => value
    },
    'Container.Style.BorderLeft': {
        property: 'border-left',
        valueFunction: (value) => value
    },
    'Container.Style.TransitionDuration': {
        property: 'transition-duration',
        valueFunction: (value) => value
    },
    'Content.Style.WordSpacing': {
        property: 'word-spacing',
        valueFunction: (value) => value
    },
    'Container.Style.AnimationName': {
        property: 'animation-name',
        valueFunction: (value) => value
    },
    'Container.Style.AnimationPlayState': {
        property: 'animation-play-state',
        valueFunction: (value) => value
    },
    'Content.Style.LetterSpacing': {
        property: 'letter-spacing',
        valueFunction: (value) => value
    },
    'Container.Style.BorderBottomStyle': {
        property: 'border-bottom-style',
        valueFunction: (value) => value
    },
    'Content.Style.WordBreak': {
        property: 'word-break',
        valueFunction: (value) => value
    },
    'Container.Style.BorderBottomRightRadius': {
        property: 'border-bottom-right-radius',
        valueFunction: (value) => value
    },
    'Content.Style.FontStyle': {
        property: 'font-style',
        valueFunction: (value) => value
    },
    'Container.Style.Order': {
        property: 'order',
        valueFunction: (value) => value
    },
    'Container.Style.OutlineStyle': {
        property: 'outline-style',
        valueFunction: (value) => value
    },
    'Container.Style.BorderBottomLeftRadius': {
        property: 'border-bottom-left-radius',
        valueFunction: (value) => value
    },
    'Container.Style.BorderImageSource': {
        property: 'border-image-source',
        valueFunction: (value) => value
    },
    'Content.Style.TextAlignLast': {
        property: 'text-align-last',
        valueFunction: (value) => value
    },
    'Container.Style.BorderImageWidth': {
        property: 'border-image-width',
        valueFunction: (value) => value
    },
    'Content.Style.FontWeight': {
        property: 'font-weight',
        valueFunction: (value) => value
    },
    'Content.Style.ListStyleImage': {
        property: 'list-style-image',
        valueFunction: (value) => value
    },
    'Container.Style.Opacity': {
        property: 'opacity',
        valueFunction: (value) => value
    },
    'Container.Style.Clear': {
        property: 'clear',
        valueFunction: (value) => value
    },
    'Container.Style.BorderTopColor': {
        property: 'border-top-color',
        valueFunction: (value) => value
    },
    'Container.Style.Border': {
        property: 'border',
        valueFunction: (value) => value
    },
    'Container.Style.BorderRightColor': {
        property: 'border-right-color',
        valueFunction: (value) => value
    },
    'Container.Style.TransitionTimingFunction': {
        property: 'transition-timing-function',
        valueFunction: (value) => value
    },
    'Container.Style.BorderBottomWidth': {
        property: 'border-bottom-width',
        valueFunction: (value) => value
    },
    'Container.Style.BorderStyle': {
        property: 'border-style',
        valueFunction: (value) => value
    },
    'Container.Style.BorderTopRightRadius': {
        property: 'border-top-right-radius',
        valueFunction: (value) => value
    },
    'Content.Style.CaptionSide': {
        property: 'caption-side',
        valueFunction: (value) => value
    },
    'Content.Style.FontFamily': {
        property: 'font-family',
        valueFunction: (value) => value
    },
    'Content.Style.TextDecorationColor': {
        property: 'text-decoration-color',
        valueFunction: (value) => value
    },
    'Container.Style.TransitionProperty': {
        property: 'transition-property',
        valueFunction: (value) => value
    },
    'Container.Style.BackgroundOrigin': {
        property: 'background-origin',
        valueFunction: (value) => value
    },
    'Content.Style.TextIndent': {
        property: 'text-indent',
        valueFunction: (value) => value
    },
    'Container.Style.Visibility': {
        property: 'visibility',
        valueFunction: (value) => value
    },
    'Container.Style.BorderColor': {
        property: 'border-color',
        valueFunction: (value) => value
    },
    'Container.Style.BorderTop': {
        property: 'border-top',
        valueFunction: (value) => value
    },
    'Content.Style.FontVariant': {
        property: 'font-variant',
        valueFunction: (value) => value
    },
    'Container.Style.Outline': {
        property: 'outline',
        valueFunction: (value) => value
    },
    'Container.Style.BorderBottomColor': {
        property: 'border-bottom-color',
        valueFunction: (value) => value
    },
    'Container.Style.BorderTopStyle': {
        property: 'border-top-style',
        valueFunction: (value) => value
    },
    'Container.Style.BorderWidth': {
        property: 'border-width',
        valueFunction: (value) => value
    },
    'Content.Style.ListStyleType': {
        property: 'list-style-type',
        valueFunction: (value) => value
    },
    'Container.Style.OutlineOffset': {
        property: 'outline-offset',
        valueFunction: (value) => value
    },
    'Container.Style.Animation': {
        property: 'animation',
        valueFunction: (value) => value
    },
    'Container.Style.Background': {
        property: 'background',
        valueFunction: (value) => value
    },
    'Container.Style.BackgroundRepeat': {
        property: 'background-repeat',
        valueFunction: (value) => value
    },
    'Container.Style.BorderTopWidth': {
        property: 'border-top-width',
        valueFunction: (value) => value
    },
    'Content.Style.WordWrap': {
        property: 'word-wrap',
        valueFunction: (value) => value
    },
    'Container.Style.BackgroundColor': {
        property: 'background-color',
        valueFunction: (value) => value
    },
    'Content.Style.TextOverflow': {
        property: 'text-overflow',
        valueFunction: (value) => value
    },
    'Content.Style.TextShadow': {
        property: 'text-shadow',
        valueFunction: (value) => value
    },
    'Container.Style.BackgroundClip': {
        property: 'background-clip',
        valueFunction: (value) => value
    },
    'Container.Style.BorderLeftWidth': {
        property: 'border-left-width',
        valueFunction: (value) => value
    },
    'Container.Style.Resize': {
        property: 'resize',
        valueFunction: (value) => value
    },
    'Container.Style.AnimationDirection': {
        property: 'animation-direction',
        valueFunction: (value) => value
    },
    'Content.Style.Color': {
        property: 'color',
        valueFunction: (value) => value
    },
    'Container.Style.OutlineColor': {
        property: 'outline-color',
        valueFunction: (value) => value
    },
    'Container.Style.BorderImageRepeat': {
        property: 'border-image-repeat',
        valueFunction: (value) => value
    },
    'Content.Style.FontStretch': {
        property: 'font-stretch',
        valueFunction: (value) => value
    },
    'Content.Style.TextTransform': {
        property: 'text-transform',
        valueFunction: (value) => value
    },

    'Content.Layout.FlexGrow': {
        property: 'flex-grow',
        valueFunction: (value) => value
    },
    'Content.Layout.AlignSelf': {
        property: 'align-self',
        valueFunction: (value) => value
    },
    'Content.Layout.Content': {
        property: 'content',
        valueFunction: (value) => value
    },
    'Content.Layout.ColumnSpan': {
        property: 'column-span',
        valueFunction: (value) => value
    },
    'Content.Layout.Flex': {
        property: 'flex',
        valueFunction: (value) => value
    },
    'Content.Layout.FlexShrink': {
        property: 'flex-shrink',
        valueFunction: (value) => value
    },
    'Content.Layout.Order': {
        property: 'order',
        valueFunction: (value) => value
    },
    'Content.Layout.FlexBasis': {
        property: 'flex-basis',
        valueFunction: (value) => value
    },
    'Content.Layout.AlignItems': {
        property: 'align-items',
        valueFunction: (value) => value
    },

    'Content.Style.ColumnRuleWidth': {
        property: 'column-rule-width',
        valueFunction: (value) => value
    },
    'Content.Style.ColumnRule': {
        property: 'column-rule',
        valueFunction: (value) => value
    },
    'Content.Style.Direction': {
        property: 'direction',
        valueFunction: (value) => value
    },
    'Content.Style.ColumnRuleStyle': {
        property: 'column-rule-style',
        valueFunction: (value) => value
    },
    'Content.Style.ColumnRuleColor': {
        property: 'column-rule-color',
        valueFunction: (value) => value
    },
    'Content.Style.ColumnFill': {
        property: 'column-fill',
        valueFunction: (value) => value
    },
    'Content.Style.EmptyCells': {
        property: 'empty-cells',
        valueFunction: (value) => value
    },
    'Content.Style.Cursor': {
        property: 'cursor',
        valueFunction: (value) => value
    },
    'Container.Style.Transform': {
        property: 'transform',
        valueFunction: (value) => value
    },
    'Container.Style.PointerEvents': {
        property: 'pointer-events',
        valueFunction: (value) => value
    },
    'Container.Style.UserSelect': {
        property: 'user-select',
        valueFunction: (value) => value
    },
    'Container.Style.BackdropFilter': {
        property: 'backdrop-filter',
        valueFunction: (value) => value
    },
    'Container.Style.ObjectFit': {
        property: 'object-fit',
        valueFunction: (value) => value
    },
    'Container.Style.ObjectPosition': {
        property: 'object-position',
        valueFunction: (value) => value
    },
    'Container.Layout.GridTemplateColumns': {
        property: 'grid-template-columns',
        valueFunction: (value) => value
    },
    'Container.Layout.GridTemplateRows': {
        property: 'grid-template-rows',
        valueFunction: (value) => value
    },
    'Container.Layout.GridColumn': {
        property: 'grid-column',
        valueFunction: (value) => value
    },
    'Container.Layout.GridRow': {
        property: 'grid-row',
        valueFunction: (value) => value
    },
    'Container.Layout.Gap': {
        property: 'gap',
        valueFunction: (value) => value
    },
    'Container.Layout.ScrollBehavior': {
        property: 'scroll-behavior',
        valueFunction: (value) => value
    },
    'Container.Style.Margin': {
        property: 'margin',
        valueFunction: (value) => value
    },
    'Container.Style.MarginTop': {
        property: 'margin-top',
        valueFunction: (value) => value
    },
    'Container.Style.MarginRight': {
        property: 'margin-right',
        valueFunction: (value) => value
    },
    'Container.Style.MarginBottom': {
        property: 'margin-bottom',
        valueFunction: (value) => value
    },
    'Container.Style.MarginLeft': {
        property: 'margin-left',
        valueFunction: (value) => value
    },
    'Container.Style.Padding': {
        property: 'padding',
        valueFunction: (value) => value
    },
    'Container.Style.PaddingTop': {
        property: 'padding-top',
        valueFunction: (value) => value
    },
    'Container.Style.PaddingRight': {
        property: 'padding-right',
        valueFunction: (value) => value
    },
    'Container.Style.PaddingBottom': {
        property: 'padding-bottom',
        valueFunction: (value) => value
    },
    'Container.Style.PaddingLeft': {
        property: 'padding-left',
        valueFunction: (value) => value
    },
    'Container.Style.Transform': {
        property: 'transform',
        valueFunction: (value) => value
    },
    'Container.Style.PointerEvents': {
        property: 'pointer-events',
        valueFunction: (value) => value
    },
    'Container.Style.UserSelect': {
        property: 'user-select',
        valueFunction: (value) => value
    },
    'Container.Layout.GridTemplateColumns': {
        property: 'grid-template-columns',
        valueFunction: (value) => value
    },
    'Container.Layout.GridTemplateRows': {
        property: 'grid-template-rows',
        valueFunction: (value) => value
    },
    'Container.Layout.GridColumn': {
        property: 'grid-column',
        valueFunction: (value) => value
    },
    'Container.Layout.GridRow': {
        property: 'grid-row',
        valueFunction: (value) => value
    },
    'Container.Layout.Gap': {
        property: 'gap',
        valueFunction: (value) => value
    },
    'Container.Style.BackdropFilter': {
        property: 'backdrop-filter',
        valueFunction: (value) => value
    },
    'Container.Layout.ScrollBehavior': {
        property: 'scroll-behavior',
        valueFunction: (value) => value
    },
    'Container.Style.ObjectFit': {
        property: 'object-fit',
        valueFunction: (value) => value
    },
    'Container.Style.ObjectPosition': {
        property: 'object-position',
        valueFunction: (value) => value
    },

};

const cssToGoMapping = {
    // Container Layout properties
    "box-shadow": {
        id: "Container.Layout.BoxShadow",
        values: ["none"],
    },
    "justify-content": {
        id: "Container.Layout.JustifyContent",
        values: ["flex-start", "flex-end", "center", "space-between", "space-around"],
    },
    "z-index": {
        id: "Container.Layout.ZIndex",
        values: ["auto"],
    },
    "float": {
        id: "Container.Layout.Float",
        values: ["none", "left", "right"],
    },
    "overflow": {
        id: "Container.Layout.Overflow",
        values: ["visible", "hidden", "scroll", "auto"],
    },
    "overflow-y": {
        id: "Container.Layout.OverflowY",
        values: ["visible", "hidden", "scroll", "auto"],
    },
    "perspective": {
        id: "Container.Layout.Perspective",
        values: ["none"],
    },
    "border-collapse": {
        id: "Container.Layout.BorderCollapse",
        values: ["separate", "collapse"],
    },
    "page-break-before": {
        id: "Container.Layout.PageBreakBefore",
        values: ["auto", "always", "avoid", "left", "right"],
    },
    "columns": {
        id: "Container.Layout.Columns",
        values: ["auto"],
    },
    "column-count": {
        id: "Container.Layout.ColumnCount",
        values: ["auto"],
    },
    "min-height": {
        id: "Container.Layout.MinHeight",
        values: [],
    },
    "page-break-inside": {
        id: "Container.Layout.PageBreakInside",
        values: ["auto", "avoid"],
    },
    "column-gap": {
        id: "Container.Layout.ColumnGap",
        values: ["normal"],
    },
    "clip": {
        id: "Container.Layout.Clip",
        values: ["auto"],
    },
    "flex-direction": {
        id: "Container.Layout.FlexDirection",
        values: ["row", "row-reverse", "column", "column-reverse"],
    },
    "page-break-after": {
        id: "Container.Layout.PageBreakAfter",
        values: ["auto", "always", "avoid", "left", "right"],
    },
    "top": {
        id: "Container.Layout.Top",
        values: ["auto"],
    },
    "counter-increment": {
        id: "Container.Layout.CounterIncrement",
        values: ["none"],
    },
    "height": {
        id: "Container.Layout.Height",
        values: ["auto"],
    },
    "transform-style": {
        id: "Container.Layout.TransformStyle",
        values: ["flat", "preserve-3d"],
    },
    "overflow-x": {
        id: "Container.Layout.OverflowX",
        values: ["visible", "hidden", "scroll", "auto"],
    },
    "flex-wrap": {
        id: "Container.Layout.FlexWrap",
        values: ["nowrap", "wrap", "wrap-reverse"],
    },
    "max-width": {
        id: "Container.Layout.MaxWidth",
        values: ["none"],
    },
    "bottom": {
        id: "Container.Layout.Bottom",
        values: ["auto"],
    },
    "counter-reset": {
        id: "Container.Layout.CounterReset",
        values: ["none"],
    },
    "right": {
        id: "Container.Layout.Right",
        values: ["auto"],
    },
    "box-sizing": {
        id: "Container.Layout.BoxSizing",
        values: ["content-box", "border-box"],
    },
    "position": {
        id: "Container.Layout.Position",
        values: ["static", "absolute", "fixed", "relative", "sticky"],
    },
    "table-layout": {
        id: "Container.Layout.TableLayout",
        values: ["auto", "fixed"],
    },
    "width": {
        id: "Container.Layout.Width",
        values: ["auto"],
    },
    "max-height": {
        id: "Container.Layout.MaxHeight",
        values: ["none"],
    },
    "column-width": {
        id: "Container.Layout.ColumnWidth",
        values: ["auto"],
    },
    "min-width": {
        id: "Container.Layout.MinWidth",
        values: [],
    },
    "vertical-align": {
        id: "Content.Layout.VerticalAlign",
        values: ["baseline", "top", "text-top", "middle", "bottom", "text-bottom"],
    },
    "perspective-origin": {
        id: "Container.Layout.PerspectiveOrigin",
        values: [],
    },
    "align-content": {
        id: "Container.Layout.AlignContent",
        values: ["stretch", "center", "flex-start", "flex-end", "space-between", "space-around"],
    },
    "flex-flow": {
        id: "Container.Layout.FlexFlow",
        values: [],
    },
    "display": {
        id: "Container.Layout.Display",
        values: ["inline", "block", "contents", "flex", "grid", "none"],
    },
    "left": {
        id: "Container.Layout.Left",
        values: ["auto"],
    },
    "grid-template-columns": {
        id: "Container.Layout.GridTemplateColumns",
        values: [],
    },
    "grid-template-rows": {
        id: "Container.Layout.GridTemplateRows",
        values: [],
    },
    "grid-column": {
        id: "Container.Layout.GridColumn",
        values: [],
    },
    "grid-row": {
        id: "Container.Layout.GridRow",
        values: [],
    },
    "gap": {
        id: "Container.Layout.Gap",
        values: [],
    },
    "scroll-behavior": {
        id: "Container.Layout.ScrollBehavior",
        values: ["auto", "smooth"],
    },

    // Container Style properties
    "background-image": {
        id: "Container.Style.BackgroundImage",
        values: ["none"],
    },
    "border-left-style": {
        id: "Container.Style.BorderLeftStyle",
        values: ["none", "hidden", "dotted", "dashed", "solid", "double", "groove", "ridge", "inset", "outset"],
    },
    "transition-delay": {
        id: "Container.Style.TransitionDelay",
        values: [],
    },
    "animation-duration": {
        id: "Container.Style.AnimationDuration",
        values: [],
    },
    "list-style": {
        id: "Content.Style.ListStyle",
        values: [],
    },
    "outline-width": {
        id: "Container.Style.OutlineWidth",
        values: ["medium", "thin", "thick"],
    },
    "border-top-left-radius": {
        id: "Container.Style.BorderTopLeftRadius",
        values: [],
    },
    "white-space": {
        id: "Content.Style.WhiteSpace",
        values: ["normal", "nowrap", "pre", "pre-line", "pre-wrap"],
    },
    "border-right": {
        id: "Container.Style.BorderRight",
        values: [],
    },
    "text-decoration-line": {
        id: "Content.Style.TextDecorationLine",
        values: ["none", "underline", "overline", "line-through"],
    },
    "animation-delay": {
        id: "Container.Style.AnimationDelay",
        values: [],
    },
    "background-position": {
        id: "Container.Style.BackgroundPosition",
        values: ["left top", "left center", "left bottom", "right top", "right center", "right bottom", "center top", "center center", "center bottom"],
    },
    "border-image": {
        id: "Container.Style.BorderImage",
        values: [],
    },
    "border-spacing": {
        id: "Container.Style.BorderSpacing",
        values: [],
    },
    "border-image-outset": {
        id: "Container.Style.BorderImageOutset",
        values: [],
    },
    "border-image-slice": {
        id: "Container.Style.BorderImageSlice",
        values: [],
    },
    "border-left-color": {
        id: "Container.Style.BorderLeftColor",
        values: ["transparent"],
    },
    "font-size": {
        id: "Content.Style.FontSize",
        values: ["medium", "xx-small", "x-small", "small", "large", "x-large", "xx-large", "smaller", "larger"],
    },
    "line-height": {
        id: "Content.Style.LineHeight",
        values: ["normal"],
    },
    "text-decoration-style": {
        id: "Content.Style.TextDecorationStyle",
        values: ["solid", "double", "dotted", "dashed", "wavy"],
    },
    "backface-visibility": {
        id: "Container.Style.BackfaceVisibility",
        values: ["visible", "hidden"],
    },
    "border-right-style": {
        id: "Container.Style.BorderRightStyle",
        values: ["none", "hidden", "dotted", "dashed", "solid", "double", "groove", "ridge", "inset", "outset"],
    },
    "text-decoration": {
        id: "Content.Style.TextDecoration",
        values: [],
    },
    "transition": {
        id: "Container.Style.Transition",
        values: [],
    },
    "animation-iteration-count": {
        id: "Container.Style.AnimationIterationCount",
        values: ["infinite"],
    },
    "border-bottom": {
        id: "Container.Style.BorderBottom",
        values: [],
    },
    "animation-timing-function": {
        id: "Container.Style.AnimationTimingFunction",
        values: [],
    },
    "border-radius": {
        id: "Container.Style.BorderRadius",
        values: [],
    },
    "quotes": {
        id: "Content.Style.Quotes",
        values: ["none", "auto"],
    },
    "tab-size": {
        id: "Container.Style.TabSize",
        values: [],
    },
    "animation-fill-mode": {
        id: "Container.Style.AnimationFillMode",
        values: ["none", "forwards", "backwards", "both"],
    },
    "background-size": {
        id: "Container.Style.BackgroundSize",
        values: ["auto", "cover", "contain"],
    },
    "font-size-adjust": {
        id: "Content.Style.FontSizeAdjust",
        values: ["none"],
    },
    "list-style-position": {
        id: "Content.Style.ListStylePosition",
        values: ["inside", "outside"],
    },
    "text-align": {
        id: "Content.Style.TextAlign",
        values: ["left", "right", "center", "justify"],
    },
    "text-justify": {
        id: "Content.Style.TextJustify",
        values: ["auto", "inter-word", "inter-character", "none"],
    },
    "background-attachment": {
        id: "Container.Style.BackgroundAttachment",
        values: ["scroll", "fixed", "local"],
    },
    "border-right-width": {
        id: "Container.Style.BorderRightWidth",
        values: ["medium", "thin", "thick"],
    },
    "font": {
        id: "Content.Style.Font",
        values: [],
    },
    "border-left": {
        id: "Container.Style.BorderLeft",
        values: [],
    },
    "transition-duration": {
        id: "Container.Style.TransitionDuration",
        values: [],
    },
    "word-spacing": {
        id: "Content.Style.WordSpacing",
        values: ["normal"],
    },
    "animation-name": {
        id: "Container.Style.AnimationName",
        values: ["none"],
    },
    "animation-play-state": {
        id: "Container.Style.AnimationPlayState",
        values: ["paused", "running"],
    },
    "letter-spacing": {
        id: "Content.Style.LetterSpacing",
        values: ["normal"],
    },
    "border-bottom-style": {
        id: "Container.Style.BorderBottomStyle",
        values: ["none", "hidden", "dotted", "dashed", "solid", "double", "groove", "ridge", "inset", "outset"],
    },
    "word-break": {
        id: "Content.Style.WordBreak",
        values: ["normal", "break-all", "keep-all", "break-word"],
    },
    "border-bottom-right-radius": {
        id: "Container.Style.BorderBottomRightRadius",
        values: [],
    },
    "font-style": {
        id: "Content.Style.FontStyle",
        values: ["normal", "italic", "oblique"],
    },
    "order": {
        id: "Container.Style.Order",
        values: [],
    },
    "outline-style": {
        id: "Container.Style.OutlineStyle",
        values: ["none", "hidden", "dotted", "dashed", "solid", "double", "groove", "ridge", "inset", "outset"],
    },
    "border-bottom-left-radius": {
        id: "Container.Style.BorderBottomLeftRadius",
        values: [],
    },
    "border-image-source": {
        id: "Container.Style.BorderImageSource",
        values: ["none"],
    },
    "text-align-last": {
        id: "Content.Style.TextAlignLast",
        values: ["auto", "left", "right", "center", "justify", "start", "end"],
    },
    "border-image-width": {
        id: "Container.Style.BorderImageWidth",
        values: ["auto"],
    },
    "font-weight": {
        id: "Content.Style.FontWeight",
        values: ["normal", "bold", "bolder", "lighter", "100", "200", "300", "400", "500", "600", "700", "800", "900"],
    },
    "list-style-image": {
        id: "Content.Style.ListStyleImage",
        values: ["none"],
    },
    "opacity": {
        id: "Container.Style.Opacity",
        values: [],
    },
    "clear": {
        id: "Container.Style.Clear",
        values: ["none", "left", "right", "both"],
    },
    "border-top-color": {
        id: "Container.Style.BorderTopColor",
        values: ["transparent"],
    },
    "border": {
        id: "Container.Style.Border",
        values: [],
    },
    "border-right-color": {
        id: "Container.Style.BorderRightColor",
        values: ["transparent"],
    },
    "transition-timing-function": {
        id: "Container.Style.TransitionTimingFunction",
        values: [],
    },
    "border-bottom-width": {
        id: "Container.Style.BorderBottomWidth",
        values: ["medium", "thin", "thick"],
    },
    "border-style": {
        id: "Container.Style.BorderStyle",
        values: ["none", "hidden", "dotted", "dashed", "solid", "double", "groove", "ridge", "inset", "outset"],
    },
    "border-top-right-radius": {
        id: "Container.Style.BorderTopRightRadius",
        values: [],
    },
    "caption-side": {
        id: "Content.Style.CaptionSide",
        values: ["top", "bottom"],
    },
    "font-family": {
        id: "Content.Style.FontFamily",
        values: [],
    },
    "text-decoration-color": {
        id: "Content.Style.TextDecorationColor",
        values: [],
    },
    "transition-property": {
        id: "Container.Style.TransitionProperty",
        values: ["none"],
    },
    "background-origin": {
        id: "Container.Style.BackgroundOrigin",
        values: ["padding-box", "border-box", "content-box"],
    },
    "text-indent": {
        id: "Content.Style.TextIndent",
        values: [],
    },
    "visibility": {
        id: "Container.Style.Visibility",
        values: ["visible", "hidden", "collapse"],
    },
    "border-color": {
        id: "Container.Style.BorderColor",
        values: ["transparent"],
    },
    "border-top": {
        id: "Container.Style.BorderTop",
        values: [],
    },
    "font-variant": {
        id: "Content.Style.FontVariant",
        values: ["normal", "small-caps"],
    },
    "outline": {
        id: "Container.Style.Outline",
        values: [],
    },
    "border-bottom-color": {
        id: "Container.Style.BorderBottomColor",
        values: ["transparent"],
    },
    "border-top-style": {
        id: "Container.Style.BorderTopStyle",
        values: ["none", "hidden", "dotted", "dashed", "solid", "double", "groove", "ridge", "inset", "outset"],
    },
    "border-width": {
        id: "Container.Style.BorderWidth",
        values: ["medium", "thin", "thick"],
    },
    "list-style-type": {
        id: "Content.Style.ListStyleType",
        values: ["disc", "armenian", "circle", "cjk-ideographic", "decimal", "decimal-leading-zero", "georgian", "hebrew", "hiragana", "hiragana-iroha", "katakana", "katakana-iroha", "lower-alpha", "lower-greek", "lower-latin", "lower-roman", "none", "square", "upper-alpha", "upper-greek", "upper-latin", "upper-roman"],
    },
    "outline-offset": {
        id: "Container.Style.OutlineOffset",
        values: [],
    },
    "animation": {
        id: "Container.Style.Animation",
        values: [],
    },
    "background": {
        id: "Container.Style.Background",
        values: [],
    },
    "background-repeat": {
        id: "Container.Style.BackgroundRepeat",
        values: ["repeat", "repeat-x", "repeat-y", "no-repeat", "space", "round"],
    },
    "border-top-width": {
        id: "Container.Style.BorderTopWidth",
        values: ["medium", "thin", "thick"],
    },
    "word-wrap": {
        id: "Content.Style.WordWrap",
        values: ["normal", "break-word"],
    },
    "background-color": {
        id: "Container.Style.BackgroundColor",
        values: ["transparent", "aliceblue", "antiquewhite", "aqua", "aquamarine", "azure",
            "beige", "bisque", "black", "blanchedalmond", "blue", "blueviolet", "brown",
            "burlywood", "cadetblue", "chartreuse", "chocolate", "coral", "cornflowerblue",
            "cornsilk", "crimson", "cyan", "darkblue", "darkcyan", "darkgoldenrod",
            "darkgray", "darkgreen", "darkkhaki", "darkmagenta", "darkolivegreen",
            "darkorange", "darkorchid", "darkred", "darksalmon", "darkseagreen",
            "darkslateblue", "darkslategray", "darkturquoise", "darkviolet", "deeppink",
            "deepskyblue", "dimgray", "dodgerblue", "firebrick", "floralwhite",
            "forestgreen", "fuchsia", "gainsboro", "ghostwhite", "gold", "goldenrod",
            "gray", "green", "greenyellow", "honeydew", "hotpink", "indianred", "indigo",
            "ivory", "khaki", "lavender", "lavenderblush", "lawngreen", "lemonchiffon",
            "lightblue", "lightcoral", "lightcyan", "lightgoldenrodyellow", "lightgray",
            "lightgreen", "lightpink", "lightsalmon", "lightseagreen", "lightskyblue",
            "lightslategray", "lightsteelblue", "lightyellow", "lime", "limegreen",
            "linen", "magenta", "maroon", "mediumaquamarine", "mediumblue", "mediumorchid",
            "mediumpurple", "mediumseagreen", "mediumslateblue", "mediumspringgreen",
            "mediumturquoise", "mediumvioletred", "midnightblue", "mintcream", "mistyrose",
            "moccasin", "navajowhite", "navy", "oldlace", "olive", "olivedrab", "orange",
            "orangered", "orchid", "palegoldenrod", "palegreen", "paleturquoise",
            "palevioletred", "papayawhip", "peachpuff", "peru", "pink", "plum", "powderblue",
            "purple", "rebeccapurple", "red", "rosybrown", "royalblue", "saddlebrown",
            "salmon", "sandybrown", "seagreen", "seashell", "sienna", "silver", "skyblue",
            "slateblue", "slategray", "snow", "springgreen", "steelblue", "tan", "teal",
            "thistle", "tomato", "turquoise", "violet", "wheat", "white", "whitesmoke",
            "yellow", "yellowgreen"],
    },
    "text-overflow": {
        id: "Content.Style.TextOverflow",
        values: ["clip", "ellipsis"],
    },
    "text-shadow": {
        id: "Content.Style.TextShadow",
        values: ["none"],
    },
    "background-clip": {
        id: "Container.Style.BackgroundClip",
        values: ["border-box", "padding-box", "content-box"],
    },
    "border-left-width": {
        id: "Container.Style.BorderLeftWidth",
        values: ["medium", "thin", "thick"],
    },
    "resize": {
        id: "Container.Style.Resize",
        values: ["none", "both", "horizontal", "vertical"],
    },
    "animation-direction": {
        id: "Container.Style.AnimationDirection",
        values: ["normal", "reverse", "alternate", "alternate-reverse"],
    },
    "color": {
        id: "Content.Style.Color",
        values: ["transparent", "aliceblue", "antiquewhite", "aqua", "aquamarine", "azure",
            "beige", "bisque", "black", "blanchedalmond", "blue", "blueviolet", "brown",
            "burlywood", "cadetblue", "chartreuse", "chocolate", "coral", "cornflowerblue",
            "cornsilk", "crimson", "cyan", "darkblue", "darkcyan", "darkgoldenrod",
            "darkgray", "darkgreen", "darkkhaki", "darkmagenta", "darkolivegreen",
            "darkorange", "darkorchid", "darkred", "darksalmon", "darkseagreen",
            "darkslateblue", "darkslategray", "darkturquoise", "darkviolet", "deeppink",
            "deepskyblue", "dimgray", "dodgerblue", "firebrick", "floralwhite",
            "forestgreen", "fuchsia", "gainsboro", "ghostwhite", "gold", "goldenrod",
            "gray", "green", "greenyellow", "honeydew", "hotpink", "indianred", "indigo",
            "ivory", "khaki", "lavender", "lavenderblush", "lawngreen", "lemonchiffon",
            "lightblue", "lightcoral", "lightcyan", "lightgoldenrodyellow", "lightgray",
            "lightgreen", "lightpink", "lightsalmon", "lightseagreen", "lightskyblue",
            "lightslategray", "lightsteelblue", "lightyellow", "lime", "limegreen",
            "linen", "magenta", "maroon", "mediumaquamarine", "mediumblue", "mediumorchid",
            "mediumpurple", "mediumseagreen", "mediumslateblue", "mediumspringgreen",
            "mediumturquoise", "mediumvioletred", "midnightblue", "mintcream", "mistyrose",
            "moccasin", "navajowhite", "navy", "oldlace", "olive", "olivedrab", "orange",
            "orangered", "orchid", "palegoldenrod", "palegreen", "paleturquoise",
            "palevioletred", "papayawhip", "peachpuff", "peru", "pink", "plum", "powderblue",
            "purple", "rebeccapurple", "red", "rosybrown", "royalblue", "saddlebrown",
            "salmon", "sandybrown", "seagreen", "seashell", "sienna", "silver", "skyblue",
            "slateblue", "slategray", "snow", "springgreen", "steelblue", "tan", "teal",
            "thistle", "tomato", "turquoise", "violet", "wheat", "white", "whitesmoke",
            "yellow", "yellowgreen"],
    },
    "outline-color": {
        id: "Container.Style.OutlineColor",
        values: ["invert"],
    },
    "border-image-repeat": {
        id: "Container.Style.BorderImageRepeat",
        values: ["stretch", "repeat", "round", "space"],
    },
    "font-stretch": {
        id: "Content.Style.FontStretch",
        values: ["ultra-condensed", "extra-condensed", "condensed", "semi-condensed", "normal", "semi-expanded", "expanded", "extra-expanded", "ultra-expanded"],
    },
    "text-transform": {
        id: "Content.Style.TextTransform",
        values: ["none", "capitalize", "uppercase", "lowercase"],
    },
    "transform": {
        id: "Container.Style.Transform",
        values: [],
    },
    "pointer-events": {
        id: "Container.Style.PointerEvents",
        values: ["none", "auto"],
    },
    "user-select": {
        id: "Container.Style.UserSelect",
        values: ["none", "text", "auto", "all"],
    },
    "backdrop-filter": {
        id: "Container.Style.BackdropFilter",
        values: [],
    },
    "object-fit": {
        id: "Container.Style.ObjectFit",
        values: ["fill", "contain", "cover", "none", "scale-down"],
    },
    "object-position": {
        id: "Container.Style.ObjectPosition",
        values: [],
    },

    // Content Layout properties
    "flex-grow": {
        id: "Content.Layout.FlexGrow",
        values: [],
    },
    "align-self": {
        id: "Content.Layout.AlignSelf",
        values: ["auto", "stretch", "center", "flex-start", "flex-end", "baseline"],
    },
    "content": {
        id: "Content.Layout.Content",
        values: ["normal", "none", "counter", "open-quote", "close-quote", "no-open-quote", "no-close-quote"],
    },
    "column-span": {
        id: "Content.Layout.ColumnSpan",
        values: ["none", "all"],
    },
    "flex": {
        id: "Content.Layout.Flex",
        values: ["auto", "initial", "none", "inherit"],
    },
    "flex-shrink": {
        id: "Content.Layout.FlexShrink",
        values: [],
    },
    "flex-basis": {
        id: "Content.Layout.FlexBasis",
        values: ["auto"],
    },
    "align-items": {
        id: "Content.Layout.AlignItems",
        values: ["stretch", "center", "flex-start", "flex-end", "baseline"],
    },

    // Content Style properties
    "column-rule-width": {
        id: "Content.Style.ColumnRuleWidth",
        values: ["medium", "thin", "thick"],
    },
    "column-rule": {
        id: "Content.Style.ColumnRule",
        values: [],
    },
    "direction": {
        id: "Content.Style.Direction",
        values: ["ltr", "rtl"],
    },
    "column-rule-style": {
        id: "Content.Style.ColumnRuleStyle",
        values: ["none", "hidden", "dotted", "dashed", "solid", "double", "groove", "ridge", "inset", "outset"],
    },
    "column-rule-color": {
        id: "Content.Style.ColumnRuleColor",
        values: [],
    },
    "column-fill": {
        id: "Content.Style.ColumnFill",
        values: ["balance", "auto"],
    },
    "empty-cells": {
        id: "Content.Style.EmptyCells",
        values: ["show", "hide"],
    },
    "cursor": {
        id: "Content.Style.Cursor",
        values: ["alias", "all-scroll", "auto", "cell", "context-menu", "col-resize", "copy", "crosshair", "default", "e-resize", "ew-resize", "grab", "grabbing", "help", "move", "n-resize", "ne-resize", "nesw-resize", "ns-resize", "nw-resize", "nwse-resize", "no-drop", "none", "not-allowed", "pointer", "progress", "row-resize", "s-resize", "se-resize", "sw-resize", "text", "vertical-text", "w-resize", "wait", "zoom-in", "zoom-out"],
    },
    "padding": {
        id: "Container.Layout.Padding",
        values: [],
    },
    "padding-top": {
        id: "Container.Layout.PaddingTop",
        values: [],
    },
    "padding-right": {
        id: "Container.Layout.PaddingRight",
        values: [],
    },
    "padding-bottom": {
        id: "Container.Layout.PaddingBottom",
        values: [],
    },
    "padding-left": {
        id: "Container.Layout.PaddingLeft",
        values: [],
    },

    // Margin properties
    "margin": {
        id: "Container.Layout.Margin",
        values: ["auto"],
    },
    "margin-top": {
        id: "Container.Layout.MarginTop",
        values: ["auto"],
    },
    "margin-right": {
        id: "Container.Layout.MarginRight",
        values: ["auto"],
    },
    "margin-bottom": {
        id: "Container.Layout.MarginBottom",
        values: ["auto"],
    },
    "margin-left": {
        id: "Container.Layout.MarginLeft",
        values: ["auto"],
    },

    // Text-related properties
    "text-orientation": {
        id: "Content.Style.TextOrientation",
        values: ["mixed", "upright", "sideways"],
    },
    "text-underline-position": {
        id: "Content.Style.TextUnderlinePosition",
        values: ["auto", "under", "left", "right"],
    },
    "text-rendering": {
        id: "Content.Style.TextRendering",
        values: ["auto", "optimizeSpeed", "optimizeLegibility", "geometricPrecision"],
    },
    "font-kerning": {
        id: "Content.Style.FontKerning",
        values: ["auto", "normal", "none"],
    },
    "hanging-punctuation": {
        id: "Container.Style.HangingPunctuation",
        values: ["none", "first", "last", "force-end", "allow-end"],
    },

    // Flexbox-specific properties
    "place-content": {
        id: "Container.Layout.PlaceContent",
        values: [],
    },
    "place-items": {
        id: "Container.Layout.PlaceItems",
        values: [],
    },
    "place-self": {
        id: "Content.Layout.PlaceSelf",
        values: [],
    },

    // Grid-specific properties
    "grid-template-areas": {
        id: "Container.Layout.GridTemplateAreas",
        values: ["none"],
    },
    "grid-auto-columns": {
        id: "Container.Layout.GridAutoColumns",
        values: ["auto"],
    },
    "grid-auto-rows": {
        id: "Container.Layout.GridAutoRows",
        values: ["auto"],
    },
    "grid-auto-flow": {
        id: "Container.Layout.GridAutoFlow",
        values: ["row", "column", "dense", "row dense", "column dense"],
    },
    "grid-area": {
        id: "Content.Layout.GridArea",
        values: ["auto"],
    },

    // Transform-related properties
    "transform-origin": {
        id: "Container.Style.TransformOrigin",
        values: [],
    },
    "transform-box": {
        id: "Container.Style.TransformBox",
        values: ["content-box", "border-box", "fill-box", "stroke-box", "view-box"],
    },

    // Miscellaneous properties
    "aspect-ratio": {
        id: "Container.Layout.AspectRatio",
        values: ["auto"],
    },
    "scroll-snap-type": {
        id: "Container.Layout.ScrollSnapType",
        values: ["none", "x mandatory", "y mandatory", "block mandatory", "inline mandatory", "both mandatory"],
    },
    "overscroll-behavior": {
        id: "Container.Layout.OverscrollBehavior",
        values: ["auto", "contain", "none"],
    },
    "contain": {
        id: "Container.Layout.Contain",
        values: ["none", "strict", "content", "size", "layout", "style", "paint"],
    },
    "will-change": {
        id: "Container.Style.WillChange",
        values: ["auto"],
    },
    "filter": {
        id: "Container.Style.Filter",
        values: ["none"],
    },
    "mix-blend-mode": {
        id: "Container.Style.MixBlendMode",
        values: ["normal", "multiply", "screen", "overlay", "darken", "lighten", "color-dodge", "color-burn", "hard-light", "soft-light", "difference", "exclusion", "hue", "saturation", "color", "luminosity"],
    },
    "isolation": {
        id: "Container.Style.Isolation",
        values: ["auto", "isolate"],
    },

    // Newer CSS properties
    "accent-color": {
        id: "Container.Style.AccentColor",
        values: ["auto"],
    },
    "content-visibility": {
        id: "Container.Layout.ContentVisibility",
        values: ["visible", "auto", "hidden"],
    },
    "scrollbar-color": {
        id: "Container.Style.ScrollbarColor",
        values: ["auto"],
    },
    "scrollbar-width": {
        id: "Container.Style.ScrollbarWidth",
        values: ["auto", "thin", "none"],
    },

};



function cssToGoCode(css, elementId) {
    const lines = css.split('\n');
    let goCode = `func(e ui.Element) ui.Element {\n`;
    let currentPseudoClass = '';

    for (let line of lines) {
        line = line.trim();
        if (line.endsWith('{')) {
            // Check for pseudo-class
            const pseudoMatch = line.match(/:(\w+)/);
            if (pseudoMatch) {
                currentPseudoClass = pseudoMatch[1].charAt(0).toUpperCase() + pseudoMatch[1].slice(1);
            }
            continue;
        }
        if (line === '}') {
            currentPseudoClass = '';
            continue;
        }
        if (line === '') continue;

        if (line.includes(':')) {
            const [property, value] = line.split(':').map(s => s.trim());
            const cleanValue = value.replace(';', '');

            if (cssToGoMapping[property]) {
                const goChain = cssToGoMapping[property];
                let [namespace, category, method] = goChain.split('.');

                // Handle different value types
                let goValue = cleanValue;
                if (cleanValue.startsWith('"') && cleanValue.endsWith('"')) {
                    goValue = cleanValue;
                } else if (!isNaN(cleanValue)) {
                    goValue = cleanValue;
                } else if (cleanValue.includes('px') || cleanValue.includes('%')) {
                    goValue = `"${cleanValue}"`;
                } else {
                    goValue = `"${cleanValue}"`;
                }

                let codeSnippet = '';
                if (method.endsWith('Value')) {
                    codeSnippet = `doc.CSS.${namespace}${currentPseudoClass ? '.' + currentPseudoClass : ''}.${category}.${method}(${goValue})(e)`;
                } else {
                    codeSnippet = `doc.CSS.${namespace}${currentPseudoClass ? '.' + currentPseudoClass : ''}.${category}.${method}(e)`;
                }

                goCode += `\te = ${codeSnippet}\n`;
            } else {
                // Fallback for unmapped properties
                goCode += `\te = doc.CSS.${namespace}${currentPseudoClass ? '.' + currentPseudoClass : ''}.Style.CustomSetter("${property}")(${cleanValue})(e)\n`;
            }
        }
    }
    goCode += `\treturn e\n}`;
    return goCode;
}


const css = `
    color: red;
    font-size: 16px;

    :hover {
        color: blue;
        font-size: 18px;
    }

    :active {
        color: green;
    }
`;

const goCode = cssToGoCode(css);
console.log(goCode);
