### kitcat new <name> (--here)

Generate a project with this project struct :

- cmd/app/main.go
- internal/
- internal/example/web,module.go
- internal/types/interfaces,events,routes.go
- migrations/schema.sql,atlas.hcl
- config/prod,staging,test.yaml
- views/public,templates,partials,layout
- views/package.json
- views/esbuild.mjs,tsconfig.js ...
- go.mod
- Taskfile.yml
- Dockerfile
- docker-compose.yml
- .gitignore
- .env
- .env.dist

### list de commande de generation

For each generate command you can use --dir to specify a directory
Otherwise the command will generate in the package 


`kitcat g model <name>`
generate model in internal/types/model_<name>.go 
have --dir option to specify a directory (default internal/types)

`kitcat g consumer <pkg> <name>`
generate consumer in internal/<pkg>/consumer_<name>.go
generate a event param and event name in internal/types/events.go at the end of file
have --dir option to specify a directory (default internal/<pkg>)

`kitcat g http-handler <pkg>`
generate http handler in internal/<pkg>/http_handler.go
alias : `kitcat g http-h <pkg>`
have a --dir option to specify a directory (default internal)
have a --name option to specify a name for the handler -> http_handler_<name>.go (default struct HTTPHandler, default file http_handler.go)

`kitcat g http-middleware <pkg> <name>`
generate http middleware in internal/<pkg>/http_middleware_<name>.go
alias : `kitcat g http-m <name>` 
have a --dir option to specify a directory (default internal)
have a --handler option to specify a struct name (default HTTPHandler) 

`kitcat g interface <pkg> <struct>`
extract functions of a type and output in internal/types/interfaces.go
will find the type in internal/<pkg>/*.go
have a --dir option to specify a directory
have a --file option to specify a file

`kitcat g docker-compose <dependencies...>`
update docker-compose.yml with dependencies provided

`kitcat g env <envname> <outfile>`
output a .env file with envname variables required ($ENV or ${ENV} in .yml file)

`kitcat g route <pkg> <name>`
generate a route in internal/<pkg>/http_handler.go
alias : `kitcat g r <pkg> <name> <method> <path>`
have a --dir option to specify a directory (default internal)
have a --file option to specify a file (default http_handler.go)
have a --handler option to specify a handler name (default HTTPHandler)

`kitcat g routes`
update/create internal/types/routes.go with deduced routes from http handlers Routes method
it will be used for templates in order to do for instance : 
`<a href="{{ .Routes.GetProductID "123456" }}">Home</a>`

