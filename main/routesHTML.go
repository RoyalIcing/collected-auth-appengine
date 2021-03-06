package main

import (
	"bufio"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"
)

const (
	assetsHeadHTML = `
<link rel="stylesheet" href="/public/main.7b36b102.css">
<script defer src="https://unpkg.com/stimulus@1.0.1/dist/stimulus.umd.js"></script>
<link href="//cdn.jsdelivr.net/npm/graphiql@0.11.11/graphiql.css" rel="stylesheet" />
<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/codemirror/5.23.0/theme/solarized.css" />
`
	assetsBeforeBodyCloseHTML = `<script src="/public/frontend.fe3cdceb.js"></script>`
)

func viewErrorMessage(errorMessage string, w *bufio.Writer) {
	w.WriteString(`<p class="py-1 px-2 bg-white text-red">` + errorMessage + "</p>")
}

type htmlHandlerOptions struct {
	form                   bool
	dynamicElementsEnabled map[string]bool
	bodyClass              string
}

func writeDynamicElementsScript(w http.ResponseWriter, dynamicElementsEnabled map[string]bool) {
	if len(dynamicElementsEnabled) == 0 {
		return
	}

	t := template.Must(template.New("dynamicElementsScript").Parse(`
<script>
document.addEventListener("DOMContentLoaded", () => {
const app = Stimulus.Application.start();

{{if .posts}}
app.register('posts', class extends Stimulus.Controller {
	static get targets() {
		return [ 'post', 'replyHolder' ];
	}

	beginReply({ target: button }) {
		const actions = button.closest('[data-target="posts.actions"]');
		const createReplyForm = actions.querySelector('[data-target="posts.createReplyForm"]');
		const createForm = this.targets.find('createForm'); // this.createFormTarget;
		createReplyForm.innerHTML = createForm.innerHTML;
	}
	
	markdownInputChanged({ target: { value } }) {
		const isCommand = value[0] === '/';
		const isMarkdownHeading = value[0] === '#' && value[1] === ' ';
		const isGraphQLQuery = /^query\s+.*{/.test(value);
		this.changeSubmitMode(isCommand ? 'run' : isMarkdownHeading ? 'draft' : isGraphQLQuery ? 'graphQLQuery' : 'submit');
	}

	changeSubmitMode(mode) {
		this.targets.find('submitPostButton').classList.toggle('hidden', mode !== 'submit');
		this.targets.find('runCommandButton').classList.toggle('hidden', mode !== 'run');
		this.targets.find('beginDraftButton').classList.toggle('hidden', mode !== 'draft');
		this.targets.find('runGraphQLQueryButton').classList.toggle('hidden', mode !== 'graphQLQuery');

		const mainTextareaEl = this.targets.find('mainTextarea');
		mainTextareaEl.classList.toggle('font-mono', mode === 'run' || mode === 'graphQLQuery');
		mainTextareaEl.classList.toggle('focus:h-screen', mode === 'draft');
		mainTextareaEl.classList.toggle('text-lg', mode === 'draft');
		mainTextareaEl.classList.toggle('border-purple', mode === 'draft');
	}
});
{{end}}
{{if .developer}}
app.register('developer', class extends Stimulus.Controller {
	static get targets() {
		return [ 'queryCode' ];
	}

	runQuery({ target: button }) {
		const queryCodeEl = this.targets.find('queryCode'); // this.queryCodeTarget;
		const resultEl = this.targets.find('result');
		resultEl.textContent = "Loading…";
		fetch('/graphql', {
			method: 'POST',
			body: JSON.stringify({
				query: queryCodeEl.textContent
			})
		})
			.then(res => res.json())
			.then(json => {
				resultEl.textContent = JSON.stringify(json, null, 2);
			});
	}
});
{{end}}

});
</script>
`))

	t.Execute(w, dynamicElementsEnabled)
}

// WithHTMLHeaders adds HTTP headers
func WithHTMLHeaders(f http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("Content-Type", "text/html; charset=utf-8")
		header.Set("X-Content-Type-Options", "nosniff")

		f(w, r)
	})
}

func htmlHeadStart(w io.Writer) {
	io.WriteString(w, `<!doctype html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
`)

	io.WriteString(w, assetsHeadHTML)

}

func htmlHeadEndBodyStart(w io.Writer, options struct{ bodyClass string }) {
	io.WriteString(w, `
<style>
.grid-1\/3-2\/3 {
	display: grid;
	grid-template-columns: 33.333% 66.667%;
}
.grid-column-gap-1 {
	grid-column-gap: 0.25rem;
}
.grid-row-gap-1 {
	grid-row-gap: 0.25rem;
}
</style>
`)

	io.WriteString(w, `
<script>
window.collectedTasks = [];
</script>
`)

	io.WriteString(w, `</head>`)

	io.WriteString(w, `<body class="`+options.bodyClass+`">`)
}

func htmlBodyEnd(w io.Writer) {
	io.WriteString(w, assetsBeforeBodyCloseHTML)
	io.WriteString(w, "</body></html>")
}

func WithHTMLTemplate(f http.HandlerFunc, options htmlHandlerOptions) http.HandlerFunc {
	return WithHTMLHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("Content-Type", "text/html; charset=utf-8")
		header.Set("X-Content-Type-Options", "nosniff")

		var formErr error
		if options.form {
			formErr = r.ParseForm()
		}

		htmlHeadStart(w)
		io.WriteString(w, "<title>Collected</title>")
		htmlHeadEndBodyStart(w, struct{ bodyClass string }{bodyClass: ""})

		if formErr != nil {
			w.WriteHeader(400)
			io.WriteString(w, "Invalid form request: "+formErr.Error())
		} else {
			f(w, r)
		}

		writeDynamicElementsScript(w, options.dynamicElementsEnabled)

		htmlBodyEnd(w)
	}))
}

// ViewModel is the base view model
type ViewModel struct {
	Title string
}

type viewSectionWriter struct {
	w            *bufio.Writer
	outerTagName string
	outerClasses []string
	innerClasses []string
}

func (section *viewSectionWriter) class(class string) *viewSectionWriter {
	section.outerClasses = append(section.outerClasses, class)
	return section
}

func (section *viewSectionWriter) innerClass(class string) *viewSectionWriter {
	section.innerClasses = append(section.innerClasses, class)
	return section
}

func (section *viewSectionWriter) innerSlim() *viewSectionWriter {
	return section.innerClass("max-w-md mx-auto")
}

func (section *viewSectionWriter) write(f func(w io.Writer)) {
	section.w.WriteString(`<` + section.outerTagName + ` class="` + strings.Join(section.outerClasses, " ") + `">`)
	section.w.WriteString(`<div class="` + strings.Join(section.innerClasses, " ") + `">`)
	f(section.w)
	section.w.WriteString("</div>")
	section.w.WriteString(`</` + section.outerTagName + `>`)
}

func (section *viewSectionWriter) writeHTMLString(html string) {
	section.write(func(w io.Writer) {
		io.WriteString(w, html)
	})
}

func (section *viewSectionWriter) writeTemplate(source string, data interface{}) {
	t := template.New("section")

	t = t.Funcs(template.FuncMap{
		"props": func() map[string]interface{} {
			return make(map[string]interface{})
		},
		"setURL": func(url string, props map[string]interface{}) map[string]interface{} {
			props["URL"] = url
			return props
		},
		"setText": func(text string, props map[string]interface{}) map[string]interface{} {
			props["Text"] = text
			return props
		},
		"setColor": func(colorName string, props map[string]interface{}) map[string]interface{} {
			props["Color"] = colorName
			return props
		},
		"setIsSubmit": func(props map[string]interface{}) map[string]interface{} {
			props["ButtonType"] = "submit"
			return props
		},
		"button": func(props map[string]interface{}) template.HTML {
			color, ok := props["Color"].(string)
			if !ok {
				color = "blue"
			}
			text := props["Text"].(string)
			buttonType, ok := props["ButtonType"].(string)
			if !ok {
				buttonType = "button"
			}

			return template.HTML(`
<button type="` + buttonType + `" class="mt-2 px-4 py-2 font-bold text-white bg-` + color + `-dark border border-` + color + `-darker rounded shadow no-underline hover:bg-` + color + ` hover:border-` + color + `-dark">` + text + `</button>
`)
		},
		"buttonLink": func(props map[string]interface{}) template.HTML {
			url := props["URL"].(string)
			color, ok := props["Color"].(string)
			if !ok {
				color = "blue"
			}
			text := props["Text"].(string)

			return template.HTML(`
<a href="` + url + `" class="mt-2 px-4 py-2 font-bold text-white bg-` + color + `-dark border border-` + color + `-darker rounded shadow no-underline hover:bg-` + color + ` hover:border-` + color + `-dark">` + text + `</a>
`)
		},
		"setInputFormName": func(name string, props map[string]interface{}) map[string]interface{} {
			props["InputFormName"] = name
			return props
		},
		"setLabel": func(label string, props map[string]interface{}) map[string]interface{} {
			props["Label"] = label
			return props
		},
		"fieldWithLabel": func(props map[string]interface{}) template.HTML {
			inputFormName := props["InputFormName"].(string)
			label := props["Label"].(string)

			return template.HTML(`
<label class="block my-2">
	<span class="font-bold">` + label + `</span>
	<input name="` + inputFormName + `" class="block w-full mt-1 p-2 bg-grey-lightest border border-grey rounded shadow-inner">
</label>
`)
		},
	})

	t = template.Must(t.Parse(source))

	section.write(func(w io.Writer) {
		t.Execute(section.w, data)
	})
}

// ViewPage renders a naked HTML page with provided main content
func (vm ViewModel) ViewPage(w io.Writer, viewHeader func(addSection func(outerTagName string) *viewSectionWriter), viewMainContent func(addSection func(outerTagName string) *viewSectionWriter)) {
	bw := bufio.NewWriter(w)
	defer bw.Flush()

	htmlHeadStart(bw)
	io.WriteString(bw, "<title>"+template.HTMLEscapeString(vm.Title)+"</title>")
	htmlHeadEndBodyStart(bw, struct{ bodyClass string }{bodyClass: ""})

	addSection := func(outerTagName string) *viewSectionWriter {
		return &viewSectionWriter{
			w:            bw,
			outerTagName: outerTagName,
		}
	}

	viewHeader(addSection)

	bw.WriteString(`<main>`)
	viewMainContent(addSection)
	bw.WriteString(`</main>`)

	htmlBodyEnd(bw)
}

// OrgViewModel models viewing an org
type OrgViewModel struct {
	OrgSlug string
}

// HTMLURL builds a URL to a org’s home page
func (m OrgViewModel) HTMLURL() string {
	return fmt.Sprintf("/org:%s", m.OrgSlug)
}

// HTMLChannelsURL builds a URL to a org’s channels
func (m OrgViewModel) HTMLChannelsURL() string {
	return fmt.Sprintf("/org:%s/channels", m.OrgSlug)
}

func (m OrgViewModel) viewNav(w *bufio.Writer) {
	t := template.Must(template.New("nav").Parse(`
<nav class="text-white bg-black">
<div class="max-w-md mx-auto flex flex-col sm:flex-row items-center sm:items-start leading-normal">
<strong class="py-1">
	<a href="{{.HTMLURL}}" class="no-underline hover:underline text-white">{{.OrgSlug}}</a>
</strong>
</div>
</nav>
`))

	t.Execute(w, m)
}

// ViewPage renders a page with navigation and provided main content
func (m OrgViewModel) ViewPage(w io.Writer, viewMainContent func(viewSection func(wide bool, viewInner func(w *bufio.Writer)))) {
	sw := bufio.NewWriter(w)
	defer sw.Flush()

	m.viewNav(sw)

	viewSection := func(wide bool, viewInner func(w *bufio.Writer)) {
		if wide {
			sw.WriteString(`<div class="">`)
		} else {
			sw.WriteString(`<div class="max-w-md mx-auto">`)
		}
		viewInner(sw)
		sw.WriteString(`</div>`)
	}

	sw.WriteString(`<main>`)
	viewMainContent(viewSection)
	sw.WriteString(`</main>`)
}

// ChannelViewModel models viewing a channel within an org
type ChannelViewModel struct {
	Org         OrgViewModel
	ChannelSlug string
}

// Channel makes a model for
func (m OrgViewModel) Channel(channelSlug string) ChannelViewModel {
	return ChannelViewModel{
		Org:         m,
		ChannelSlug: channelSlug,
	}
}

// HTMLPostsURL builds a URL to a channel’s posts web page
func (m ChannelViewModel) HTMLPostsURL() string {
	return fmt.Sprintf("/org:%s/channel:%s/posts", m.Org.OrgSlug, m.ChannelSlug)
}

// HTMLPostURL builds a URL to a post
func (m ChannelViewModel) HTMLPostURL(postID string) string {
	return fmt.Sprintf("/org:%s/channel:%s/posts/%s", m.Org.OrgSlug, m.ChannelSlug, postID)
}

// HTMLPostChildPostsURL builds a URL to a post’s child posts web page
func (m ChannelViewModel) HTMLPostChildPostsURL(postID string) string {
	return fmt.Sprintf("/org:%s/channel:%s/posts/%s/posts", m.Org.OrgSlug, m.ChannelSlug, postID)
}

// ViewHeader renders the nav for a channel
func (m ChannelViewModel) ViewHeader(fontSize string, w *bufio.Writer) {
	w.WriteString(fmt.Sprintf(`
<header class="pt-4 pb-3 bg-indigo-darker">
	<div class="max-w-md mx-auto">
		<div class="mx-2 md:mx-0 flex flex-wrap flex-col sm:flex-row items-center sm:items-start sm:justify-between">
			<h1 class="%s min-w-full sm:min-w-0 mb-2 sm:mb-0">
				<a href="%s" class="text-white no-underline hover:underline">💬 %s</a>
			</h1>
			<input type="search" placeholder="Search %s" class="w-64 px-2 py-2 bg-indigo rounded">
		</div>
	</div>
</header>
`, fontSize, m.HTMLPostsURL(), m.ChannelSlug, m.ChannelSlug))
}

// ViewPage renders a page with navigation and provided main content
func (m ChannelViewModel) ViewPage(w io.Writer, viewMainContent func(viewSection func(wide bool, viewInner func(w *bufio.Writer)))) {
	m.Org.ViewPage(w, func(viewSection func(wide bool, viewInner func(sw *bufio.Writer))) {
		viewSection(true, func(sw *bufio.Writer) {
			m.ViewHeader("text-2xl text-center", sw)
		})

		viewMainContent(viewSection)
	})
}

// ToOrgViewModel converts route vars into OrgViewModel
func (v RouteVars) ToOrgViewModel() OrgViewModel {
	return OrgViewModel{
		OrgSlug: v.orgSlug(),
	}
}

// ToChannelViewModel converts route vars into ChannelViewModel
func (v RouteVars) ToChannelViewModel() ChannelViewModel {
	return v.ToOrgViewModel().Channel(v.channelSlug())
}
