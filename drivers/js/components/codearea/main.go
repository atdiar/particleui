// package code is a package that provides a code input component.
package code

import (
	js "github.com/atdiar/particleui/drivers/js/compat"

	ui "github.com/atdiar/particleui"
	. "github.com/atdiar/particleui/drivers/js"
)

// This package use a js library to provide a code input component.
// Could have rewritten everything in Go and perhaps will in the future if we have direct access to the DOM.
// For now it is not important. Even further, it provides a good example of integration of a js library into a zui project.

const scriptID = "zui-codearea"

func implementation(d *Document) ScriptElement {
	return d.Script.WithID(scriptID).SetInnerHTML(jscode)
}

func addIfAbsent(d *Document) {
	if d.GetElementById(scriptID) == nil {
		d.Head().AppendChild(implementation(d))
	}
}

type AreaElement struct {
	*ui.Element
}

func Area(d *Document, id string) AreaElement {
	addIfAbsent(d)
	a := AreaElement{d.Div.WithID(id).AsElement()}
	v, ok := JSValue(a)
	if !ok {
		panic("Element has no js value")
	}
	js.Global().Call("createCodeEditor", v)
	return a
}

func (e AreaElement) SetLanguage(l string) AreaElement {
	v, ok := JSValue(e.Element)
	if !ok {
		panic("Element has no js vlue")
	}
	v.Call("setLanguage", l)
	return e
}

func (e AreaElement) Resize(width, height string) AreaElement {
	v, ok := JSValue(e.Element)
	if !ok {
		panic("Element has no js vlue")
	}
	v.Call("resize", width, height)
	return e
}

func (e AreaElement) SetTheme(t string) AreaElement {
	v, ok := JSValue(e.Element)
	if !ok {
		panic("Element has no js vlue")
	}
	v.Call("setTheme", t)
	return e
}

func (e AreaElement) SetValue(v string) AreaElement {
	jsv, ok := JSValue(e.Element)
	if !ok {
		panic("Element has no js vlue")
	}
	jsv.Call("setValue", v)
	return e
}

func (e AreaElement) GetValue() string {
	v, ok := JSValue(e.Element)
	if !ok {
		panic("Element has no js vlue")
	}
	return v.Call("getValue").String()
}

func (e AreaElement) Snapshot(b bool) AreaElement {
	v, ok := JSValue(e.Element)
	if !ok {
		panic("Element has no js vlue")
	}
	v.Call("snapshot", b)
	return e
}

func (e AreaElement) Refresh() AreaElement {
	v, ok := JSValue(e.Element)
	if !ok {
		panic("Element has no js vlue")
	}
	v.Call("refresh")
	return e

}

func (e AreaElement) SetOutput(messages []string, t string) AreaElement {
	v, ok := JSValue(e.Element)
	if !ok {
		panic("Element has no js vlue")
	}
	v.Call("setOutput", messages, t)
	return e
}

func (e AreaElement) ClearOutput() AreaElement {
	v, ok := JSValue(e.Element)
	if !ok {
		panic("Element has no js vlue")
	}
	v.Call("clearOutput")
	return e

}

func (e AreaElement) Undo() AreaElement {
	v, ok := JSValue(e.Element)
	if !ok {
		panic("Element has no js vlue")
	}
	v.Call("undo")
	return e
}

func (e AreaElement) Redo() AreaElement {
	v, ok := JSValue(e.Element)
	if !ok {
		panic("Element has no js vlue")
	}
	v.Call("redo")
	return e
}

func (e AreaElement) On(event string, callback func()) AreaElement {
	// let's retrieve the native textarea element first
	v, ok := JSValue(e.Element)
	if !ok {
		panic("Element has no js vlue")
	}
	ta := v.Get("textarea")
	cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		callback()
		return nil
	})
	e.OnDeleted(ui.OnMutation(func(evt ui.MutationEvent) bool {
		cb.Release()
		return false
	}).RunOnce().RunASAP())

	ta.Call("on", event, cb)

	return e
}

var jscode = `
function createCodeEditor(container, options = {}) {
    const defaultOptions = {
        value: '',
        language: 'javascript',
        lineNumbers: true,
        errorChecking: true,
        syntaxHighlighting: true,
        tabSize: 4,
        theme: 'light', // 'light' or 'dark'
        snapshot: false,
        width: '100%',  // Default width
        height: '70vh'  // Default height
    };

    const settings = { ...defaultOptions, ...options };

    // Create and append style element
    const style = document.createElement('style');
    style.textContent = ` + "`" + `
        .code-editor-wrapper {
            font-size: 14px;
            line-height: 1.5;
            font-family: monospace;
            border: 1px solid #ccc;
            border-radius: 8px;
            overflow: hidden;
            width: ${settings.width};
            height: ${settings.height};
        }
        .code-area {
            position: relative;
            height: calc(100% - 30px); // Adjust for info bar
            height: 70vh;
            overflow: hidden;
            border-radius: 4px;
        }
        .line-numbers {
            position: absolute;
            left: 0;
            top: 0;
            bottom: 0;
            width: 40px;
            padding: 10px 5px;
            text-align: right;
            user-select: none;
            overflow: hidden;
        }
        .code-wrapper {
            position: absolute;
            left: 50px;
            right: 0;
            top: 0;
            bottom: 0;
        }
        #codeAreaElement, #codeHighlight {
            position: absolute;
            left: 0;
            top: 0;
            width: 100%;
            height: 100%;
            margin: 0;
            padding: 10px;
            border: none;
            font-family: inherit;
            font-size: inherit;
            line-height: inherit;
            tab-size: 4;
            box-sizing: border-box;
            white-space: pre;
            overflow: hidden;
            scrollbar-width: thin;
            scrollbar-color: #ccc transparent;
        }
        #codeAreaElement:hover, #codeHighlight:hover {
            cursor: text;
            overflow: auto;
        }
        #codeHighlight code {
            display: inline-block;
            min-width: 100%;
            box-sizing: border-box;
        }
        
        #codeAreaElement {
            z-index: 1;
            color: transparent;
            background: transparent;
            caret-color: black;
            resize: none;
            outline: none;
        }
        #codeHighlight {
            z-index: 0;
            pointer-events: none;
            overflow: hidden;
        }
        #codeHighlight code {
            display: block;
            font-family: inherit;
            font-size: inherit;
            line-height: inherit;
            padding: 0;
            margin: 0;
        }
        .loc-counter {
            text-align: right;
            padding: 5px 10px;
            font-size: 12px;
        }
        .output-area {
            padding: 10px;
            display: none;
            max-height: 100px;
            overflow-y: auto;
        }
        .output-area p {
            margin: 5px 0;
        }
        .error-lines {
            position: absolute;
            left: 0;
            top: 0;
            width: 100%;
            height: 100%;
            font-family: inherit;
            font-size: inherit;
            line-height: inherit;
            pointer-events: none;
            z-index: 1;
        }

        .error-line {
            position: absolute;
            left: 0;
            right: 0;
            height: 21px; /* Adjust this value to match your line-height */
        }

        .info-bar {
            display: flex;
            justify-content: space-between;
            padding: 5px 10px;
            background-color: inherit;
            font-size: 12px;
        }
        .language-display, .loc-counter {
            color: #666;
        }
        .neon-blue-text {
            padding-left: 3px;
            padding-right: 2px;
            color: rgb(139, 139, 255);
            border: 1px solid rgb(139, 139, 255);
            border-radius: 2px;
            box-shadow: 0 0 5px rgba(139, 139, 255,0.5);
            display: inline-block;
            font-size: x-small;
            text-shadow: 2px 2px 7px rgba(47,31,206,0.66), 2px 8px 10px rgba(204,204,206,0.58);
            pointer-events: none;
        }

        .code-editor-wrapper.snapshot #codeAreaElement {
            pointer-events: none;
            user-select: none;
            opacity: 0;
        }
        .code-editor-wrapper.snapshot #codeHighlight {
            pointer-events: auto;
            user-select: text;
            z-index: 2;
        }

        .code-editor-wrapper code[class*="language-"],
        .code-editor-wrapper pre[class*="language-"] {
            text-shadow: none !important;
        }
        .code-editor-wrapper.dark .token.operator,
        .code-editor-wrapper.dark .token.entity,
        .code-editor-wrapper.dark .token.url,
        .code-editor-wrapper.dark .language-css .token.string,
        .code-editor-wrapper.dark .style .token.string {
            background: transparent; /* Remove the background */
        }

        /* Light theme */
        .code-editor-wrapper.light {
            color: #333;
            background-color: #fff;
        }
        .code-editor-wrapper.light code[class*="language-"],
        .code-editor-wrapper.light pre[class*="language-"] {
            color: #333;
            background: transparent;
        }
        .code-editor-wrapper.light .line-numbers {
            background-color: #f0f0f0;
            border-right: 1px solid #ccc;
            color: #999;
        }
        .code-editor-wrapper.light #codeAreaElement {
            caret-color: #333;
        }
        .code-editor-wrapper.light #codeHighlight {
            color: #333;
            background-color: #fff;
        }
        .code-editor-wrapper.light .loc-counter {
            background-color: #e0e0e0;
            color: #666;
        }
        .code-editor-wrapper.light .output-area {
            background-color: #f8f8f8;
            border-top: 1px solid #ddd;
        }
        .code-editor-wrapper.light .error-line {
            background-color: rgba(255, 0, 0, 0.1);
        }
        .code-editor-wrapper.light .output-area p.error { color: #d32f2f; }
        .code-editor-wrapper.light .output-area p.info { color: #1976d2; }

        /* Dark theme */
        .code-editor-wrapper.dark {
            color: #f8f8f2;
            background-color: rgba(0,0,0,0.6);
            backdrop-filter: blur(10px);
            border-color: #333;
        }
        .code-editor-wrapper.dark code[class*="language-"],
        .code-editor-wrapper.dark pre[class*="language-"] {
            color: #f8f8f2;
            background: transparent;
        }
        .code-editor-wrapper.dark .line-numbers {
            background-color: rgba(0,0,0,0.6);
            color: #666;
        }
        .code-editor-wrapper.dark #codeAreaElement {
            caret-color: #e0e0e0;
            scrollbar-color: #666 transparent;
        }
        .code-editor-wrapper.dark #codeHighlight {
            color: #e0e0e0;
            background-color: rgba(0,0,0,0.6);
        }
        .code-editor-wrapper.dark .loc-counter {
            background-color: rgba(0,0,0,0.6);
            color: #888;
        }
        .code-editor-wrapper.dark .output-area {
            background-color: #252525;
            border-top: 1px solid #333;
        }
        .code-editor-wrapper.dark .error-line {
            background-color: rgba(255, 0, 0, 0.2);
        }
        .code-editor-wrapper.dark .output-area p.error { color: #f44336; }
        .code-editor-wrapper.dark .output-area p.info { color: #2196f3; }
        
        .code-editor-wrapper .token-line {
            display: inline-block;
            width: 100%;
        }
        .code-editor-wrapper .token-line:hover {
            background-color: rgba(0, 0, 0, 0.1);
        }
        .code-editor-wrapper.dark .token-line:hover {
            background-color: rgba(255, 255, 255, 0.1);
        }
    ` + "`" + `;
    document.head.appendChild(style);

    // Create DOM elements
    const wrapper = document.createElement('div');
    wrapper.className = ` + "`" + `code-editor-wrapper ${settings.theme}` + "`" + `;
    container.appendChild(wrapper);

    const infoBar = document.createElement('div');
    infoBar.className = 'info-bar';
    wrapper.appendChild(infoBar);

    const codeArea = document.createElement('div');
    codeArea.className = 'code-area';
    wrapper.appendChild(codeArea);

    const lineNumbers = document.createElement('div');
    lineNumbers.className = 'line-numbers';
    codeArea.appendChild(lineNumbers);

    const codeWrapper = document.createElement('div');
    codeWrapper.className = 'code-wrapper';
    codeArea.appendChild(codeWrapper);

    const textarea = document.createElement('textarea');
    textarea.id = 'codeAreaElement' + "-" + container.id;
    textarea.spellcheck = false;
    codeWrapper.appendChild(textarea);

    const highlight = document.createElement('pre');
    highlight.id = 'codeHighlight'+ "-" + container.id;
    const code = document.createElement('code');
    code.className = ` + "`" + `language-${settings.language}` + "`" + `;
    highlight.appendChild(code);
    codeWrapper.appendChild(highlight);

    const languageDisplay = document.createElement('span');
    languageDisplay.className = 'language-display neon-blue-text';
    infoBar.appendChild(languageDisplay);


    const locCounter = document.createElement('div');
    locCounter.className = 'loc-counter';
    wrapper.appendChild(locCounter);

    const outputArea = document.createElement('div');
    outputArea.className = 'output-area';
    wrapper.appendChild(outputArea);

    const errorLines = document.createElement('div');
    errorLines.className = 'error-lines';
    codeWrapper.appendChild(errorLines);

    
    let history = [];
    let historyIndex = -1;
    let debounceTimeout = null;
    let lastInsertedPair = null;
    const pairChars = { '{': '}', '(': ')', '[': ']' };
    const bracketPairs = { '(': ')', '[': ']', '{': '}' };
    const closingBrackets = { ')': '(', ']': '[', '}': '{' };
    let highlightedBrackets = [];

    function saveState() {
        const currentState = {
            value: textarea.value,
            selectionStart: textarea.selectionStart,
            selectionEnd: textarea.selectionEnd,
            scrollTop: textarea.scrollTop,
            scrollLeft: textarea.scrollLeft
        };
        
        if (historyIndex === -1 || JSON.stringify(history[historyIndex]) !== JSON.stringify(currentState)) {
            historyIndex++;
            history = history.slice(0, historyIndex);
            history.push(currentState);
        }
    }

    function restoreState(state, preserveScroll = false) {
        const scrollTop = preserveScroll ? textarea.scrollTop : state.scrollTop;
        const scrollLeft = preserveScroll ? textarea.scrollLeft : state.scrollLeft;
        textarea.value = state.value;
        textarea.selectionStart = state.selectionStart;
        textarea.selectionEnd = state.selectionEnd;
        refreshEditor(true);
        textarea.scrollTop = scrollTop;
        textarea.scrollLeft = scrollLeft;
        syncScroll();
    }

    function undo() {
        if (historyIndex > 0) {
            historyIndex--;
            restoreState(history[historyIndex], true);
        }
    }

    function redo() {
        if (historyIndex < history.length - 1) {
            historyIndex++;
            restoreState(history[historyIndex], true);
        }
    }

    function handleEnter(e) {
        e.preventDefault();
        const start = textarea.selectionStart;
        const end = textarea.selectionEnd;
        const text = textarea.value;
        const lineStart = text.lastIndexOf('\n', start - 1) + 1;
        const lineEnd = text.indexOf('\n', start);
        const currentLine = text.substring(lineStart, lineEnd === -1 ? text.length : lineEnd);
        
        const indentMatch = currentLine.match(/^\s*/);
        const currentIndent = indentMatch ? indentMatch[0] : '';
        
        let insertion = '\n' + currentIndent;
        let newCursorPos = start + insertion.length;

        if (text[start - 1] === '{' && text[start] === '}') {
            const additionalIndent = ' '.repeat(settings.tabSize);
            insertion = '\n' + currentIndent + additionalIndent + '\n' + currentIndent;
            newCursorPos = start + currentIndent.length + additionalIndent.length + 1;
        }
        
        textarea.value = text.substring(0, start) + 
                         insertion + 
                         text.substring(end);
        
        textarea.selectionStart = textarea.selectionEnd = newCursorPos;
        saveState();
        refreshEditor();
    }

    function insertPair(opening, closing) {
        const start = textarea.selectionStart;
        const end = textarea.selectionEnd;
        
        if (start === end) {
            textarea.value = textarea.value.substring(0, start) + 
                            opening + closing + 
                            textarea.value.substring(end);
            textarea.selectionStart = textarea.selectionEnd = start + 1;
            lastInsertedPair = { char: closing, position: start + 1 };
        } else {
            textarea.value = textarea.value.substring(0, start) + 
                            opening + 
                            textarea.value.substring(start, end) + 
                            closing + 
                            textarea.value.substring(end);
            textarea.selectionStart = start + 1;
            textarea.selectionEnd = end + 1;
        }
        
        saveState();
        refreshEditor();
    }
    

    function handleKeydown(e) {
        if (e.key === 'Tab') {
            e.preventDefault();
            if (textarea.selectionStart === textarea.selectionEnd) {
                // Single line indentation
                const start = textarea.selectionStart;
                const end = textarea.selectionEnd;

                const newValue = textarea.value.substring(0, start) + 
                                 " ".repeat(settings.tabSize) + 
                                 textarea.value.substring(end);

                textarea.value = newValue;
                textarea.selectionStart = textarea.selectionEnd = start + settings.tabSize;
            } else {
                // Multi-line indentation
                const start = textarea.value.lastIndexOf('\n', textarea.selectionStart - 1) + 1;
                const end = textarea.value.indexOf('\n', textarea.selectionEnd);
                const selectedLines = textarea.value.substring(start, end === -1 ? textarea.value.length : end).split('\n');
                
                const indentedLines = selectedLines.map(line => 
                    e.shiftKey ? line.replace(/^ {1,4}/, '') : ' '.repeat(settings.tabSize) + line
                );
                
                const newValue = textarea.value.substring(0, start) + 
                                 indentedLines.join('\n') + 
                                 textarea.value.substring(end === -1 ? textarea.value.length : end);
                
                const newStart = start;
                const newEnd = start + indentedLines.join('\n').length;
                
                textarea.value = newValue;
                textarea.selectionStart = newStart;
                textarea.selectionEnd = newEnd;
            }
            
            saveState();
            refreshEditor();
        } else if (e.key === 'Enter') {
            handleEnter(e);
        } if (Object.values(pairChars).includes(e.key)) {
            const cursorPosition = textarea.selectionStart;
            if (textarea.value[cursorPosition] === e.key) {
                // If the next character is already the closing bracket we're trying to type
                e.preventDefault();
                textarea.selectionStart = textarea.selectionEnd = cursorPosition + 1;
                refreshEditor();
            } else if (pairChars.hasOwnProperty(textarea.value[cursorPosition - 1]) && 
                    pairChars[textarea.value[cursorPosition - 1]] === e.key) {
                // If we're typing a closing bracket right after its opening bracket
                e.preventDefault();
                insertPair(textarea.value[cursorPosition - 1], e.key);
            }
            // Reset lastInsertedPair in both cases
            lastInsertedPair = null;
        } else if (pairChars.hasOwnProperty(e.key)) {
            e.preventDefault();
            insertPair(e.key, pairChars[e.key]);
        } else if (e.key === 'Backspace') {
            const start = textarea.selectionStart;
            const end = textarea.selectionEnd;
            if (start === end && start > 0) {
                const charBefore = textarea.value[start - 1];
                const charAfter = textarea.value[start];
                if (pairChars.hasOwnProperty(charBefore) && pairChars[charBefore] === charAfter) {
                    e.preventDefault();
                    textarea.value = textarea.value.substring(0, start - 1) + 
                                     textarea.value.substring(start + 1);
                    textarea.selectionStart = textarea.selectionEnd = start - 1;
                    saveState();
                    refreshEditor();
                }
            }
        } else if (e.key === 'z' && (e.ctrlKey || e.metaKey)) {
            e.preventDefault();
            if (e.shiftKey) {
                redo();
            } else {
                undo();
            }
        }
        
        if (!Object.values(pairChars).includes(e.key) && !pairChars.hasOwnProperty(e.key)) {
            lastInsertedPair = null;
        }
    }
  
    
    code.className = ` + "`" + `language-${settings.language}` + "`" + `;

    function updateLanguageDisplay() {
        languageDisplay.textContent = ` + "`" + `${settings.language}` + "`" + `;
    }
    
    // Helper functions
    function updateLOCCounter() {
        const lineCount = textarea.value.split('\n').length;
        locCounter.textContent = ` + "`" + `LOC: ${lineCount}` + "`" + `;
    }

    function updateLineNumbers() {
        const content = textarea.value;
        const lineCount = content.split('\n').length;
        const numbers = Array(lineCount).fill().map((_, i) =>` + "`" + `<div>${i + 1}</div>` + "`" + `).join('');
        lineNumbers.innerHTML = numbers;
        lineNumbers.style.height = ` + "`" + `${textarea.scrollHeight}px` + "`" + `;
    }

    function updateOutput(messages, type = 'info') {
        errorLines.innerHTML = '';

        if (messages.length > 0) {
            const outputHTML = messages.map(message =>
                ` + "`" + `<p class="${type}">${message.text}${message.lineNumber ? ` + "`" + ` on line ${message.lineNumber}` + "`" + ` : ''}</p>` + "`" + `
            ).join('');
            outputArea.innerHTML = outputHTML;
            outputArea.style.display = 'block';

            if (type === 'error') {
                const codeLines = textarea.value.split('\n');

                const highlightRect = highlight.getBoundingClientRect();
                const highlightStyle = window.getComputedStyle(highlight);
                const paddingTop = parseFloat(highlightStyle.paddingTop);
                const lineHeight = parseFloat(highlightStyle.lineHeight);

                messages.forEach(message => {
                    if (message.lineNumber && message.lineNumber <= codeLines.length) {
                        const errorLine = document.createElement('div');
                        errorLine.className = 'error-line';
                        
                        // Calculate the top position considering padding and line height
                        const topPosition = paddingTop + (message.lineNumber - 1) * lineHeight;
                        
                        errorLine.style.top = ` + "`" + `${topPosition}px` + "`" + `;
                        errorLine.style.height = ` + "`" + `${lineHeight}px` + "`" + `;
                        errorLine.style.width = '100%';
                        errorLine.style.left = '0';
                        
                        errorLines.appendChild(errorLine);
                    }
                });
            }
        } else {
            outputArea.innerHTML = '';
            outputArea.style.display = 'none';
        }
    }

    function checkForErrors(code) {
        const errors = [];
        const lines = code.split('\n');
        lines.forEach((line, index) => {
            if (line.includes('error')) {
                errors.push({ text: "Example error detected", lineNumber: index + 1 });
            }
        });
        return errors;
    }

    function updateHighlight() {
    const content = textarea.value;
    code.textContent = content;
    
    if (settings.syntaxHighlighting) {
        Prism.highlightElement(code);
    }

    if (settings.errorChecking) {
        const errors = checkForErrors(content);
        updateOutput(errors, 'error');
    }

    // Ensure highlight matches textarea size
    highlight.style.width = ` + "`" + `${textarea.clientWidth}px` + "`" + `;
    highlight.style.height = ` + "`" + `${textarea.clientHeight}px` + "`" + `;
}

    function checkForErrors(code) {
        const errors = [];
        const lines = code.split('\n');
        lines.forEach((line, index) => {
            if (line.includes('error')) {
                errors.push({ text: "Example error detected", lineNumber: index + 1 });
            }
        });
        return errors;
    }

    function syncScroll() {
        highlight.scrollTop = textarea.scrollTop;
        highlight.scrollLeft = textarea.scrollLeft;
        lineNumbers.style.top = ` + "`" + `-${textarea.scrollTop}px` + "`" + `;

        // Ensure the highlight content is aligned with the textarea content
        const highlightContent = highlight.querySelector('code');
        if (highlightContent) {
            highlightContent.style.minHeight = ` + "`" + `${textarea.scrollHeight}px` + "`" + `;
        }
    }

    function refreshEditor() {
        updateHighlight();
        updateLineNumbers();
        updateLOCCounter();
        updateLanguageDisplay();
        setSnapshotMode(settings.snapshot);
        syncScroll();
    }

    function refreshEditor(preserveScroll = false) {
        const scrollTop = textarea.scrollTop;
        const scrollLeft = textarea.scrollLeft;

        updateHighlight();
        updateLineNumbers();
        updateLOCCounter();
        updateLanguageDisplay();
        setSnapshotMode(settings.snapshot);

        if (preserveScroll) {
            textarea.scrollTop = scrollTop;
            textarea.scrollLeft = scrollLeft; 
            syncScroll();     
        }
        
    }

    // Event listeners
    let lastInsertedBrace = null;

    

    // Resize observer
    const resizeObserver = new ResizeObserver(refreshEditor);
    resizeObserver.observe(wrapper);

    function addEventListeners() {
        if (!settings.snapshot) {
            textarea.addEventListener('keydown', handleKeydown);

            textarea.addEventListener('input', function(e) {
                clearTimeout(debounceTimeout);
                debounceTimeout = setTimeout(() => {
                    saveState();
                    refreshEditor();
                }, 300);
            });
            textarea.addEventListener('scroll', syncScroll);
            textarea.addEventListener('input', function(e) {
                refreshEditor();
            });


            textarea.addEventListener('scroll', syncScroll);

            textarea.addEventListener('keydown', function(e) {
                if (e.key === 'Tab') {
                    e.preventDefault();
                    const start = this.selectionStart;
                    const end = this.selectionEnd;

                    const newValue = this.value.substring(0, start) + 
                                    " ".repeat(settings.tabSize) + 
                                    this.value.substring(end);

                    this.value = newValue;
                    this.selectionStart = this.selectionEnd = start + settings.tabSize;
                    
                    refreshEditor();
                }
            });
        }
    }

    function resizeEditor(width, height) {
        wrapper.style.width = width;
        wrapper.style.height = height;
        refreshEditor(true);
    }

    function setLanguage(language) {
        settings.language = language;
        code.className = ` + "`+`language-${language}`+`" + `;
        refreshEditor(true);
    }

    function setValue(value) {
        const scrollTop = textarea.scrollTop;
        const scrollLeft = textarea.scrollLeft;
        textarea.value = value;
        saveState();
        refreshEditor(true);
        // Restore scroll position after refresh
        textarea.scrollTop = scrollTop;
        textarea.scrollLeft = scrollLeft;
        syncScroll();
    }

    function setTheme(theme) {
        wrapper.className = ` + "`" + `code-editor-wrapper ${theme}` + "`" + `;
        if (theme === 'dark') {
            document.documentElement.style.setProperty('--prism-background', 'rgba(0,0,0,0.6)');
            document.documentElement.style.setProperty('--prism-color', '#f8f8f2');
            document.getElementById('syntaxTheme').href = 'https://cdnjs.cloudflare.com/ajax/libs/prism/1.24.1/themes/prism-tomorrow.min.css';
        } else {
            document.documentElement.style.setProperty('--prism-background', '#f5f2f0');
            document.documentElement.style.setProperty('--prism-color', '#333');
            document.getElementById('syntaxTheme').href = 'https://cdnjs.cloudflare.com/ajax/libs/prism/1.24.1/themes/prism.min.css';
        }
        refreshEditor();
    }

    function setSnapshotMode(isSnapshot) {
        settings.snapshot = isSnapshot;
        wrapper.classList.toggle('snapshot', isSnapshot);
        textarea.readOnly = isSnapshot;
    }

    // Initialize
    setSnapshotMode(settings.snapshot);
    textarea.value = settings.value;
    refreshEditor();
    saveState();

    // Public methods
     Object.assign(container, {
        textarea,
        getValue: () => textarea.value,
        setValue,
        on: (event, callback) => textarea.addEventListener(event, callback),
        refresh: refreshEditor,
        setTheme,
        setLanguage,
		snapshot: (b) => { setSnapshotMode(b); refreshEditor(); },
        setOutput: (messages, type = 'info') => updateOutput(messages, type),
        clearOutput: () => updateOutput([]),
        resize: resizeEditor,
        undo,
        redo,
        insertPair
    });
}    
    
`
