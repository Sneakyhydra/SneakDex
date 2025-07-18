# Variables
DC           ?= docker-compose -f docker-compose.yml
DC_PROD      ?= docker-compose -f docker-compose.prod.yml
SERVICE      ?= all
ENV          ?= dev
CMD          ?= up
SCALE        ?=

# Colors
RED          = \033[0;31m
GREEN        = \033[0;32m
YELLOW       = \033[1;33m
BLUE         = \033[0;34m
PURPLE       = \033[0;35m
CYAN         = \033[0;36m
WHITE        = \033[1;37m
NC           = \033[0m

# Services list (used by help & up-all)
SERVICES     := prometheus grafana kafka redis crawler parser indexer app

# General Commands
.PHONY: help up down start stop restart build rebuild logs status reset clean exec

# ==============================================================================
# HELP
# ==============================================================================
help:
	@echo "$(CYAN)Available commands:$(NC)"
	@echo "  $(GREEN)make <command> [SERVICE=name] [ENV=dev|prod] [SCALE=\"service=n …\"]$(NC)"
	@echo ""
	@echo "$(YELLOW)Commands:$(NC)"
	@echo "  up           - Start service(s) with build"
	@echo "  down         - Stop & remove service(s)"
	@echo "  start        - Start without build"
	@echo "  stop         - Stop without removing"
	@echo "  restart      - Restart service(s)"
	@echo "  build        - Build images"
	@echo "  rebuild      - Force rebuild images"
	@echo "  logs         - View logs"
	@echo "  status       - Show status"
	@echo "  reset        - Full reset (down + up)"
	@echo "  clean        - Clean everything incl. volumes"
	@echo "  exec         - Exec shell into container (SERVICE required)"
	@echo ""
	@echo "$(YELLOW)Examples:$(NC)"
	@echo "  make up SERVICE=app ENV=dev"
	@echo "  make up SERVICE=all SCALE=\"parser=3 crawler=2\" ENV=prod"
	@echo "  make logs SERVICE=kafka ENV=prod"
	@echo "  make reset"
	@echo ""
	@echo "$(YELLOW)Available services:$(NC) $(SERVICES)"

# ==============================================================================
# GENERIC TARGETS
# ==============================================================================
define cmd_template
	@echo "$(CYAN)>> $(1)-$(SERVICE)-$(ENV) <<$(NC)"
	$(if $(findstring prod,$(ENV)), $(DC_PROD), $(DC)) $(1) \
	$(if $(filter all,$(SERVICE)),, $(SERVICE)) $(2) \
	$(foreach s,$(SCALE),--scale $(s))
endef

up:
	$(call cmd_template,up,-d --build)

down:
	$(call cmd_template,down,)

start:
	$(call cmd_template,up,-d)

stop:
	$(call cmd_template,stop,)

restart:
	$(call cmd_template,restart,)

build:
	$(call cmd_template,build,)

rebuild:
	$(call cmd_template,build,--no-cache)

logs:
	$(call cmd_template,logs,-f --tail=100)

status:
	$(call cmd_template,ps,)

reset:
	$(MAKE) down SERVICE=$(SERVICE) ENV=$(ENV)
	$(MAKE) up SERVICE=$(SERVICE) ENV=$(ENV)

clean:
	$(call cmd_template,down,-v --remove-orphans)
	docker system prune -f

exec:
ifeq ($(SERVICE),all)
	@echo "$(RED)Error: SERVICE is required for exec$(NC)"
	@exit 1
else
	$(call cmd_template,exec,$(SERVICE) /bin/sh)
endif
