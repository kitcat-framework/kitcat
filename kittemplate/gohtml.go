package kittemplate

import (
	"fmt"
	"github.com/Masterminds/sprig/v3"
	"github.com/expectedsh/kitcat"
	"github.com/spf13/viper"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// GoTemplateExtension is the extension used for go templates
var GoTemplateExtension = "*.gohtml"

// LayoutTemplate is the template used to define a layout
// '%s' will be the name of the layout template
var LayoutTemplate = `{{define "layout" }} {{ template "%s" . }} {{ end }}`

type GoHTMLEngineConfig struct {
	// Folder is where the templates are stored.
	// This is usually where your store your pages
	Folder string `cfg:"folder"`

	// LayoutsFolder is where the layout templates are stored.
	// This is where you can set a base layout for your templates.
	// The name of your layout must be the same as the name of your template.
	// If the name is base.gohtml, the layout must contain {{ define "base" }} to be considered as a layout.
	LayoutsFolder string `cfg:"layouts_folder"`

	// PartialsFolder is where the partial templates are stored.
	// This is where you can set a base layout for your templates.
	// A template must start with {{ define "name" }} to be considered as a partial.
	// The name of the define must the same as the name of the file.
	PartialsFolder string `cfg:"partials_folder"`
}

func (c *GoHTMLEngineConfig) InitConfig(prefix string) kitcat.ConfigUnmarshal {
	prefix = prefix + ".kittemplate.template_engines.gohtml"

	viper.SetDefault(prefix+".folder", "templates")
	viper.SetDefault(prefix+".layouts_folder", "layouts")
	viper.SetDefault(prefix+".partials_folder", "partials")

	return kitcat.ConfigUnmarshalHandler(prefix, c, "unable to unmarshal gohtml template config: %w")
}

func init() {
	kitcat.RegisterConfig(new(GoHTMLEngineConfig))
}

// createTemplateFunc is used to create a template
// In production mode, the template will be cached
type createTemplateFunc func() (*template.Template, error)

type GoHTMLEngine struct {
	*GoHTMLEngineConfig

	templates       map[string]createTemplateFunc
	layoutTemplates map[string]createTemplateFunc
}

func NewGoHTMLTemplateEngine(config *GoHTMLEngineConfig) (*GoHTMLEngine, error) {
	p := &GoHTMLEngine{
		GoHTMLEngineConfig: config,
		layoutTemplates:    make(map[string]createTemplateFunc),
		templates:          make(map[string]createTemplateFunc),
	}

	if err := p.init(); err != nil {
		return nil, err
	}

	return p, nil
}

func (p *GoHTMLEngine) Execute(writer io.Writer, name string, applyOptions ...EngineOption) error {
	opts := EngineOptions{}
	for _, applyOpt := range applyOptions {
		applyOpt(&opts)
	}

	var (
		tmpl *template.Template
		err  error
	)
	if opts.Layout != nil {
		tmpl, err = p.layoutTemplates[filepath.Join(*opts.Layout, name)]()
		if err != nil {
			return fmt.Errorf("failed to execute layout template %s: %w", name, err)
		}
	} else {
		tmpl, err = p.templates[name]()
		if err != nil {
			return fmt.Errorf("failed to execute template %s: %w", name, err)
		}
	}

	if tmpl == nil {
		return fmt.Errorf("template %s not found", name)
	}

	err = tmpl.Execute(writer, opts.Data)
	if err != nil {
		return fmt.Errorf("failed to execute template %s: %w", name, err)
	}

	return nil
}

func (p *GoHTMLEngine) Name() string {
	return "gohtml"
}

func (p *GoHTMLEngine) init() error {

	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	layoutFiles, includeFiles, partialsFiles, err := p.getFiles(dir)
	if err != nil {
		return err
	}

	for _, file := range includeFiles {

		// for each files in the folder we are creating :
		// - a template for each layout including all partials
		// - a template including all partials
		//
		// In this manner you can either use a layout or not in your views

		files := append(partialsFiles, file)
		fileName := fileNameWithoutExtension(filepath.Base(file))

		p.templates[fileName] = func() (*template.Template, error) {
			includeFileTemplate, err := withFunc(template.New(filepath.Base(file))).ParseFiles(files...)
			if err != nil {
				return nil, fmt.Errorf("failed to parse template %s: %w", file, err)
			}

			return includeFileTemplate, nil
		}

		for _, layoutFile := range layoutFiles {
			fileName := filepath.Join(
				fileNameWithoutExtension(filepath.Base(layoutFile)),
				fileNameWithoutExtension(filepath.Base(file)))

			files := append(partialsFiles, layoutFile, file)

			layoutTemplate, err := withFunc(template.New("layout")).Parse(fmt.Sprintf(LayoutTemplate,
				fileNameWithoutExtension(filepath.Base(layoutFile))))

			if err != nil {
				return fmt.Errorf("failed to parse layout template %s for file %s: %w", layoutFile, file, err)
			}

			p.layoutTemplates[fileName] = func() (*template.Template, error) {
				tmpl, err := withFunc(layoutTemplate).ParseFiles(files...)
				if err != nil {
					return nil, fmt.Errorf("failed to parse %s for file %s: %w", layoutFile, file, err)
				}

				return tmpl, nil
			}
		}
	}

	return nil
}

func withFunc(tmpl *template.Template) *template.Template {
	return tmpl.Funcs(sprig.FuncMap())
}

func (p *GoHTMLEngine) getFiles(dir string) (
	layoutFiles []string,
	includeFiles []string,
	partialsFiles []string,
	err error,
) {
	layoutFiles, err = filepath.Glob(filepath.Join(dir, p.Folder, p.LayoutsFolder, GoTemplateExtension))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load layout files for %s: %w", p.LayoutsFolder, err)
	}

	includeFiles, err = filepath.Glob(filepath.Join(dir, p.Folder, GoTemplateExtension))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load include files for %s: %w", p.Folder, err)
	}

	partialsFiles, err = filepath.Glob(filepath.Join(dir, p.Folder, p.PartialsFolder, GoTemplateExtension))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load partials files for %s: %w", p.PartialsFolder, err)
	}

	return layoutFiles, includeFiles, partialsFiles, nil
}

func fileNameWithoutExtension(fileName string) string {
	return strings.TrimSuffix(fileName, filepath.Ext(fileName))
}
