package view

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/bamgoo/bamgoo"
	. "github.com/bamgoo/base"
)

func init() {
	module.RegisterDriver(bamgoo.DEFAULT, &defaultDriver{})
}

type (
	defaultDriver struct{}

	defaultConnection struct {
		instance *Instance
	}

	defaultParser struct {
		connection *defaultConnection
		body       Body

		engine *template.Template
		layout string
		path   string
		model  Map
		render string

		title       string
		author      string
		description string
		keywords    string
		metas       []string
		styles      []string
		scripts     []string
	}
)

func (d *defaultDriver) Connect(inst *Instance) (Connection, error) {
	return &defaultConnection{instance: inst}, nil
}

func (c *defaultConnection) Open() error {
	return nil
}

func (c *defaultConnection) Health() (Health, error) {
	return Health{Workload: 0}, nil
}

func (c *defaultConnection) Close() error {
	return nil
}

func (c *defaultConnection) Parse(body Body) (string, error) {
	parser := c.newParser(body)
	return parser.Parse()
}

func (c *defaultConnection) newParser(body Body) *defaultParser {
	parser := &defaultParser{connection: c, body: body}
	parser.metas = []string{}
	parser.styles = []string{}
	parser.scripts = []string{}

	helpers := template.FuncMap{}
	for k, v := range body.Helpers {
		helpers[k] = v
	}

	helpers["layout"] = parser.layoutHelper
	helpers["title"] = parser.titleHelper
	helpers["author"] = parser.authorHelper
	helpers["keywords"] = parser.keywordsHelper
	helpers["description"] = parser.descriptionHelper
	helpers["body"] = parser.bodyHelper
	helpers["render"] = parser.renderHelper
	helpers["meta"] = parser.metaHelper
	helpers["metas"] = parser.metasHelper
	helpers["style"] = parser.styleHelper
	helpers["styles"] = parser.stylesHelper
	helpers["script"] = parser.scriptHelper
	helpers["scripts"] = parser.scriptsHelper

	cfg := c.instance.Config
	parser.engine = template.New("default").Delims(cfg.Left, cfg.Right).Funcs(helpers)
	return parser
}

func (p *defaultParser) Parse() (string, error) {
	return p.layoutParse()
}

func (p *defaultParser) layoutParse() (string, error) {
	bodyText, err := p.bodyParse(p.body.View, p.body.Model)
	if err != nil {
		return "", err
	}

	if p.layout == "" {
		return bodyText, nil
	}

	if p.model == nil {
		p.model = Map{}
	}
	p.render = bodyText

	viewName := ""
	layoutHtml := ""
	if strings.Contains(p.layout, "\n") {
		viewName = bamgoo.Generate("layout")
		layoutHtml = p.layout
	} else {
		filename, err := p.findLayoutFile(p.layout)
		if err != nil {
			return "", err
		}
		bts, err := os.ReadFile(filename)
		if err != nil {
			return "", fmt.Errorf("layout %s read error", p.layout)
		}
		viewName = path.Base(filename)
		layoutHtml = string(bts)
	}

	engine, _ := p.engine.Clone()
	tpl, err := engine.New(viewName).Parse(layoutHtml)
	if err != nil {
		return "", fmt.Errorf("layout %s parse error: %v", viewName, err)
	}

	buf := bytes.NewBuffer(make([]byte, 0))
	data := Map{}
	for k, v := range p.body.Data {
		data[k] = v
	}
	data["model"] = p.model

	if err := tpl.Execute(buf, data); err != nil {
		return "", fmt.Errorf("layout %s parse error: %v", viewName, err)
	}
	return buf.String(), nil
}

func (p *defaultParser) bodyParse(name string, args ...Any) (string, error) {
	var bodyModel Any
	if len(args) > 0 {
		bodyModel = args[0]
	}

	viewName := ""
	html := ""
	if strings.Contains(name, "\n") {
		viewName = bamgoo.Generate("view")
		html = name
	} else {
		filename, err := p.findBodyFile(name)
		if err != nil {
			return "", err
		}
		p.path = filepath.Dir(filename)

		bts, err := os.ReadFile(filename)
		if err != nil {
			return "", fmt.Errorf("view %s read error", name)
		}
		viewName = path.Base(filename)
		html = string(bts)
	}

	engine, _ := p.engine.Clone()
	tpl, err := engine.New(viewName).Parse(html)
	if err != nil {
		return "", fmt.Errorf("view %s parse error: %v", viewName, err)
	}

	buf := bytes.NewBuffer(make([]byte, 0))
	data := Map{}
	for k, v := range p.body.Data {
		data[k] = v
	}
	data["model"] = bodyModel

	if err := tpl.Execute(buf, data); err != nil {
		return "", fmt.Errorf("view %s parse error: %v", viewName, err)
	}
	return buf.String(), nil
}

func (p *defaultParser) renderParse(name string, args ...Any) (string, error) {
	model := Any(Map{})
	if len(args) > 0 {
		model = args[0]
	}

	viewName := ""
	html := ""
	if strings.Contains(name, "\n") {
		viewName = bamgoo.Generate("render")
		html = name
	} else {
		filename, err := p.findRenderFile(name)
		if err != nil {
			return "", err
		}
		bts, err := os.ReadFile(filename)
		if err != nil {
			return "", fmt.Errorf("render %s read error", name)
		}
		viewName = path.Base(filename)
		html = string(bts)
	}

	engine, _ := p.engine.Clone()
	tpl := engine.Lookup(viewName)
	if tpl == nil {
		var err error
		tpl, err = engine.New(viewName).Parse(html)
		if err != nil {
			return "", fmt.Errorf("render %s parse error: %v", viewName, err)
		}
	}

	buf := bytes.NewBuffer(make([]byte, 0))
	data := Map{}
	for k, v := range p.body.Data {
		data[k] = v
	}
	data["model"] = model

	if err := tpl.Execute(buf, data); err != nil {
		return "", fmt.Errorf("render %s parse error: %v", viewName, err)
	}
	return buf.String(), nil
}

func (p *defaultParser) findLayoutFile(name string) (string, error) {
	cfg := p.connection.instance.Config
	body := p.body

	candidates := []string{}
	if p.path != "" {
		candidates = append(candidates, fmt.Sprintf("%s/%s.html", p.path, name))
	}
	candidates = append(candidates, fmt.Sprintf("%s/%s/%s.html", cfg.Root, body.Language, name))
	if body.Site != "" {
		candidates = append(candidates, fmt.Sprintf("%s/%s/%s/%s.html", cfg.Root, body.Site, body.Language, name))
		candidates = append(candidates, fmt.Sprintf("%s/%s/%s/%s/%s.html", cfg.Root, body.Site, body.Language, cfg.Shared, name))
	}
	candidates = append(candidates, fmt.Sprintf("%s/%s/%s.html", cfg.Root, body.Language, name))
	if body.Site != "" {
		candidates = append(candidates, fmt.Sprintf("%s/%s/%s/%s.html", cfg.Root, body.Site, cfg.Shared, name))
		candidates = append(candidates, fmt.Sprintf("%s/%s/%s.html", cfg.Root, body.Site, name))
	}
	candidates = append(candidates, fmt.Sprintf("%s/%s/%s.html", cfg.Root, cfg.Shared, name))
	candidates = append(candidates, fmt.Sprintf("%s/%s.html", cfg.Root, name))

	for _, f := range candidates {
		if st, err := os.Stat(f); err == nil && !st.IsDir() {
			return f, nil
		}
	}
	return "", fmt.Errorf("layout %s not exist", name)
}

func (p *defaultParser) findBodyFile(name string) (string, error) {
	cfg := p.connection.instance.Config
	body := p.body

	candidates := []string{}
	if body.Site != "" {
		candidates = append(candidates, fmt.Sprintf("%s/%s/%s/%s.html", cfg.Root, body.Site, body.Language, name))
		candidates = append(candidates, fmt.Sprintf("%s/%s/%s/%s/%s.html", cfg.Root, body.Site, cfg.Shared, body.Language, name))
	}
	candidates = append(candidates, fmt.Sprintf("%s/%s/%s.html", cfg.Root, body.Language, name))
	candidates = append(candidates, fmt.Sprintf("%s/%s/%s/index.html", cfg.Root, body.Language, name))
	candidates = append(candidates, fmt.Sprintf("%s/%s/%s/%s.html", cfg.Root, body.Language, cfg.Shared, name))
	candidates = append(candidates, fmt.Sprintf("%s/%s/%s/%s/index.html", cfg.Root, body.Language, cfg.Shared, name))
	if body.Site != "" {
		candidates = append(candidates, fmt.Sprintf("%s/%s/%s.html", cfg.Root, body.Site, name))
		candidates = append(candidates, fmt.Sprintf("%s/%s/%s/index.html", cfg.Root, body.Site, name))
		candidates = append(candidates, fmt.Sprintf("%s/%s/%s/%s.html", cfg.Root, body.Site, cfg.Shared, name))
	}
	candidates = append(candidates, fmt.Sprintf("%s/%s.html", cfg.Root, name))
	candidates = append(candidates, fmt.Sprintf("%s/%s/index.html", cfg.Root, name))
	candidates = append(candidates, fmt.Sprintf("%s/%s/%s.html", cfg.Root, cfg.Shared, name))
	candidates = append(candidates, fmt.Sprintf("%s/%s/%s/index.html", cfg.Root, cfg.Shared, name))

	for _, f := range candidates {
		if st, err := os.Stat(f); err == nil && !st.IsDir() {
			return f, nil
		}
	}
	return "", fmt.Errorf("view %s not exist", name)
}

func (p *defaultParser) findRenderFile(name string) (string, error) {
	cfg := p.connection.instance.Config
	body := p.body

	candidates := []string{}
	if p.path != "" {
		candidates = append(candidates, fmt.Sprintf("%s/%s.html", p.path, name))
	}
	if body.Site != "" {
		candidates = append(candidates, fmt.Sprintf("%s/%s/%s/%s/%s.html", cfg.Root, body.Site, body.Language, cfg.Shared, name))
		candidates = append(candidates, fmt.Sprintf("%s/%s/%s/%s.html", cfg.Root, body.Site, body.Language, name))
	}
	candidates = append(candidates, fmt.Sprintf("%s/%s/%s/%s.html", cfg.Root, body.Language, cfg.Shared, name))
	candidates = append(candidates, fmt.Sprintf("%s/%s/%s.html", cfg.Root, body.Language, name))
	if body.Site != "" {
		candidates = append(candidates, fmt.Sprintf("%s/%s/%s/%s.html", cfg.Root, body.Site, cfg.Shared, name))
		candidates = append(candidates, fmt.Sprintf("%s/%s/%s.html", cfg.Root, body.Site, name))
	}
	candidates = append(candidates, fmt.Sprintf("%s/%s/%s.html", cfg.Root, cfg.Shared, name))
	candidates = append(candidates, fmt.Sprintf("%s/%s.html", cfg.Root, name))

	for _, f := range candidates {
		if st, err := os.Stat(f); err == nil && !st.IsDir() {
			return f, nil
		}
	}
	return "", fmt.Errorf("render %s not exist", name)
}

func (p *defaultParser) layoutHelper(name string, vals ...Any) string {
	models := []Map{}
	for _, v := range vals {
		switch vv := v.(type) {
		case Map:
			models = append(models, vv)
		case string:
			m := Map{}
			if err := json.Unmarshal([]byte(vv), &m); err == nil {
				models = append(models, m)
			}
		}
	}

	p.layout = name
	if len(models) > 0 {
		p.model = models[0]
	} else {
		p.model = Map{}
	}
	return ""
}

func (p *defaultParser) titleHelper(args ...string) template.HTML {
	if len(args) > 0 {
		p.title = args[0]
		return ""
	}
	return template.HTML(p.title)
}

func (p *defaultParser) authorHelper(args ...string) template.HTML {
	if len(args) > 0 {
		p.author = args[0]
		return ""
	}
	return template.HTML(p.author)
}

func (p *defaultParser) keywordsHelper(args ...string) template.HTML {
	if len(args) > 0 {
		p.keywords = args[0]
		return ""
	}
	return template.HTML(p.keywords)
}

func (p *defaultParser) descriptionHelper(args ...string) template.HTML {
	if len(args) > 0 {
		p.description = args[0]
		return ""
	}
	return template.HTML(p.description)
}

func (p *defaultParser) bodyHelper() template.HTML {
	return template.HTML(p.render)
}

func (p *defaultParser) renderHelper(name string, vals ...Any) template.HTML {
	out, err := p.renderParse(name, vals...)
	if err != nil {
		return template.HTML(fmt.Sprintf("render error: %v", err))
	}
	return template.HTML(out)
}

func (p *defaultParser) metaHelper(name, content string, https ...bool) string {
	isHTTP := false
	if len(https) > 0 {
		isHTTP = https[0]
	}
	if isHTTP {
		p.metas = append(p.metas, fmt.Sprintf(`<meta http-equiv="%v" content="%v" />`, name, content))
	} else {
		p.metas = append(p.metas, fmt.Sprintf(`<meta name="%v" content="%v" />`, name, content))
	}
	return ""
}

func (p *defaultParser) metasHelper() template.HTML {
	if len(p.metas) == 0 {
		return ""
	}
	return template.HTML(strings.Join(p.metas, "\n"))
}

func (p *defaultParser) styleHelper(path string, args ...string) string {
	media := ""
	if len(args) > 0 {
		media = args[0]
	}
	if media == "" {
		p.styles = append(p.styles, fmt.Sprintf(`<link type="text/css" rel="stylesheet" href="%v" />`, path))
	} else {
		p.styles = append(p.styles, fmt.Sprintf(`<link type="text/css" rel="stylesheet" href="%v" media="%v" />`, path, media))
	}
	return ""
}

func (p *defaultParser) stylesHelper() template.HTML {
	if len(p.styles) == 0 {
		return ""
	}
	return template.HTML(strings.Join(p.styles, "\n"))
}

func (p *defaultParser) scriptHelper(path string, args ...string) string {
	t := "text/javascript"
	if len(args) > 0 && args[0] != "" {
		t = args[0]
	}
	p.scripts = append(p.scripts, fmt.Sprintf(`<script type="%v" src="%v"></script>`, t, path))
	return ""
}

func (p *defaultParser) scriptsHelper() template.HTML {
	if len(p.scripts) == 0 {
		return ""
	}
	return template.HTML(strings.Join(p.scripts, "\n"))
}
