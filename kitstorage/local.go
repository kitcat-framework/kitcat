package kitstorage

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/expectedsh/kitcat"
	"github.com/expectedsh/kitcat/kitweb"
	"github.com/spf13/viper"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
)

//go:embed filebrowser.gohtml
var fileBrowser string

type LocalFileSystemConfig struct {
	BasePath                      string `cfg:"base_path"`
	PathStorage                   string `cfg:"path_storage"`
	AllowFileBrowser              bool   `cfg:"allow_file_browser"`
	ShowPrivateFilesInFileBrowser bool   `cfg:"show_private_files_in_file_browser"`
}

func (l *LocalFileSystemConfig) InitConfig(prefix string) kitcat.ConfigUnmarshal {
	prefix = fmt.Sprintf("%s.kitstorage.file_systems.local", prefix)
	viper.SetDefault(prefix+".base_path", "./.fs_local/")
	viper.SetDefault(prefix+".path_storage", "/storage")
	viper.SetDefault(prefix+".allow_file_browser", true)
	viper.SetDefault(prefix+".show_private_files_in_file_browser", false)

	return kitcat.ConfigUnmarshalHandler(prefix, l, "unable to unmarshal local file system config: %w")
}

func init() {
	kitcat.RegisterConfig(new(LocalFileSystemConfig))
}

type LocalFileSystem struct {
	Config *LocalFileSystemConfig

	filePathFileMetadata string
	filesMetadata        map[string]fileMetadata

	appConfig *kitcat.AppConfig
}

func NewLocalFileSystem(config *LocalFileSystemConfig, appConfig *kitcat.AppConfig) (*LocalFileSystem, error) {
	fmt.Println("CONFIG:", config.BasePath, config.PathStorage)
	localFileSystem := &LocalFileSystem{
		Config:               config,
		filePathFileMetadata: filepath.Join(config.BasePath, ".kitstorage_files_metadata.json"),
		filesMetadata:        make(map[string]fileMetadata),
		appConfig:            appConfig,
	}

	if err := localFileSystem.loadFileMetadata(); err != nil {
		return nil, err
	}

	return localFileSystem, nil
}

func (l LocalFileSystem) loadFileMetadata() error {
	if _, err := os.Stat(l.filePathFileMetadata); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(l.filePathFileMetadata)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &l.filesMetadata)
}

func (l LocalFileSystem) Put(_ context.Context, path string, reader io.Reader, opts ...PutOptionFunc) error {
	putOptions := NewPutOptions()
	for _, opt := range opts {
		opt(putOptions)
	}

	fmd := fileMetadata{Public: putOptions.public}

	fullPath := filepath.Join(l.Config.BasePath, path)
	fmt.Println(fullPath)
	fmt.Println(filepath.Dir(fullPath))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, reader)
	if err != nil {
		return err
	}

	l.addFileMetadata(fullPath, fmd)

	return nil
}

func (l LocalFileSystem) Get(_ context.Context, path string) (io.Reader, error) {
	fullPath := filepath.Join(l.Config.BasePath, path)
	return os.Open(fullPath)
}

func (l LocalFileSystem) Exists(_ context.Context, path string) (bool, error) {
	fullPath := filepath.Join(l.Config.BasePath, path)
	_, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

func (l LocalFileSystem) Delete(_ context.Context, path string) error {
	fullPath := filepath.Join(l.Config.BasePath, path)
	return os.Remove(fullPath)
}

func (l LocalFileSystem) GetURL(_ context.Context, path string) (string, error) {
	return fmt.Sprintf("http://%s/%s", l.appConfig.Host, filepath.Join(l.Config.BasePath, path)), nil
}

func (l LocalFileSystem) ListFiles(_ context.Context, path string, recursive bool) ([]string, error) {
	fullPath := filepath.Join(l.Config.BasePath, path)
	var files []string

	err := filepath.Walk(fullPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !recursive && info.IsDir() && filePath != fullPath {
			return filepath.SkipDir
		}

		relativePath, err := filepath.Rel(l.Config.BasePath, filePath)
		if err != nil {
			return err
		}

		files = append(files, relativePath)
		return nil
	})

	return files, err
}

type fileMetadata struct {
	Public bool
}

func (l LocalFileSystem) addFileMetadata(path string, metadata fileMetadata) {
	l.filesMetadata[path] = metadata
	_ = l.writeFileMetadata()
}

func (l LocalFileSystem) removeFileMetadata(path string) {
	delete(l.filesMetadata, path)
	_ = l.writeFileMetadata()
}

func (l LocalFileSystem) writeFileMetadata() error {
	marshal, err := json.Marshal(l.filesMetadata)
	if err != nil {
		return err
	}

	err = os.WriteFile(l.filePathFileMetadata, marshal, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (l LocalFileSystem) isDir(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fileInfo.IsDir()
}

func (l LocalFileSystem) Routes(r *kitweb.Router) {
	publicHandler := http.StripPrefix(l.Config.PathStorage, http.FileServer(http.Dir(l.Config.BasePath)))
	r.RawRouter().PathPrefix(l.Config.PathStorage).Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pathWithoutPrefix := r.URL.Path[len(l.Config.PathStorage):]

		if l.Config.AllowFileBrowser && r.URL.Path == l.Config.PathStorage || l.isDir(filepath.Join(l.Config.BasePath,
			pathWithoutPrefix)) {
			l.renderFileBrowser(pathWithoutPrefix, w, r)
			return
		}

		filePath := filepath.Join(l.Config.BasePath, pathWithoutPrefix)
		meta := l.filesMetadata[filePath]

		// check if the file is public or if the request is for the file metadata
		if !meta.Public || filePath == l.filePathFileMetadata {
			http.NotFound(w, r)
			return
		}

		publicHandler.ServeHTTP(w, r)
	}))

}

func (l LocalFileSystem) renderFileBrowser(pathWithoutPrefix string, w http.ResponseWriter, r *http.Request) {
	type File struct {
		Name    string
		Size    int64
		Mode    os.FileMode
		ModTime string
		Path    string
		IsDir   bool
	}

	files, err := os.ReadDir(filepath.Join(l.Config.BasePath, pathWithoutPrefix))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var fileList []File
	for _, file := range files {
		filePath := filepath.Join(l.Config.BasePath, pathWithoutPrefix, file.Name())
		meta := l.filesMetadata[filePath]

		if !file.IsDir() && (file.Name() == l.filePathFileMetadata || !meta.Public && !l.Config.ShowPrivateFilesInFileBrowser) {
			continue
		}

		fileInfo, err := file.Info()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		f := File{
			Size:    fileInfo.Size(),
			Mode:    fileInfo.Mode(),
			ModTime: fileInfo.ModTime().Format("02 Jan 2006 15:04:05"),
			IsDir:   fileInfo.IsDir(),
			Path:    "/" + filepath.Join("storage", pathWithoutPrefix, file.Name()),
		}

		if f.IsDir {
			f.Name = file.Name() + "/"
		} else {
			f.Name = file.Name()
		}

		fileList = append(fileList, f)
	}

	if pathWithoutPrefix != "" {
		fileList = append(fileList, File{
			Name:  "..",
			IsDir: true,
			Path:  "/" + filepath.Join("storage", filepath.Dir(pathWithoutPrefix)),
		})
	}

	slices.SortFunc(fileList, func(a, b File) int {
		if a.IsDir && !b.IsDir {
			return -1
		}
		if !a.IsDir && b.IsDir {
			return 1
		}

		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}

		return 0
	})

	urlQuery := func(s string) string {
		return template.URLQueryEscaper(s)
	}

	tmpl, err := template.New("filebrowser").Funcs(template.FuncMap{"urlquery": urlQuery}).Parse(fileBrowser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, fileList)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (l LocalFileSystem) Name() string {
	return "local_file_system"
}
