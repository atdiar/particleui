 
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
    'Container.Layout.VerticalAlign': {
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
    'Container.Style.ListStyle': {
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
    'Container.Style.WhiteSpace': {
        property: 'white-space',
        valueFunction: (value) => value
    },
    'Container.Style.BorderRight': {
        property: 'border-right',
        valueFunction: (value) => value
    },
    'Container.Style.TextDecorationLine': {
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
    'Container.Style.FontSize': {
        property: 'font-size',
        valueFunction: (value) => value
    },
    'Container.Style.LineHeight': {
        property: 'line-height',
        valueFunction: (value) => value
    },
    'Container.Style.TextDecorationStyle': {
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
    'Container.Style.TextDecoration': {
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
    'Container.Style.Quotes': {
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
    'Container.Style.FontSizeAdjust': {
        property: 'font-size-adjust',
        valueFunction: (value) => value
    },
    'Container.Style.ListStylePosition': {
        property: 'list-style-position',
        valueFunction: (value) => value
    },
    'Container.Style.TextAlign': {
        property: 'text-align',
        valueFunction: (value) => value
    },
    'Container.Style.TextJustify': {
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
    'Container.Style.Font': {
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
    'Container.Style.WordSpacing': {
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
    'Container.Style.LetterSpacing': {
        property: 'letter-spacing',
        valueFunction: (value) => value
    },
    'Container.Style.BorderBottomStyle': {
        property: 'border-bottom-style',
        valueFunction: (value) => value
    },
    'Container.Style.WordBreak': {
        property: 'word-break',
        valueFunction: (value) => value
    },
    'Container.Style.BorderBottomRightRadius': {
        property: 'border-bottom-right-radius',
        valueFunction: (value) => value
    },
    'Container.Style.FontStyle': {
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
    'Container.Style.TextAlignLast': {
        property: 'text-align-last',
        valueFunction: (value) => value
    },
    'Container.Style.BorderImageWidth': {
        property: 'border-image-width',
        valueFunction: (value) => value
    },
    'Container.Style.FontWeight': {
        property: 'font-weight',
        valueFunction: (value) => value
    },
    'Container.Style.ListStyleImage': {
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
    'Container.Style.CaptionSide': {
        property: 'caption-side',
        valueFunction: (value) => value
    },
    'Container.Style.FontFamily': {
        property: 'font-family',
        valueFunction: (value) => value
    },
    'Container.Style.TextDecorationColor': {
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
    'Container.Style.TextIndent': {
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
    'Container.Style.FontVariant': {
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
    'Container.Style.ListStyleType': {
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
    'Container.Style.WordWrap': {
        property: 'word-wrap',
        valueFunction: (value) => value
    },
    'Container.Style.BackgroundColor': {
        property: 'background-color',
        valueFunction: (value) => value
    },
    'Container.Style.TextOverflow': {
        property: 'text-overflow',
        valueFunction: (value) => value
    },
    'Container.Style.TextShadow': {
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
    'Container.Style.Color': {
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
    'Container.Style.FontStretch': {
        property: 'font-stretch',
        valueFunction: (value) => value
    },
    'Container.Style.TextTransform': {
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
    'box-shadow': 'Container.Layout.BoxShadow',
    'justify-content': 'Container.Layout.JustifyContent',
    'z-index': 'Container.Layout.ZIndex',
    'float': 'Container.Layout.Float',
    'overflow': 'Container.Layout.Overflow',
    'overflow-y': 'Container.Layout.OverflowY',
    'perspective': 'Container.Layout.Perspective',
    'border-collapse': 'Container.Layout.BorderCollapse',
    'page-break-before': 'Container.Layout.PageBreakBefore',
    'columns': 'Container.Layout.Columns',
    'column-count': 'Container.Layout.ColumnCount',
    'min-height': 'Container.Layout.MinHeight',
    'page-break-inside': 'Container.Layout.PageBreakInside',
    'column-gap': 'Container.Layout.ColumnGap',
    'clip': 'Container.Layout.Clip',
    'flex-direction': 'Container.Layout.FlexDirection',
    'page-break-after': 'Container.Layout.PageBreakAfter',
    'top': 'Container.Layout.Top',
    'counter-increment': 'Container.Layout.CounterIncrement',
    'height': 'Container.Layout.Height',
    'transform-style': 'Container.Layout.TransformStyle',
    'overflow-x': 'Container.Layout.OverflowX',
    'flex-wrap': 'Container.Layout.FlexWrap',
    'max-width': 'Container.Layout.MaxWidth',
    'bottom': 'Container.Layout.Bottom',
    'counter-reset': 'Container.Layout.CounterReset',
    'right': 'Container.Layout.Right',
    'box-sizing': 'Container.Layout.BoxSizing',
    'position': 'Container.Layout.Position',
    'table-layout': 'Container.Layout.TableLayout',
    'width': 'Container.Layout.Width',
    'max-height': 'Container.Layout.MaxHeight',
    'column-width': 'Container.Layout.ColumnWidth',
    'min-width': 'Container.Layout.MinWidth',
    'vertical-align': 'Container.Layout.VerticalAlign',
    'perspective-origin': 'Container.Layout.PerspectiveOrigin',
    'align-content': 'Container.Layout.AlignContent',
    'flex-flow': 'Container.Layout.FlexFlow',
    'display': 'Container.Layout.Display',
    'left': 'Container.Layout.Left',
    'background-image': 'Container.Style.BackgroundImage',
    'border-left-style': 'Container.Style.BorderLeftStyle',
    'transition-delay': 'Container.Style.TransitionDelay',
    'animation-duration': 'Container.Style.AnimationDuration',
    'list-style': 'Container.Style.ListStyle',
    'outline-width': 'Container.Style.OutlineWidth',
    'border-top-left-radius': 'Container.Style.BorderTopLeftRadius',
    'white-space': 'Container.Style.WhiteSpace',
    'border-right': 'Container.Style.BorderRight',
    'text-decoration-line': 'Container.Style.TextDecorationLine',
    'animation-delay': 'Container.Style.AnimationDelay',
    'background-position': 'Container.Style.BackgroundPosition',
    'border-image': 'Container.Style.BorderImage',
    'border-spacing': 'Container.Style.BorderSpacing',
    'border-image-outset': 'Container.Style.BorderImageOutset',
    'border-image-slice': 'Container.Style.BorderImageSlice',
    'border-left-color': 'Container.Style.BorderLeftColor',
    'font-size': 'Container.Style.FontSize',
    'line-height': 'Container.Style.LineHeight',
    'text-decoration-style': 'Container.Style.TextDecorationStyle',
    'backface-visibility': 'Container.Style.BackfaceVisibility',
    'border-right-style': 'Container.Style.BorderRightStyle',
    'text-decoration': 'Container.Style.TextDecoration',
    'transition': 'Container.Style.Transition',
    'animation-iteration-count': 'Container.Style.AnimationIterationCount',
    'border-bottom': 'Container.Style.BorderBottom',
    'animation-timing-function': 'Container.Style.AnimationTimingFunction',
    'border-radius': 'Container.Style.BorderRadius',
    'quotes': 'Container.Style.Quotes',
    'tab-size': 'Container.Style.TabSize',
    'animation-fill-mode': 'Container.Style.AnimationFillMode',
    'background-size': 'Container.Style.BackgroundSize',
    'font-size-adjust': 'Container.Style.FontSizeAdjust',
    'list-style-position': 'Container.Style.ListStylePosition',
    'text-align': 'Container.Style.TextAlign',
    'text-justify': 'Container.Style.TextJustify',
    'background-attachment': 'Container.Style.BackgroundAttachment',
    'border-right-width': 'Container.Style.BorderRightWidth',
    'font': 'Container.Style.Font',
    'border-left': 'Container.Style.BorderLeft',
    'transition-duration': 'Container.Style.TransitionDuration',
    'word-spacing': 'Container.Style.WordSpacing',
    'animation-name': 'Container.Style.AnimationName',
    'animation-play-state': 'Container.Style.AnimationPlayState',
    'letter-spacing': 'Container.Style.LetterSpacing',
    'border-bottom-style': 'Container.Style.BorderBottomStyle',
    'word-break': 'Container.Style.WordBreak',
    'border-bottom-right-radius': 'Container.Style.BorderBottomRightRadius',
    'font-style': 'Container.Style.FontStyle',
    'order': 'Container.Style.Order',
    'outline-style': 'Container.Style.OutlineStyle',
    'border-bottom-left-radius': 'Container.Style.BorderBottomLeftRadius',
    'border-image-source': 'Container.Style.BorderImageSource',
    'text-align-last': 'Container.Style.TextAlignLast',
    'border-image-width': 'Container.Style.BorderImageWidth',
    'font-weight': 'Container.Style.FontWeight',
    'list-style-image': 'Container.Style.ListStyleImage',
    'opacity': 'Container.Style.Opacity',
    'clear': 'Container.Style.Clear',
    'border-top-color': 'Container.Style.BorderTopColor',
    'border': 'Container.Style.Border',
    'border-right-color': 'Container.Style.BorderRightColor',
    'transition-timing-function': 'Container.Style.TransitionTimingFunction',
    'border-bottom-width': 'Container.Style.BorderBottomWidth',
    'border-style': 'Container.Style.BorderStyle',
    'border-top-right-radius': 'Container.Style.BorderTopRightRadius',
    'caption-side': 'Container.Style.CaptionSide',
    'font-family': 'Container.Style.FontFamily',
    'text-decoration-color': 'Container.Style.TextDecorationColor',
    'transition-property': 'Container.Style.TransitionProperty',
    'background-origin': 'Container.Style.BackgroundOrigin',
    'text-indent': 'Container.Style.TextIndent',
    'visibility': 'Container.Style.Visibility',
    'border-color': 'Container.Style.BorderColor',
    'border-top': 'Container.Style.BorderTop',
    'font-variant': 'Container.Style.FontVariant',
    'outline': 'Container.Style.Outline',
    'border-bottom-color': 'Container.Style.BorderBottomColor',
    'border-top-style': 'Container.Style.BorderTopStyle',
    'border-width': 'Container.Style.BorderWidth',
    'list-style-type': 'Container.Style.ListStyleType',
    'outline-offset': 'Container.Style.OutlineOffset',
    'animation': 'Container.Style.Animation',
    'background': 'Container.Style.Background',
    'background-repeat': 'Container.Style.BackgroundRepeat',
    'border-top-width': 'Container.Style.BorderTopWidth',
    'word-wrap': 'Container.Style.WordWrap',
    'background-color': 'Container.Style.BackgroundColor',
    'text-overflow': 'Container.Style.TextOverflow',
    'text-shadow': 'Container.Style.TextShadow',
    'background-clip': 'Container.Style.BackgroundClip',
    'border-left-width': 'Container.Style.BorderLeftWidth',
    'resize': 'Container.Style.Resize',
    'animation-direction': 'Container.Style.AnimationDirection',
    'color': 'Container.Style.Color',
    'outline-color': 'Container.Style.OutlineColor',
    'border-image-repeat': 'Container.Style.BorderImageRepeat',
    'font-stretch': 'Container.Style.FontStretch',
    'text-transform': 'Container.Style.TextTransform',
    'flex-grow': 'Content.Layout.FlexGrow',
    'align-self': 'Content.Layout.AlignSelf',
    'content': 'Content.Layout.Content',
    'column-span': 'Content.Layout.ColumnSpan',
    'flex': 'Content.Layout.Flex',
    'flex-shrink': 'Content.Layout.FlexShrink',
    'flex-basis': 'Content.Layout.FlexBasis',
    'align-items': 'Content.Layout.AlignItems',
    'column-rule-width': 'Content.Style.ColumnRuleWidth',
    'column-rule': 'Content.Style.ColumnRule',
    'direction': 'Content.Style.Direction',
    'column-rule-style': 'Content.Style.ColumnRuleStyle',
    'column-rule-color': 'Content.Style.ColumnRuleColor',
    'column-fill': 'Content.Style.ColumnFill',
    'empty-cells': 'Content.Style.EmptyCells',
    'cursor': 'Content.Style.Cursor',
    'transform': 'Container.Style.Transform',
    'pointer-events': 'Container.Style.PointerEvents',
    'user-select': 'Container.Style.UserSelect',
    'backdrop-filter': 'Container.Style.BackdropFilter',
    'object-fit': 'Container.Style.ObjectFit',
    'object-position': 'Container.Style.ObjectPosition',
    'margin': 'Container.Layout.Margin',
    'margin-top': 'Container.Layout.MarginTop',
    'margin-right': 'Container.Layout.MarginRight',
    'margin-bottom': 'Container.Layout.MarginBottom',
    'margin-left': 'Container.Layout.MarginLeft',
    'padding-top': 'Container.Layout.PaddingTop',
    'padding-right': 'Container.Layout.PaddingRight',
    'padding-bottom': 'Container.Layout.PaddingBottom',
    'padding-left': 'Container.Layout.PaddingLeft',
    'grid-template': 'Container.Layout.GridTemplate',
    'grid-auto-columns': 'Container.Layout.GridAutoColumns', // Not in Go
    'grid-auto-rows': 'Container.Layout.GridAutoRows', // Not in Go
    'grid-auto-flow': 'Container.Layout.GridAutoFlow', // ?? in go?
    'grid': 'Container.Layout.Grid', // ?? in go?
    'grid-gap': 'Container.Layout.GridGap', // Not in Go
    'grid-row-gap': 'Container.Layout.GridRowGap',// Not in Go
    'grid-column-gap': 'Container.Layout.GridColumnGap', // Not in Go
    'grid-row-start': 'Container.Layout.GridRowStart', // Not in Go
    'grid-column-start': 'Container.Layout.GridColumnStart', // Not in Go
    'padding': 'Container.Layout.Padding',
    'grid-template-columns': 'Container.Layout.GridTemplateColumns',
    'grid-template-rows': 'Container.Layout.GridTemplateRows',
    'grid-column': 'Container.Layout.GridColumn',
    'grid-row': 'Container.Layout.GridRow',
    'gap': 'Container.Layout.Gap',
    'scroll-behavior': 'Container.Layout.ScrollBehavior'
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




/*

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
    'Container.Layout.VerticalAlign': {
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
    'Container.Style.ListStyle': {
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
    'Container.Style.WhiteSpace': {
        property: 'white-space',
        valueFunction: (value) => value
    },
    'Container.Style.BorderRight': {
        property: 'border-right',
        valueFunction: (value) => value
    },
    'Container.Style.TextDecorationLine': {
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
    'Container.Style.FontSize': {
        property: 'font-size',
        valueFunction: (value) => value
    },
    'Container.Style.LineHeight': {
        property: 'line-height',
        valueFunction: (value) => value
    },
    'Container.Style.TextDecorationStyle': {
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
    'Container.Style.TextDecoration': {
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
    'Container.Style.Quotes': {
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
    'Container.Style.FontSizeAdjust': {
        property: 'font-size-adjust',
        valueFunction: (value) => value
    },
    'Container.Style.ListStylePosition': {
        property: 'list-style-position',
        valueFunction: (value) => value
    },
    'Container.Style.TextAlign': {
        property: 'text-align',
        valueFunction: (value) => value
    },
    'Container.Style.TextJustify': {
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
    'Container.Style.Font': {
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
    'Container.Style.WordSpacing': {
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
    'Container.Style.LetterSpacing': {
        property: 'letter-spacing',
        valueFunction: (value) => value
    },
    'Container.Style.BorderBottomStyle': {
        property: 'border-bottom-style',
        valueFunction: (value) => value
    },
    'Container.Style.WordBreak': {
        property: 'word-break',
        valueFunction: (value) => value
    },
    'Container.Style.BorderBottomRightRadius': {
        property: 'border-bottom-right-radius',
        valueFunction: (value) => value
    },
    'Container.Style.FontStyle': {
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
    'Container.Style.TextAlignLast': {
        property: 'text-align-last',
        valueFunction: (value) => value
    },
    'Container.Style.BorderImageWidth': {
        property: 'border-image-width',
        valueFunction: (value) => value
    },
    'Container.Style.FontWeight': {
        property: 'font-weight',
        valueFunction: (value) => value
    },
    'Container.Style.ListStyleImage': {
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
    'Container.Style.CaptionSide': {
        property: 'caption-side',
        valueFunction: (value) => value
    },
    'Container.Style.FontFamily': {
        property: 'font-family',
        valueFunction: (value) => value
    },
    'Container.Style.TextDecorationColor': {
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
    'Container.Style.TextIndent': {
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
    'Container.Style.FontVariant': {
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
    'Container.Style.ListStyleType': {
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
    'Container.Style.WordWrap': {
        property: 'word-wrap',
        valueFunction: (value) => value
    },
    'Container.Style.BackgroundColor': {
        property: 'background-color',
        valueFunction: (value) => value
    },
    'Container.Style.TextOverflow': {
        property: 'text-overflow',
        valueFunction: (value) => value
    },
    'Container.Style.TextShadow': {
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
    'Container.Style.Color': {
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
    'Container.Style.FontStretch': {
        property: 'font-stretch',
        valueFunction: (value) => value
    },
    'Container.Style.TextTransform': {
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
*/