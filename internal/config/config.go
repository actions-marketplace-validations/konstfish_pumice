package config

import (
	"flag"
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type PageConfig struct {
	Title string `yaml:"title"`
}

type SiteConfig struct {
	URL     string `yaml:"url"`
	OGImage string `yaml:"ogImage"`
	Favicon string `yaml:"favicon"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
}

type GiscusConfig struct {
	Repo       string `yaml:"repo"`
	RepoID     string `yaml:"repoId"`
	Category   string `yaml:"category"`
	CategoryID string `yaml:"categoryId"`
	Mapping    string `yaml:"mapping"`
	Theme      string `yaml:"theme"`
	Lang       string `yaml:"lang"`
}

type FooterLink struct {
	Text string `yaml:"text"`
	URL  string `yaml:"url"`
}

type FooterConfig struct {
	Text  string       `yaml:"text"`
	Links []FooterLink `yaml:"links"`
}

type Config struct {
	ContentDir string       `yaml:"contentDir"`
	BuildDir   string       `yaml:"buildDir"`
	StaticDir  string       `yaml:"staticDir"`
	Site       SiteConfig   `yaml:"site"`
	Page       PageConfig   `yaml:"page"`
	Footer     FooterConfig `yaml:"footer"`
	Server     ServerConfig `yaml:"server"`
	Giscus     GiscusConfig `yaml:"giscus"`
	Serve      bool         `yaml:"-"` // CLI only
	ConfigPath string       `yaml:"-"` // CLI only
}

// defaults returns a Config with all default values.
func defaults() Config {
	return Config{
		ContentDir: "content",
		BuildDir:   "build",
		Page: PageConfig{
			Title: "Pumice",
		},
		Server: ServerConfig{
			Port: "8080",
		},
		ConfigPath: "pumice.yaml",
	}
}

type Manager struct {
	config *Config
	flags  *flag.FlagSet

	// raw CLI values — used to detect what was explicitly set
	flagContentDir *string
	flagBuildDir   *string
	flagStaticDir  *string
	flagPageTitle  *string
	flagPort       *string
	flagServe      *bool
	flagConfig     *string
}

func NewManager() *Manager {
	m := &Manager{
		flags: flag.NewFlagSet("pumice", flag.ExitOnError),
	}

	d := defaults()

	m.flagConfig = m.flags.String("config", d.ConfigPath, "path to config file")
	m.flagContentDir = m.flags.String("content", "", "content directory")
	m.flagBuildDir = m.flags.String("build", "", "build output directory")
	m.flagStaticDir = m.flags.String("static", "", "static files directory")
	m.flagPageTitle = m.flags.String("title", "", "site title")
	m.flagPort = m.flags.String("port", "", "port to serve on")
	m.flagServe = m.flags.Bool("serve", false, "serve the built site")

	return m
}

// Load parses CLI args, loads the YAML config file, and merges them.
// Priority: CLI flags > YAML config > defaults.
func (m *Manager) Load(args []string) error {
	if err := m.flags.Parse(args); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}

	// Start with defaults
	cfg := defaults()

	// Layer YAML config on top
	configPath := *m.flagConfig
	if err := m.loadYAML(configPath, &cfg); err != nil {
		return err
	}

	// Layer CLI flags on top (only if explicitly set)
	m.flags.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "content":
			cfg.ContentDir = *m.flagContentDir
		case "build":
			cfg.BuildDir = *m.flagBuildDir
		case "static":
			cfg.StaticDir = *m.flagStaticDir
		case "title":
			cfg.Page.Title = *m.flagPageTitle
		case "port":
			cfg.Server.Port = *m.flagPort
		case "serve":
			cfg.Serve = *m.flagServe
		}
	})

	cfg.ConfigPath = configPath
	m.config = &cfg
	return nil
}

func (m *Manager) loadYAML(path string, cfg *Config) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // No config file, use defaults
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("parsing config file: %w", err)
	}

	return nil
}

func (m *Manager) GetConfig() *Config {
	if m.config == nil {
		d := defaults()
		return &d
	}
	return m.config
}

func (m *Manager) GetContentDir() string {
	return m.GetConfig().ContentDir
}

func (m *Manager) GetBuildDir() string {
	return m.GetConfig().BuildDir
}

func (m *Manager) GetSiteURL() string {
	return m.GetConfig().Site.URL
}

func (m *Manager) GetOGImage() string {
	return m.GetConfig().Site.OGImage
}

func (m *Manager) GetFavicon() string {
	return m.GetConfig().Site.Favicon
}

func (m *Manager) GetStaticDir() string {
	return m.GetConfig().StaticDir
}

func (m *Manager) GetPageTitle() string {
	return m.GetConfig().Page.Title
}

func (m *Manager) GetPort() string {
	return m.GetConfig().Server.Port
}

func (m *Manager) GetFooter() FooterConfig {
	return m.GetConfig().Footer
}

func (m *Manager) GetGiscus() GiscusConfig {
	return m.GetConfig().Giscus
}

func (m *Manager) ShouldServe() bool {
	return m.GetConfig().Serve
}
