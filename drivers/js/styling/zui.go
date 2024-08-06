// Package zds defines the zui design system that is used to build http://zui.dev
package zds

import (
	. "github.com/atdiar/particleui"
	doc "github.com/atdiar/particleui/drivers/js"
)

// Reset creates a reset stylesheet for the given stylesheetid.
// It should be the first stylesheet activated for a document.
func Reset(stylesheetid string) func(*Element) *Element {
	return func(root *Element) *Element {
		d := doc.GetDocument(root)
		sheet, ok := d.GetStyleSheet(stylesheetid)
		if !ok {
			sheet = d.NewStyleSheet(stylesheetid)
			// sheet activation
			actives := append([]string{stylesheetid}, d.GetActiveStyleSheets()...)
			d.SetActiveStyleSheets(actives...)
		}

		// Box sizing rules
		sheet.InsertRule(`*, *::before, *::after`, "box-sizing: border-box;")

		// Remove default margin
		sheet.InsertRule(`*`, `
			margin: 0;
		`)

		// Prevent font size inflation
		sheet.InsertRule(`html`, `
			-moz-text-size-adjust: none;
  			-webkit-text-size-adjust: none;
  			text-size-adjust: none;
			`)

		// Remove default margin in favour of better control in authored CSS
		sheet.InsertRule(`body, h1, h2, h3, h4, p, figure, blockquote, dl, dd`, `
			margin-block-end: 0;
		`)

		// Remove list styles on ul, ol elements with a list role, which suggests default styling will be removed
		sheet.InsertRule(`ul[role="list"], ol[role="list"]`, `
			list-style-type: none;
		`)

		// Set core body defaults
		sheet.InsertRule(`body`, `
			min-height: 100vh;
  			line-height: 1.5;
			webkit-font-smoothing: antialiased;
		`)

		// Set shorter line heights on headings and interactive elements
		sheet.InsertRule(`h1, h2, h3, h4, button, input, label`, `
			line-height: 1.1;
		`)

		// Balance text wrapping on headings
		sheet.InsertRule(`h1, h2, h3, h4`, `
			text-wrap: balance;
		`)

		// Avoid text overflows
		sheet.InsertRule(`h1, h2, h3, h4, p, figure, blockquote, dl, dd`, `
			overflow-wrap: break-word;
		`)

		// A elements that don't have a class get default styles
		sheet.InsertRule(`a:not([class])`, `
			text-decoration-skip-ink: auto;
			color: currentColor;
		`)

		// Improve media defaults
		sheet.InsertRule(`img, picture, video, canvas, svg`, `
			display: block;
			max-width: 100%;
		`)

		// Inherit fonts for inputs and buttons
		sheet.InsertRule(`button, input, select, textarea`, `
			font-family: inherit;
  			font-size: inherit;
		`)

		// Make sure textareas without a rows attribute are not tiny
		sheet.InsertRule(`textarea:not([rows])`, `
			min-height: 10em;
		`)

		// Anything that has been anchored to should have extra scroll margin
		sheet.InsertRule(`:target`, `
			scroll-margin-block: 5ex;
		`)

		// Create a root stacking context
		sheet.InsertRule(`#root, #__next`, `
			isolation: isolate;
		`)

		// Remove default padding//
		sheet.Update()

		return root
	}
}

// TODO make partial functions to modify this.
// Perhaps that some kind of modifier can be accepted as argument.
// Or some way to parameterize this (colors and whatnot)
func DarkModeNoClassDefaults(e *Element) *Element {
	id := "zui-dm-noclass"
	d := doc.GetDocument(e)
	sheet := d.NewStyleSheet(id)
	defer sheet.Update()
	defer d.SetActiveStyleSheets(id)

	sheet.InsertRule("html", `
		font-family: sans-serif;
		font-size: 10px;
		box-sizing: border-box;
		-webkit-text-size-adjust: 100%;
	`)

	sheet.InsertRule("*, :after, :before", `
		box-sizing: inherit;
		position: relative;
	`)

	sheet.InsertRule(":focus", `
		outline: 0;
	`)

	sheet.InsertRule("body", `
		color: #ffffff;
		background-color: #1c1c1c;
		margin: 0;
		font-size: 1.4rem;
		line-height: 1.8;
		font-weight: 300;
	`)

	sheet.InsertRule("@media (max-width: 50rem)", `
		body {
			overflow-x: hidden;
		}
	`)

	sheet.InsertRule("a", `
		background-color: transparent;
	`)

	sheet.InsertRule("a:active, a:hover", `
		outline: 0;
	`)

	sheet.InsertRule("b, strong", `
		font-weight: 700;
	`)

	sheet.InsertRule("h1, h2, h3, h4, h5, h6", `
		color: #ffffff;
		margin: 0 0 2rem;
		font-weight: 200;
	`)

	sheet.InsertRule("img", `
		border: 0;
	`)

	sheet.InsertRule("::-moz-selection", `
		background-color: #4d4d4d;
		color: #ffffff;
	`)

	sheet.InsertRule("::selection", `
		background-color: #4d4d4d;
		color: #ffffff;
	`)

	sheet.InsertRule("a:not([class])", `
		color: #ffffff;
		text-decoration: none;
		display: inline-block;
		z-index: 1;
	`)

	sheet.InsertRule("p a:not([class]):before", `
		content: "";
		display: inline-block;
		width: 100%;
		height: 100%;
		background: #ffffff;
		position: absolute;
		opacity: .2;
		-webkit-transform: scale3d(1, .1, 1);
		transform: scale3d(1, .1, 1);
		-webkit-transform-origin: bottom;
		transform-origin: bottom;
		z-index: -1;
	`)

	sheet.InsertRule("p a:not([class]):hover:before", `
		opacity: .4;
		-webkit-transform: none;
		transform: none;
	`)

	sheet.InsertRule("blockquote:not([class])", `
		margin: 2rem 0;
		padding: 1rem 2rem;
		border-left: 4px solid #4d4d4d;
	`)

	sheet.InsertRule("button:not([class]), input[type=submit]", `
		cursor: pointer;
		color: #ffffff;
		display: inline-block;
		padding: 1.4rem 2rem;
		background: #4d4d4d;
		border: 1px solid #ffffff;
		border-radius: 2px;
		box-shadow: 0 0 0 transparent;
		text-transform: uppercase;
		text-decoration: none;
		text-align: center;
		font-size: 1.2rem;
		font-weight: 700;
		line-height: 1rem;
		margin: 0 1rem 1rem 0;
		-webkit-appearance: none;
	`)

	sheet.InsertRule("button:not([class]):before, input[type=submit]:before", `
		content: "";
		position: absolute;
		z-index: -1;
		opacity: 0;
		width: 100%;
		height: 100%;
		left: 0;
		top: 0;
		-webkit-transform: scale3d(1.2, 1.2, 1.2);
		transform: scale3d(1.2, 1.2, 1.2);
		background: #ffffff;
	`)

	sheet.InsertRule("button:not([class]):not(:disabled):hover, input[type=submit]:not(:disabled):hover", `
		box-shadow: 2px 2px 4px rgba(255, 255, 255, .3);
		background: #3d3d3d;
	`)

	sheet.InsertRule("button:not([class]):not(:disabled):hover:active, input[type=submit]:not(:disabled):hover:active", `
		box-shadow: none;
		-webkit-transition: none;
		transition: none;
	`)

	sheet.InsertRule("code:not([class])", `
		display: inline-block;
		background: #3d3d3d;
		border: 1px solid #4d4d4d;
		padding: 0 .5rem;
		color: #ffffff;
		font-size: 1.2rem;
		line-height: 1.8;
		font-family: monospace;
		border-radius: 2px;
		text-transform: none;
		font-weight: 300;
	`)

	sheet.InsertRule("pre:not([class]) code", `
		padding: 2rem;
		border: none;
		border-left: 4px solid #4d4d4d;
		border-radius: 0;
		width: 100%;
		display: block;
	`)

	sheet.InsertRule("footer", `
		color: #ffffff;
		background-color: #1c1c1c;
		width: 100%;
		max-width: 90rem;
		margin: auto;
		padding: 2rem;
		overflow: visible;
	`)

	sheet.InsertRule("footer:before", `
		content: "";
		background: #4d4d4d;
		width: 102vw;
		height: 100%;
		display: block;
		position: absolute;
		left: 50%;
		margin-left: -51vw;
		z-index: -1;
	`)

	sheet.InsertRule("footer a:not([class])", `
		color: inherit;
		text-decoration: underline;
		display: inline-block;
		-webkit-text-decoration-skip: ink;
		text-decoration-skip: ink;
	`)

	sheet.InsertRule("footer a:not([class]):hover", `
		text-decoration: none;
	`)

	return e
}

func InitialNoClassDefaults(e *Element) *Element {
	id := "zui-initial-noclass"
	d := doc.GetDocument(e)
	sheet := d.NewStyleSheet(id)
	defer sheet.Update()
	defer d.SetActiveStyleSheets(id)

	sheet.InsertRule("html", `
        font-family: sans-serif;
        font-size: 10px;
        box-sizing: border-box;
        -webkit-text-size-adjust: 100%;
    `)

	sheet.InsertRule("*, :after, :before", `
        box-sizing: inherit;
        position: relative;
    `)

	sheet.InsertRule(":focus", `
        outline: 0;
    `)

	sheet.InsertRule("body", `
        color: #2e3538;
        margin: 0;
        font-size: 1.4rem;
        line-height: 1.8;
        font-weight: 300;
    `)

	sheet.InsertRule("@media (max-width: 50rem)", `
        body {
            overflow-x: hidden;
        }
    `)

	sheet.InsertRule("a", `
        background-color: transparent;
    `)

	sheet.InsertRule("a:active, a:hover", `
        outline: 0;
    `)

	sheet.InsertRule("b, strong", `
        font-weight: 700;
    `)

	sheet.InsertRule("h1, h2, h3, h4, h5, h6", `
        margin: 0 0 2rem;
        font-weight: 200;
    `)

	sheet.InsertRule("img", `
        border: 0;
    `)

	sheet.InsertRule("::-moz-selection", `
        background-color: #679;
        color: #fff;
    `)

	sheet.InsertRule("::selection", `
        background-color: #679;
        color: #fff;
    `)

	sheet.InsertRule("a:not([class])", `
        color: inherit;
        text-decoration: none;
        display: inline-block;
        z-index: 1;
    `)

	sheet.InsertRule("p a:not([class]):not([btn]):before", `
        content: "";
        display: inline-block;
        width: 100%;
        height: 100%;
        background: #fd0;
        position: absolute;
        opacity: .5;
        -webkit-transform: scale3d(1, .1, 1);
        transform: scale3d(1, .1, 1);
        -webkit-transform-origin: bottom;
        transform-origin: bottom;
        z-index: -1;
    `)

	sheet.InsertRule("p a:not([class]):not([btn]):hover:before", `
        -webkit-transform: none;
        transform: none;
    `)

	sheet.InsertRule("blockquote:not([class])", `
        margin: 2rem 0;
        padding: 1rem 2rem;
        border-left: 4px solid #679;
    `)

	sheet.InsertRule("button:not([class]), input[type=submit]", `
        cursor: pointer;
        color: #679;
        display: inline-block;
        padding: 1.4rem 2rem;
        background: #fff;
        border: 1px solid #679;
        border-radius: 2px;
        box-shadow: 0 0 0 transparent;
        text-transform: uppercase;
        text-decoration: none;
        text-align: center;
        font-size: 1.2rem;
        font-weight: 700;
        line-height: 1rem;
        margin: 0 1rem 1rem 0;
        -webkit-appearance: none;
    `)

	sheet.InsertRule("button:not([class]):before, input[type=submit]:before", `
        content: "";
        position: absolute;
        z-index: -1;
        opacity: 0;
        width: 100%;
        height: 100%;
        left: 0;
        top: 0;
        -webkit-transform: scale3d(1.2, 1.2, 1.2);
        transform: scale3d(1.2, 1.2, 1.2);
        background: #679;
    `)

	sheet.InsertRule("button:not([class]):not(:disabled):hover, input[type=submit]:not(:disabled):hover", `
        box-shadow: 2px 2px 4px rgba(0, 0, 0, .3);
        background: #f4f5f6;
    `)

	sheet.InsertRule("button:not([class]):not(:disabled):hover:active, input[type=submit]:not(:disabled):hover:active", `
        box-shadow: none;
        -webkit-transition: none;
        transition: none;
    `)

	sheet.InsertRule("code:not([class])", `
		display: inline-block;
		background: #3d3d3d;
		border: 1px solid #4d4d4d;
		padding: 0 .5rem;
		color: #ffffff;
		font-size: 1.2rem;
		line-height: 1.8;
		font-family: monospace;
		border-radius: 2px;
		text-transform: none;
		font-weight: 300;
	`)

	sheet.InsertRule("pre:not([class]) code", `
		padding: 2rem;
		border: none;
		border-left: 4px solid #4d4d4d;
		border-radius: 0;
		width: 100%;
		display: block;
	`)

	sheet.InsertRule("footer", `
		color: #ffffff;
		background-color: #1c1c1c;
		width: 100%;
		max-width: 90rem;
		margin: auto;
		padding: 2rem;
		overflow: visible;
	`)

	sheet.InsertRule("footer:before", `
		content: "";
		background: #4d4d4d;
		width: 102vw;
		height: 100%;
		display: block;
		position: absolute;
		left: 50%;
		margin-left: -51vw;
		z-index: -1;
	`)

	sheet.InsertRule("footer a:not([class])", `
		color: inherit;
		text-decoration: underline;
		display: inline-block;
		-webkit-text-decoration-skip: ink;
		text-decoration-skip: ink;
	`)

	sheet.InsertRule("footer a:not([class]):hover", `
		text-decoration: none;
	`)

	return e
}
