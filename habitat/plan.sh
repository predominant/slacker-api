pkg_name=slacker
pkg_origin=predominant
pkg_version="0.1.0"
pkg_scaffolding="core/scaffolding-go"
scaffolding_go_build_deps=(
  github.com/BurntSushi/toml
  github.com/gorilla/websocket
  github.com/mitchellh/mapstructure
  gopkg.in/gorethink/gorethink.v4
)
pkg_exports=(
  [port]=port
)
pkg_exposes=(port)
pkg_binds_optional=(
  [database]="host port"
)
