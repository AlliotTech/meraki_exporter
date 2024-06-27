export PATH := justfile_directory() + "/env/bin:" + env_var("PATH")

# Recipes
@default:
  just --list

build:
  docker build . -t ghcr.io/alliottech/meraki_exporter:latest

up: 
  just build && docker compose up -d 

down: 
  docker compose down

log: 
  docker compose logs -f
