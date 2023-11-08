package commands

import (
	"context"
	"fmt"
	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
	"github.com/expectedsh/kitcat/pkg/kitcat-cli/utils"
	"github.com/mkideal/cli"
	"gopkg.in/yaml.v3"
	"os"
	"path"
)

type genDockerComposeFlags struct {
	cli.Helper

	Name string `cli:"name" usage:"name of your project" dft:"local-dev"`
}

var availableServices = []string{
	"pg",
	"redis",
	"mysql",
	"mailhog",
}

var GenDockerCompose = &cli.Command{
	Name:    "docker-compose",
	Aliases: []string{"dc"},
	Desc:    "this command generate docker compose file with services for kitcat framework",
	Argv:    func() interface{} { return new(genDockerComposeFlags) },
	Fn: func(ctx *cli.Context) error {
		p := ctx.Argv().(*genDockerComposeFlags)
		args := ctx.Args()
		if len(args) == 0 {
			fmt.Println("usage: kitcat-cli generate docker-compose [services...]")
			fmt.Println("You need to specify at least one service")
			fmt.Println("Available services: ")
			for _, service := range availableServices {
				fmt.Println(" - ", service)
			}

			return nil
		}

		if err := genDockerComposeFunc(args, *p); err != nil {
			return utils.Err(err)
		}

		return nil
	},
}

func genDockerComposeFunc(services []string, p genDockerComposeFlags) error {
	project := &types.Project{}

	cwd, err := utils.FindGoModPath()
	if err != nil {
		return err
	}

	if !utils.FileExist(path.Join(cwd, "docker-compose.yml")) {
		project.Name = loader.NormalizeProjectName(p.Name)
	} else {
		project, err = loader.LoadWithContext(context.Background(), types.ConfigDetails{
			WorkingDir:  cwd,
			ConfigFiles: []types.ConfigFile{{Filename: path.Join(cwd, "docker-compose.yml")}},
			Environment: nil,
		})
		if err != nil {
			return err
		}
	}

	hasService := func(n string) bool {
		for _, service := range project.Services {
			if service.Name == n {
				return true
			}
		}

		return false
	}

	for _, service := range services {
		if hasService(service) {
			continue
		}

		if project.Volumes == nil {
			project.Volumes = make(map[string]types.VolumeConfig)
		}

		if project.Services == nil {
			project.Services = make([]types.ServiceConfig, 0)
		}

		switch service {
		case "pg":
			project.Volumes["pg_data"] = types.VolumeConfig{
				Name: "pg_data",
				External: types.External{
					External: false,
				},
			}

			project.Services = append(project.Services, types.ServiceConfig{
				Name:          service,
				ContainerName: "pg",
				Image:         "postgres:16-alpine",
				Environment: types.NewMappingWithEquals([]string{
					"POSTGRES_USER=postgres",
					"POSTGRES_PASSWORD=postgres",
					"POSTGRES_DB=postgres",
				}),
				Ports: []types.ServicePortConfig{
					{
						Mode:      "ingress",
						Target:    5432,
						Published: "5444",
						Protocol:  "tcp",
					},
				},
				Volumes: []types.ServiceVolumeConfig{
					{
						Type:   "volume",
						Source: "pg_data",
						Target: "/var/lib/postgresql/data",
						Volume: &types.ServiceVolumeVolume{},
					},
				},
			})
		case "redis":
			project.Volumes["redis_data"] = types.VolumeConfig{
				Name: "redis_data",
				External: types.External{
					External: false,
				},
			}
			project.Services = append(project.Services, types.ServiceConfig{
				Name:          service,
				ContainerName: "redis",
				Image:         "redis:7-alpine",
				Ports: []types.ServicePortConfig{
					{
						Mode:      "ingress",
						Target:    6380,
						Published: "6333",
					},
				},
				Volumes: []types.ServiceVolumeConfig{
					{
						Type:   "volume",
						Source: "redis_data",
						Target: "/var/lib/redis",
					},
				},
			})
		case "mysql":
			project.Volumes["mysql_data"] = types.VolumeConfig{
				Name: "mysql_data",
				External: types.External{
					External: false,
				},
			}
			project.Services = append(project.Services, types.ServiceConfig{
				Name:          service,
				ContainerName: "mysql",
				Image:         "mysql:8.2.0",
				Environment: types.NewMappingWithEquals([]string{
					"MYSQL_DATABASE=db",
					"MYSQL_USER=mysql",
					"MYSQL_PASSWORD=mysql",
					"MYSQL_ROOT_PASSWORD=root_mysql",
				}),
				Ports: []types.ServicePortConfig{
					{
						Mode:      "ingress",
						Target:    3306,
						Published: "3444",
					},
				},
				Volumes: []types.ServiceVolumeConfig{
					{
						Type:   "volume",
						Source: "mysql_data",
						Target: "/var/lib/mysql",
					},
				},
			})
		case "mailhog":
			project.Services = append(project.Services, types.ServiceConfig{
				Name:          service,
				ContainerName: "mailhog",
				Image:         "mailhog/mailhog",
				Logging: &types.LoggingConfig{
					Driver: "none",
				},
				Ports: []types.ServicePortConfig{
					{
						Mode:      "ingress",
						Target:    1025,
						Published: "1025",
					},
					{
						Mode:      "ingress",
						Target:    8025,
						Published: "8025",
					},
				},
			})
		}
	}

	out, err := yaml.Marshal(project)
	if err != nil {
		return err
	}

	if err := os.WriteFile(path.Join(cwd, "docker-compose.yml"), out, 0644); err != nil {
		return err
	}

	return nil

}
