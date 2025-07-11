# Variables
DC = docker-compose -f docker-compose.yml
DC_PROD = docker-compose -f docker-compose.prod.yml

# Colors for output
RED = \033[0;31m
GREEN = \033[0;32m
YELLOW = \033[1;33m
BLUE = \033[0;34m
PURPLE = \033[0;35m
CYAN = \033[0;36m
WHITE = \033[1;37m
NC = \033[0m # No Color

# General Commands
.PHONY: help \
        up-dev down-dev start-dev stop-dev restart-dev build-dev rebuild-dev logs-dev status-dev reset-dev clean-dev \
        up-prod down-prod start-prod stop-prod restart-prod build-prod rebuild-prod logs-prod status-prod reset-prod clean-prod \
        up-prometheus down-prometheus start-prometheus stop-prometheus restart-prometheus build-prometheus rebuild-prometheus logs-prometheus status-prometheus reset-prometheus clean-prometheus \
        up-prometheus-prod down-prometheus-prod start-prometheus-prod stop-prometheus-prod restart-prometheus-prod build-prometheus-prod rebuild-prometheus-prod logs-prometheus-prod status-prometheus-prod reset-prometheus-prod clean-prometheus-prod \
        up-grafana down-grafana start-grafana stop-grafana restart-grafana build-grafana rebuild-grafana logs-grafana status-grafana reset-grafana clean-grafana \
        up-grafana-prod down-grafana-prod start-grafana-prod stop-grafana-prod restart-grafana-prod build-grafana-prod rebuild-grafana-prod logs-grafana-prod status-grafana-prod reset-grafana-prod clean-grafana-prod \
        up-kafka down-kafka start-kafka stop-kafka restart-kafka build-kafka rebuild-kafka logs-kafka status-kafka reset-kafka clean-kafka \
        up-redis down-redis start-redis stop-redis restart-redis build-redis rebuild-redis logs-redis status-redis reset-redis clean-redis \
        up-crawler down-crawler start-crawler stop-crawler restart-crawler build-crawler rebuild-crawler logs-crawler status-crawler reset-crawler clean-crawler \
        up-parser down-parser start-parser stop-parser restart-parser build-parser rebuild-parser logs-parser status-parser reset-parser clean-parser \
        up-indexer down-indexer start-indexer stop-indexer restart-indexer build-indexer rebuild-indexer logs-indexer status-indexer reset-indexer clean-indexer \
        up-app down-app start-app stop-app restart-app build-app rebuild-app logs-app status-app reset-app clean-app \
        up-crawler-prod down-crawler-prod start-crawler-prod stop-crawler-prod restart-crawler-prod build-crawler-prod rebuild-crawler-prod logs-crawler-prod status-crawler-prod reset-crawler-prod clean-crawler-prod \
        up-parser-prod down-parser-prod start-parser-prod stop-parser-prod restart-parser-prod build-parser-prod rebuild-parser-prod logs-parser-prod status-parser-prod reset-parser-prod clean-parser-prod \
        up-indexer-prod down-indexer-prod start-indexer-prod stop-indexer-prod restart-indexer-prod build-indexer-prod rebuild-indexer-prod logs-indexer-prod status-indexer-prod reset-indexer-prod clean-indexer-prod \
        up-app-prod down-app-prod start-app-prod stop-app-prod restart-app-prod build-app-prod rebuild-app-prod logs-app-prod status-app-prod reset-app-prod clean-app-prod \
        exec-kafka exec-redis \
		exec-crawler exec-parser exec-indexer exec-app \
		exec-crawler-prod exec-parser-prod exec-indexer-prod exec-app-prod \
        kafka-topics kafka-create-topics kafka-list-topics kafka-delete-topics \
		redis-cli redis-flushall \
        install-deps update-deps

# Help
help:
	@echo "$(CYAN)╔════════════════════════════════════════════════════════════════╗$(NC)"
	@echo "$(CYAN)║                    SneakDex Makefile Commands                   ║$(NC)"
	@echo "$(CYAN)╚════════════════════════════════════════════════════════════════╝$(NC)"
	@echo ""
	@echo "$(GREEN)🛠  DEVELOPMENT ENVIRONMENT$(NC)"
	@echo "  $(YELLOW)make up-dev$(NC)					Start all dev services (with build)"
	@echo "  $(YELLOW)make down-dev$(NC)				Stop and remove dev containers (keep volumes)"
	@echo "  $(YELLOW)make start-dev$(NC)				Start dev services (no build)"
	@echo "  $(YELLOW)make stop-dev$(NC)				Stop dev services (keep containers)"
	@echo "  $(YELLOW)make restart-dev$(NC)				Restart dev services"
	@echo "  $(YELLOW)make build-dev$(NC)				Build dev images"
	@echo "  $(YELLOW)make rebuild-dev$(NC)				Force rebuild dev images"
	@echo "  $(YELLOW)make logs-dev$(NC)				View all dev logs"
	@echo "  $(YELLOW)make status-dev$(NC)				Show dev services status"
	@echo "  $(YELLOW)make reset-dev$(NC)				Full reset of dev environment"
	@echo "  $(YELLOW)make clean-dev$(NC)				Clean dev containers and volumes"
	@echo ""
	@echo "$(GREEN)🚀 PRODUCTION ENVIRONMENT$(NC)"
	@echo "  $(BLUE)make up-prod$(NC)					Start all prod services"
	@echo "  $(BLUE)make down-prod$(NC)				Stop and remove prod containers (keep volumes)"
	@echo "  $(BLUE)make start-prod$(NC)				Start prod services (no build)"
	@echo "  $(BLUE)make stop-prod$(NC)				Stop prod services (keep containers)"
	@echo "  $(BLUE)make restart-prod$(NC)				Restart prod services"
	@echo "  $(BLUE)make build-prod$(NC)				Build prod images"
	@echo "  $(BLUE)make rebuild-prod$(NC)				Force rebuild prod images"
	@echo "  $(BLUE)make logs-prod$(NC)				View all prod logs"
	@echo "  $(BLUE)make status-prod$(NC)				Show prod services status"
	@echo "  $(BLUE)make reset-prod$(NC)				Full reset of prod environment"
	@echo "  $(BLUE)make clean-prod$(NC)				Clean prod containers and volumes"
	@echo ""
	@echo "$(GREEN)🗃️  INFRASTRUCTURE SERVICES$(NC)"
	@echo "  $(YELLOW)make up-prometheus$(NC)					Start all prometheus services (with build)"
	@echo "  $(YELLOW)make down-prometheus$(NC)				Stop and remove prometheus containers (keep volumes)"
	@echo "  $(YELLOW)make start-prometheus$(NC)				Start prometheus services (no build)"
	@echo "  $(YELLOW)make stop-prometheus$(NC)				Stop prometheus services (keep containers)"
	@echo "  $(YELLOW)make restart-prometheus$(NC)				Restart prometheus services"
	@echo "  $(YELLOW)make build-prometheus$(NC)				Build prometheus images"
	@echo "  $(YELLOW)make rebuild-prometheus$(NC)				Force rebuild prometheus images"
	@echo "  $(YELLOW)make logs-prometheus$(NC)				View all prometheus logs"
	@echo "  $(YELLOW)make status-prometheus$(NC)				Show prometheus services status"
	@echo "  $(YELLOW)make reset-prometheus$(NC)				Full reset of prometheus environment"
	@echo "  $(YELLOW)make clean-prometheus$(NC)				Clean prometheus containers and volumes"
	@echo ""
	@echo "  $(YELLOW)make up-prometheus-prod$(NC)					Start all prometheus-prod services (with build)"
	@echo "  $(YELLOW)make down-prometheus-prod$(NC)				Stop and remove prometheus-prod containers (keep volumes)"
	@echo "  $(YELLOW)make start-prometheus-prod$(NC)				Start prometheus-prod services (no build)"
	@echo "  $(YELLOW)make stop-prometheus-prod$(NC)				Stop prometheus-prod services (keep containers)"
	@echo "  $(YELLOW)make restart-prometheus-prod$(NC)				Restart prometheus-prod services"
	@echo "  $(YELLOW)make build-prometheus-prod$(NC)				Build prometheus-prod images"
	@echo "  $(YELLOW)make rebuild-prometheus-prod$(NC)				Force rebuild prometheus-prod images"
	@echo "  $(YELLOW)make logs-prometheus-prod$(NC)				View all prometheus-prod logs"
	@echo "  $(YELLOW)make status-prometheus-prod$(NC)				Show prometheus-prod services status"
	@echo "  $(YELLOW)make reset-prometheus-prod$(NC)				Full reset of prometheus-prod environment"
	@echo "  $(YELLOW)make clean-prometheus-prod$(NC)				Clean prometheus-prod containers and volumes"
	@echo ""
	@echo "  $(YELLOW)make up-grafana$(NC)					Start all grafana services (with build)"
	@echo "  $(YELLOW)make down-grafana$(NC)				Stop and remove grafana containers (keep volumes)"
	@echo "  $(YELLOW)make start-grafana$(NC)				Start grafana services (no build)"
	@echo "  $(YELLOW)make stop-grafana$(NC)				Stop grafana services (keep containers)"
	@echo "  $(YELLOW)make restart-grafana$(NC)				Restart grafana services"
	@echo "  $(YELLOW)make build-grafana$(NC)				Build grafana images"
	@echo "  $(YELLOW)make rebuild-grafana$(NC)				Force rebuild grafana images"
	@echo "  $(YELLOW)make logs-grafana$(NC)				View all grafana logs"
	@echo "  $(YELLOW)make status-grafana$(NC)				Show grafana services status"
	@echo "  $(YELLOW)make reset-grafana$(NC)				Full reset of grafana environment"
	@echo "  $(YELLOW)make clean-grafana$(NC)				Clean grafana containers and volumes"
	@echo ""
	@echo "  $(YELLOW)make up-grafana-prod$(NC)					Start all grafana-prod services (with build)"
	@echo "  $(YELLOW)make down-grafana-prod$(NC)				Stop and remove grafana-prod containers (keep volumes)"
	@echo "  $(YELLOW)make start-grafana-prod$(NC)				Start grafana-prod services (no build)"
	@echo "  $(YELLOW)make stop-grafana-prod$(NC)				Stop grafana-prod services (keep containers)"
	@echo "  $(YELLOW)make restart-grafana-prod$(NC)				Restart grafana-prod services"
	@echo "  $(YELLOW)make build-grafana-prod$(NC)				Build grafana-prod images"
	@echo "  $(YELLOW)make rebuild-grafana-prod$(NC)				Force rebuild grafana-prod images"
	@echo "  $(YELLOW)make logs-grafana-prod$(NC)				View all grafana-prod logs"
	@echo "  $(YELLOW)make status-grafana-prod$(NC)				Show grafana-prod services status"
	@echo "  $(YELLOW)make reset-grafana-prod$(NC)				Full reset of grafana-prod environment"
	@echo "  $(YELLOW)make clean-grafana-prod$(NC)				Clean grafana-prod containers and volumes"
	@echo ""
	@echo "  $(YELLOW)make up-kafka$(NC)					Start all kafka services (with build)"
	@echo "  $(YELLOW)make down-kafka$(NC)				Stop and remove kafka containers (keep volumes)"
	@echo "  $(YELLOW)make start-kafka$(NC)				Start kafka services (no build)"
	@echo "  $(YELLOW)make stop-kafka$(NC)				Stop kafka services (keep containers)"
	@echo "  $(YELLOW)make restart-kafka$(NC)				Restart kafka services"
	@echo "  $(YELLOW)make build-kafka$(NC)				Build kafka images"
	@echo "  $(YELLOW)make rebuild-kafka$(NC)				Force rebuild kafka images"
	@echo "  $(YELLOW)make logs-kafka$(NC)				View all kafka logs"
	@echo "  $(YELLOW)make status-kafka$(NC)				Show kafka services status"
	@echo "  $(YELLOW)make reset-kafka$(NC)				Full reset of kafka environment"
	@echo "  $(YELLOW)make clean-kafka$(NC)				Clean kafka containers and volumes"
	@echo ""
	@echo "  $(YELLOW)make up-redis$(NC)					Start all redis services (with build)"
	@echo "  $(YELLOW)make down-redis$(NC)				Stop and remove redis containers (keep volumes)"
	@echo "  $(YELLOW)make start-redis$(NC)				Start redis services (no build)"
	@echo "  $(YELLOW)make stop-redis$(NC)				Stop redis services (keep containers)"
	@echo "  $(YELLOW)make restart-redis$(NC)				Restart redis services"
	@echo "  $(YELLOW)make build-redis$(NC)				Build redis images"
	@echo "  $(YELLOW)make rebuild-redis$(NC)				Force rebuild redis images"
	@echo "  $(YELLOW)make logs-redis$(NC)				View all redis logs"
	@echo "  $(YELLOW)make status-redis$(NC)				Show redis services status"
	@echo "  $(YELLOW)make reset-redis$(NC)				Full reset of redis environment"
	@echo "  $(YELLOW)make clean-redis$(NC)				Clean redis containers and volumes"
	@echo ""
	@echo "$(GREEN)🔧 INDIVIDUAL SERVICE MANAGEMENT$(NC)"
	@echo "  $(YELLOW)make up-crawler$(NC)					Start all crawler services (with build)"
	@echo "  $(YELLOW)make down-crawler$(NC)				Stop and remove crawler containers (keep volumes)"
	@echo "  $(YELLOW)make start-crawler$(NC)				Start crawler services (no build)"
	@echo "  $(YELLOW)make stop-crawler$(NC)				Stop crawler services (keep containers)"
	@echo "  $(YELLOW)make restart-crawler$(NC)				Restart crawler services"
	@echo "  $(YELLOW)make build-crawler$(NC)				Build crawler images"
	@echo "  $(YELLOW)make rebuild-crawler$(NC)				Force rebuild crawler images"
	@echo "  $(YELLOW)make logs-crawler$(NC)				View all crawler logs"
	@echo "  $(YELLOW)make status-crawler$(NC)				Show crawler services status"
	@echo "  $(YELLOW)make reset-crawler$(NC)				Full reset of crawler environment"
	@echo "  $(YELLOW)make clean-crawler$(NC)				Clean crawler containers and volumes"
	@echo ""
	@echo "  $(YELLOW)make up-parser$(NC)					Start all parser services (with build)"
	@echo "  $(YELLOW)make down-parser$(NC)				Stop and remove parser containers (keep volumes)"
	@echo "  $(YELLOW)make start-parser$(NC)				Start parser services (no build)"
	@echo "  $(YELLOW)make stop-parser$(NC)				Stop parser services (keep containers)"
	@echo "  $(YELLOW)make restart-parser$(NC)				Restart parser services"
	@echo "  $(YELLOW)make build-parser$(NC)				Build parser images"
	@echo "  $(YELLOW)make rebuild-parser$(NC)				Force rebuild parser images"
	@echo "  $(YELLOW)make logs-parser$(NC)				View all parser logs"
	@echo "  $(YELLOW)make status-parser$(NC)				Show parser services status"
	@echo "  $(YELLOW)make reset-parser$(NC)				Full reset of parser environment"
	@echo "  $(YELLOW)make clean-parser$(NC)				Clean parser containers and volumes"
	@echo ""
	@echo "  $(YELLOW)make up-indexer$(NC)					Start all indexer services (with build)"
	@echo "  $(YELLOW)make down-indexer$(NC)				Stop and remove indexer containers (keep volumes)"
	@echo "  $(YELLOW)make start-indexer$(NC)				Start indexer services (no build)"
	@echo "  $(YELLOW)make stop-indexer$(NC)				Stop indexer services (keep containers)"
	@echo "  $(YELLOW)make restart-indexer$(NC)				Restart indexer services"
	@echo "  $(YELLOW)make build-indexer$(NC)				Build indexer images"
	@echo "  $(YELLOW)make rebuild-indexer$(NC)				Force rebuild indexer images"
	@echo "  $(YELLOW)make logs-indexer$(NC)				View all indexer logs"
	@echo "  $(YELLOW)make status-indexer$(NC)				Show indexer services status"
	@echo "  $(YELLOW)make reset-indexer$(NC)				Full reset of indexer environment"
	@echo "  $(YELLOW)make clean-indexer$(NC)				Clean indexer containers and volumes"
	@echo ""
	@echo "  $(YELLOW)make up-app$(NC)					Start all app services (with build)"
	@echo "  $(YELLOW)make down-app$(NC)				Stop and remove app containers (keep volumes)"
	@echo "  $(YELLOW)make start-app$(NC)				Start app services (no build)"
	@echo "  $(YELLOW)make stop-app$(NC)				Stop app services (keep containers)"
	@echo "  $(YELLOW)make restart-app$(NC)				Restart app services"
	@echo "  $(YELLOW)make build-app$(NC)				Build app images"
	@echo "  $(YELLOW)make rebuild-app$(NC)				Force rebuild app images"
	@echo "  $(YELLOW)make logs-app$(NC)				View all app logs"
	@echo "  $(YELLOW)make status-app$(NC)				Show app services status"
	@echo "  $(YELLOW)make reset-app$(NC)				Full reset of app environment"
	@echo "  $(YELLOW)make clean-app$(NC)				Clean app containers and volumes"
	@echo ""
	@echo "$(GREEN)🔧 INDIVIDUAL SERVICE MANAGEMENT (PROD)$(NC)"
	@echo "  $(YELLOW)make up-crawler-prod$(NC)					Start all crawler-prod services (with build)"
	@echo "  $(YELLOW)make down-crawler-prod$(NC)				Stop and remove crawler-prod containers (keep volumes)"
	@echo "  $(YELLOW)make start-crawler-prod$(NC)				Start crawler-prod services (no build)"
	@echo "  $(YELLOW)make stop-crawler-prod$(NC)				Stop crawler-prod services (keep containers)"
	@echo "  $(YELLOW)make restart-crawler-prod$(NC)				Restart crawler-prod services"
	@echo "  $(YELLOW)make build-crawler-prod$(NC)				Build crawler-prod images"
	@echo "  $(YELLOW)make rebuild-crawler-prod$(NC)				Force rebuild crawler-prod images"
	@echo "  $(YELLOW)make logs-crawler-prod$(NC)				View all crawler-prod logs"
	@echo "  $(YELLOW)make status-crawler-prod$(NC)				Show crawler-prod services status"
	@echo "  $(YELLOW)make reset-crawler-prod$(NC)				Full reset of crawler-prod environment"
	@echo "  $(YELLOW)make clean-crawler-prod$(NC)				Clean crawler-prod containers and volumes"
	@echo ""
	@echo "  $(YELLOW)make up-parser-prod$(NC)					Start all parser-prod services (with build)"
	@echo "  $(YELLOW)make down-parser-prod$(NC)				Stop and remove parser-prod containers (keep volumes)"
	@echo "  $(YELLOW)make start-parser-prod$(NC)				Start parser-prod services (no build)"
	@echo "  $(YELLOW)make stop-parser-prod$(NC)				Stop parser-prod services (keep containers)"
	@echo "  $(YELLOW)make restart-parser-prod$(NC)				Restart parser-prod services"
	@echo "  $(YELLOW)make build-parser-prod$(NC)				Build parser-prod images"
	@echo "  $(YELLOW)make rebuild-parser-prod$(NC)				Force rebuild parser-prod images"
	@echo "  $(YELLOW)make logs-parser-prod$(NC)				View all parser-prod logs"
	@echo "  $(YELLOW)make status-parser-prod$(NC)				Show parser-prod services status"
	@echo "  $(YELLOW)make reset-parser-prod$(NC)				Full reset of parser-prod environment"
	@echo "  $(YELLOW)make clean-parser-prod$(NC)				Clean parser-prod containers and volumes"
	@echo ""
	@echo "  $(YELLOW)make up-indexer-prod$(NC)					Start all indexer-prod services (with build)"
	@echo "  $(YELLOW)make down-indexer-prod$(NC)				Stop and remove indexer-prod containers (keep volumes)"
	@echo "  $(YELLOW)make start-indexer-prod$(NC)				Start indexer-prod services (no build)"
	@echo "  $(YELLOW)make stop-indexer-prod$(NC)				Stop indexer-prod services (keep containers)"
	@echo "  $(YELLOW)make restart-indexer-prod$(NC)				Restart indexer-prod services"
	@echo "  $(YELLOW)make build-indexer-prod$(NC)				Build indexer-prod images"
	@echo "  $(YELLOW)make rebuild-indexer-prod$(NC)				Force rebuild indexer-prod images"
	@echo "  $(YELLOW)make logs-indexer-prod$(NC)				View all indexer-prod logs"
	@echo "  $(YELLOW)make status-indexer-prod$(NC)				Show indexer-prod services status"
	@echo "  $(YELLOW)make reset-indexer-prod$(NC)				Full reset of indexer-prod environment"
	@echo "  $(YELLOW)make clean-indexer-prod$(NC)				Clean indexer-prod containers and volumes"
	@echo ""
	@echo "  $(YELLOW)make up-app-prod$(NC)					Start all app-prod services (with build)"
	@echo "  $(YELLOW)make down-app-prod$(NC)				Stop and remove app-prod containers (keep volumes)"
	@echo "  $(YELLOW)make start-app-prod$(NC)				Start app-prod services (no build)"
	@echo "  $(YELLOW)make stop-app-prod$(NC)				Stop app-prod services (keep containers)"
	@echo "  $(YELLOW)make restart-app-prod$(NC)				Restart app-prod services"
	@echo "  $(YELLOW)make build-app-prod$(NC)				Build app-prod images"
	@echo "  $(YELLOW)make rebuild-app-prod$(NC)				Force rebuild app-prod images"
	@echo "  $(YELLOW)make logs-app-prod$(NC)				View all app-prod logs"
	@echo "  $(YELLOW)make status-app-prod$(NC)				Show app-prod services status"
	@echo "  $(YELLOW)make reset-app-prod$(NC)				Full reset of app-prod environment"
	@echo "  $(YELLOW)make clean-app-prod$(NC)				Clean app-prod containers and volumes"
	@echo ""
	@echo "$(GREEN)🐚 SHELL ACCESS$(NC)"
	@echo "  $(WHITE)make exec-kafka$(NC)				Shell into Kafka container"
	@echo "  $(WHITE)make exec-redis$(NC)				Shell into Redis container"
	@echo ""
	@echo "  $(WHITE)make exec-crawler$(NC)			Shell into crawler container"
	@echo "  $(WHITE)make exec-parser$(NC)				Shell into parser container"
	@echo "  $(WHITE)make exec-indexer$(NC)			Shell into indexer container"
	@echo "  $(WHITE)make exec-app$(NC)			Shell into app container"
	@echo ""
	@echo "  $(WHITE)make exec-crawler-prod$(NC)				Shell into crawler-prod container"
	@echo "  $(WHITE)make exec-parser-prod$(NC)				Shell into parser-prod container"
	@echo "  $(WHITE)make exec-indexer-prod$(NC)				Shell into indexer-prod container"
	@echo "  $(WHITE)make exec-app-prod$(NC)				Shell into app-prod container"
	@echo ""
	@echo "$(GREEN)📊 KAFKA MANAGEMENT$(NC)"
	@echo "  $(YELLOW)make kafka-topics$(NC)				List all Kafka topics"
	@echo "  $(YELLOW)make kafka-create-topics$(NC)			Create default topics"
	@echo "  $(YELLOW)make kafka-list-topics$(NC)			List topics with details"
	@echo "  $(YELLOW)make kafka-delete-topics$(NC)			Delete all topics"
	@echo ""
	@echo "$(GREEN)📊 REDIS MANAGEMENT$(NC)"
	@echo "  $(YELLOW)make redis-cli$(NC)				Access Redis CLI"
	@echo "  $(YELLOW)make redis-flushall$(NC)				Flush all Redis data"
	@echo ""
	@echo "$(GREEN)📦 DEPENDENCIES (DEV)$(NC)"
	@echo "  $(PURPLE)make install-deps$(NC)				Install/update dependencies"
	@echo "  $(PURPLE)make update-deps$(NC)				Update all dependencies"

# ============================================================================
# DEVELOPMENT ENVIRONMENT
# ============================================================================

up-dev:
	@echo "$(GREEN)🚀 Starting development environment...$(NC)"
	COMPOSE_BAKE=true $(DC) up --build -d

down-dev:
	@echo "$(RED)🛑 Stopping and removing development containers...$(NC)"
	$(DC) down

start-dev:
	@echo "$(GREEN)▶️  Starting development services (no build)...$(NC)"
	COMPOSE_BAKE=true $(DC) up -d

stop-dev:
	@echo "$(YELLOW)⏹️  Stopping development services...$(NC)"
	$(DC) stop

restart-dev:
	@echo "$(BLUE)🔄 Restarting development services...$(NC)"
	$(DC) restart

build-dev:
	@echo "$(BLUE)🔨 Building development images...$(NC)"
	COMPOSE_BAKE=true $(DC) build

rebuild-dev:
	@echo "$(BLUE)🔨 Force rebuilding development images...$(NC)"
	COMPOSE_BAKE=true $(DC) build --no-cache

logs-dev:
	@echo "$(CYAN)📋 Viewing development logs...$(NC)"
	$(DC) logs -f --tail=100

status-dev:
	@echo "$(CYAN)📊 Development services status:$(NC)"
	$(DC) ps

reset-dev:
	@echo "$(RED)🔄 Resetting development environment...$(NC)"
	$(DC) down -v
	COMPOSE_BAKE=true $(DC) up --build -d

clean-dev:
	@echo "$(RED)🧹 Cleaning development environment...$(NC)"
	$(DC) down -v --remove-orphans
	docker system prune -f

# ============================================================================
# PRODUCTION ENVIRONMENT
# ============================================================================

up-prod:
	@echo "$(GREEN)🚀 Starting production environment...$(NC)"
	COMPOSE_BAKE=true $(DC_PROD) up --build -d

down-prod:
	@echo "$(RED)🛑 Stopping and removing production containers...$(NC)"
	$(DC_PROD) down

start-prod:
	@echo "$(GREEN)▶️  Starting production services (no build)...$(NC)"
	COMPOSE_BAKE=true $(DC_PROD) up -d

stop-prod:
	@echo "$(YELLOW)⏹️  Stopping production services...$(NC)"
	$(DC_PROD) stop

restart-prod:
	@echo "$(BLUE)🔄 Restarting production services...$(NC)"
	$(DC_PROD) restart

build-prod:
	@echo "$(BLUE)🔨 Building production images...$(NC)"
	COMPOSE_BAKE=true $(DC_PROD) build

rebuild-prod:
	@echo "$(BLUE)🔨 Force rebuilding production images...$(NC)"
	COMPOSE_BAKE=true $(DC_PROD) build --no-cache

logs-prod:
	@echo "$(CYAN)📋 Viewing production logs...$(NC)"
	$(DC_PROD) logs -f --tail=100

status-prod:
	@echo "$(CYAN)📊 Production services status:$(NC)"
	$(DC_PROD) ps

reset-prod:
	@echo "$(RED)🔄 Resetting production environment...$(NC)"
	$(DC_PROD) down -v
	COMPOSE_BAKE=true $(DC_PROD) up --build -d

clean-prod:
	@echo "$(RED)🧹 Cleaning production environment...$(NC)"
	$(DC_PROD) down -v --remove-orphans
	docker system prune -f

# ============================================================================
# INFRASTRUCTURE SERVICES
# ============================================================================

up-prometheus:
	@echo "$(GREEN)🚀 Starting prometheus...$(NC)"
	COMPOSE_BAKE=true $(DC) up --build -d prometheus

down-prometheus:
	@echo "$(RED)🛑 Stopping and removing prometheus containers...$(NC)"
	$(DC) down prometheus

start-prometheus:
	@echo "$(GREEN)▶️  Starting prometheus services (no build)...$(NC)"
	COMPOSE_BAKE=true $(DC) up -d prometheus

stop-prometheus:
	@echo "$(YELLOW)⏹️  Stopping prometheus services...$(NC)"
	$(DC) stop prometheus

restart-prometheus:
	@echo "$(BLUE)🔄 Restarting prometheus services...$(NC)"
	$(DC) restart prometheus

build-prometheus:
	@echo "$(BLUE)🔨 Building prometheus images...$(NC)"
	COMPOSE_BAKE=true $(DC) build prometheus

rebuild-prometheus:
	@echo "$(BLUE)🔨 Force rebuilding prometheus images...$(NC)"
	COMPOSE_BAKE=true $(DC) build --no-cache prometheus

logs-prometheus:
	@echo "$(CYAN)📋 Viewing prometheus logs...$(NC)"
	$(DC) logs -f --tail=100 prometheus

status-prometheus:
	@echo "$(CYAN)📊 Prometheus services status:$(NC)"
	$(DC) ps prometheus

reset-prometheus:
	@echo "$(RED)🔄 Resetting prometheus environment...$(NC)"
	$(DC) down -v prometheus
	COMPOSE_BAKE=true $(DC) up --build -d prometheus

clean-prometheus:
	@echo "$(RED)🧹 Cleaning prometheus environment...$(NC)"
	$(DC) down -v --remove-orphans prometheus
	docker system prune -f

up-prometheus-prod:
	@echo "$(GREEN)🚀 Starting prometheus-prod...$(NC)"
	COMPOSE_BAKE=true $(DC_PROD) up --build -d prometheus-prod

down-prometheus-prod:
	@echo "$(RED)🛑 Stopping and removing prometheus-prod containers...$(NC)"
	$(DC_PROD) down prometheus-prod

start-prometheus-prod:
	@echo "$(GREEN)▶️  Starting prometheus-prod services (no build)...$(NC)"
	COMPOSE_BAKE=true $(DC_PROD) up -d prometheus-prod

stop-prometheus-prod:
	@echo "$(YELLOW)⏹️  Stopping prometheus-prod services...$(NC)"
	$(DC_PROD) stop prometheus-prod

restart-prometheus-prod:
	@echo "$(BLUE)🔄 Restarting prometheus-prod services...$(NC)"
	$(DC_PROD) restart prometheus-prod

build-prometheus-prod:
	@echo "$(BLUE)🔨 Building prometheus-prod images...$(NC)"
	COMPOSE_BAKE=true $(DC_PROD) build prometheus-prod

rebuild-prometheus-prod:
	@echo "$(BLUE)🔨 Force rebuilding prometheus-prod images...$(NC)"
	COMPOSE_BAKE=true $(DC_PROD) build --no-cache prometheus-prod

logs-prometheus-prod:
	@echo "$(CYAN)📋 Viewing prometheus-prod logs...$(NC)"
	$(DC_PROD) logs -f --tail=100 prometheus-prod

status-prometheus-prod:
	@echo "$(CYAN)📊 Prometheus-prod services status:$(NC)"
	$(DC_PROD) ps prometheus-prod

reset-prometheus-prod:
	@echo "$(RED)🔄 Resetting prometheus-prod environment...$(NC)"
	$(DC_PROD) down -v prometheus-prod
	COMPOSE_BAKE=true $(DC_PROD) up --build -d prometheus-prod

clean-prometheus-prod:
	@echo "$(RED)🧹 Cleaning prometheus-prod environment...$(NC)"
	$(DC_PROD) down -v --remove-orphans prometheus-prod
	docker system prune -f

up-grafana:
	@echo "$(GREEN)🚀 Starting grafana...$(NC)"
	COMPOSE_BAKE=true $(DC) up --build -d grafana

down-grafana:
	@echo "$(RED)🛑 Stopping and removing grafana containers...$(NC)"
	$(DC) down grafana

start-grafana:
	@echo "$(GREEN)▶️  Starting grafana services (no build)...$(NC)"
	COMPOSE_BAKE=true $(DC) up -d grafana

stop-grafana:
	@echo "$(YELLOW)⏹️  Stopping grafana services...$(NC)"
	$(DC) stop grafana

restart-grafana:
	@echo "$(BLUE)🔄 Restarting grafana services...$(NC)"
	$(DC) restart grafana

build-grafana:
	@echo "$(BLUE)🔨 Building grafana images...$(NC)"
	COMPOSE_BAKE=true $(DC) build grafana

rebuild-grafana:
	@echo "$(BLUE)🔨 Force rebuilding grafana images...$(NC)"
	COMPOSE_BAKE=true $(DC) build --no-cache grafana

logs-grafana:
	@echo "$(CYAN)📋 Viewing grafana logs...$(NC)"
	$(DC) logs -f --tail=100 grafana

status-grafana:
	@echo "$(CYAN)📊 grafana services status:$(NC)"
	$(DC) ps grafana

reset-grafana:
	@echo "$(RED)🔄 Resetting grafana environment...$(NC)"
	$(DC) down -v grafana
	COMPOSE_BAKE=true $(DC) up --build -d grafana

clean-grafana:
	@echo "$(RED)🧹 Cleaning grafana environment...$(NC)"
	$(DC) down -v --remove-orphans grafana
	docker system prune -f

up-grafana-prod:
	@echo "$(GREEN)🚀 Starting grafana-prod...$(NC)"
	COMPOSE_BAKE=true $(DC_PROD) up --build -d grafana-prod

down-grafana-prod:
	@echo "$(RED)🛑 Stopping and removing grafana-prod containers...$(NC)"
	$(DC_PROD) down grafana-prod

start-grafana-prod:
	@echo "$(GREEN)▶️  Starting grafana-prod services (no build)...$(NC)"
	COMPOSE_BAKE=true $(DC_PROD) up -d grafana-prod

stop-grafana-prod:
	@echo "$(YELLOW)⏹️  Stopping grafana-prod services...$(NC)"
	$(DC_PROD) stop grafana-prod

restart-grafana-prod:
	@echo "$(BLUE)🔄 Restarting grafana-prod services...$(NC)"
	$(DC_PROD) restart grafana-prod

build-grafana-prod:
	@echo "$(BLUE)🔨 Building grafana-prod images...$(NC)"
	COMPOSE_BAKE=true $(DC_PROD) build grafana-prod

rebuild-grafana-prod:
	@echo "$(BLUE)🔨 Force rebuilding grafana-prod images...$(NC)"
	COMPOSE_BAKE=true $(DC_PROD) build --no-cache grafana-prod

logs-grafana-prod:
	@echo "$(CYAN)📋 Viewing grafana-prod logs...$(NC)"
	$(DC_PROD) logs -f --tail=100 grafana-prod

status-grafana-prod:
	@echo "$(CYAN)📊 grafana-prod services status:$(NC)"
	$(DC_PROD) ps grafana-prod

reset-grafana-prod:
	@echo "$(RED)🔄 Resetting grafana-prod environment...$(NC)"
	$(DC_PROD) down -v grafana-prod
	COMPOSE_BAKE=true $(DC_PROD) up --build -d grafana-prod

clean-grafana-prod:
	@echo "$(RED)🧹 Cleaning grafana-prod environment...$(NC)"
	$(DC_PROD) down -v --remove-orphans grafana-prod
	docker system prune -f

up-kafka:
	@echo "$(GREEN)🚀 Starting kafka...$(NC)"
	COMPOSE_BAKE=true $(DC) up --build -d kafka

down-kafka:
	@echo "$(RED)🛑 Stopping and removing kafka containers...$(NC)"
	$(DC) down kafka

start-kafka:
	@echo "$(GREEN)▶️  Starting kafka services (no build)...$(NC)"
	COMPOSE_BAKE=true $(DC) up -d kafka

stop-kafka:
	@echo "$(YELLOW)⏹️  Stopping kafka services...$(NC)"
	$(DC) stop kafka

restart-kafka:
	@echo "$(BLUE)🔄 Restarting kafka services...$(NC)"
	$(DC) restart kafka

build-kafka:
	@echo "$(BLUE)🔨 Building kafka images...$(NC)"
	COMPOSE_BAKE=true $(DC) build kafka

rebuild-kafka:
	@echo "$(BLUE)🔨 Force rebuilding kafka images...$(NC)"
	COMPOSE_BAKE=true $(DC) build --no-cache kafka

logs-kafka:
	@echo "$(CYAN)📋 Viewing kafka logs...$(NC)"
	$(DC) logs -f --tail=100 kafka

status-kafka:
	@echo "$(CYAN)📊 kafka services status:$(NC)"
	$(DC) ps kafka

reset-kafka:
	@echo "$(RED)🔄 Resetting kafka environment...$(NC)"
	$(DC) down -v kafka
	COMPOSE_BAKE=true $(DC) up --build -d kafka

clean-kafka:
	@echo "$(RED)🧹 Cleaning kafka environment...$(NC)"
	$(DC) down -v --remove-orphans kafka
	docker system prune -f

up-redis:
	@echo "$(GREEN)🚀 Starting redis...$(NC)"
	COMPOSE_BAKE=true $(DC) up --build -d redis

down-redis:
	@echo "$(RED)🛑 Stopping and removing redis containers...$(NC)"
	$(DC) down redis

start-redis:
	@echo "$(GREEN)▶️  Starting redis services (no build)...$(NC)"
	COMPOSE_BAKE=true $(DC) up -d redis

stop-redis:
	@echo "$(YELLOW)⏹️  Stopping redis services...$(NC)"
	$(DC) stop redis

restart-redis:
	@echo "$(BLUE)🔄 Restarting redis services...$(NC)"
	$(DC) restart redis

build-redis:
	@echo "$(BLUE)🔨 Building redis images...$(NC)"
	COMPOSE_BAKE=true $(DC) build redis

rebuild-redis:
	@echo "$(BLUE)🔨 Force rebuilding redis images...$(NC)"
	COMPOSE_BAKE=true $(DC) build --no-cache redis

logs-redis:
	@echo "$(CYAN)📋 Viewing redis logs...$(NC)"
	$(DC) logs -f --tail=100 redis

status-redis:
	@echo "$(CYAN)📊 redis services status:$(NC)"
	$(DC) ps redis

reset-redis:
	@echo "$(RED)🔄 Resetting redis environment...$(NC)"
	$(DC) down -v redis
	COMPOSE_BAKE=true $(DC) up --build -d redis

clean-redis:
	@echo "$(RED)🧹 Cleaning redis environment...$(NC)"
	$(DC) down -v --remove-orphans redis
	docker system prune -f

# ============================================================================
# INDIVIDUAL SERVICE MANAGEMENT
# ============================================================================

up-crawler:
	@echo "$(GREEN)🚀 Starting crawler...$(NC)"
	COMPOSE_BAKE=true $(DC) up --build -d crawler

down-crawler:
	@echo "$(RED)🛑 Stopping and removing crawler containers...$(NC)"
	$(DC) down crawler

start-crawler:
	@echo "$(GREEN)▶️  Starting crawler services (no build)...$(NC)"
	COMPOSE_BAKE=true $(DC) up -d crawler

stop-crawler:
	@echo "$(YELLOW)⏹️  Stopping crawler services...$(NC)"
	$(DC) stop crawler

restart-crawler:
	@echo "$(BLUE)🔄 Restarting crawler services...$(NC)"
	$(DC) restart crawler

build-crawler:
	@echo "$(BLUE)🔨 Building crawler images...$(NC)"
	COMPOSE_BAKE=true $(DC) build crawler

rebuild-crawler:
	@echo "$(BLUE)🔨 Force rebuilding crawler images...$(NC)"
	COMPOSE_BAKE=true $(DC) build --no-cache crawler

logs-crawler:
	@echo "$(CYAN)📋 Viewing crawler logs...$(NC)"
	$(DC) logs -f --tail=100 crawler

status-crawler:
	@echo "$(CYAN)📊 crawler services status:$(NC)"
	$(DC) ps crawler

reset-crawler:
	@echo "$(RED)🔄 Resetting crawler environment...$(NC)"
	$(DC) down -v crawler
	COMPOSE_BAKE=true $(DC) up --build -d crawler

clean-crawler:
	@echo "$(RED)🧹 Cleaning crawler environment...$(NC)"
	$(DC) down -v --remove-orphans crawler
	docker system prune -f

up-parser:
	@echo "$(GREEN)🚀 Starting parser...$(NC)"
	COMPOSE_BAKE=true $(DC) up --build -d parser

down-parser:
	@echo "$(RED)🛑 Stopping and removing parser containers...$(NC)"
	$(DC) down parser

start-parser:
	@echo "$(GREEN)▶️  Starting parser services (no build)...$(NC)"
	COMPOSE_BAKE=true $(DC) up -d parser

stop-parser:
	@echo "$(YELLOW)⏹️  Stopping parser services...$(NC)"
	$(DC) stop parser

restart-parser:
	@echo "$(BLUE)🔄 Restarting parser services...$(NC)"
	$(DC) restart parser

build-parser:
	@echo "$(BLUE)🔨 Building parser images...$(NC)"
	COMPOSE_BAKE=true $(DC) build parser

rebuild-parser:
	@echo "$(BLUE)🔨 Force rebuilding parser images...$(NC)"
	COMPOSE_BAKE=true $(DC) build --no-cache parser

logs-parser:
	@echo "$(CYAN)📋 Viewing parser logs...$(NC)"
	$(DC) logs -f --tail=100 parser

status-parser:
	@echo "$(CYAN)📊 parser services status:$(NC)"
	$(DC) ps parser

reset-parser:
	@echo "$(RED)🔄 Resetting parser environment...$(NC)"
	$(DC) down -v parser
	COMPOSE_BAKE=true $(DC) up --build -d parser

clean-parser:
	@echo "$(RED)🧹 Cleaning parser environment...$(NC)"
	$(DC) down -v --remove-orphans parser
	docker system prune -f

up-indexer:
	@echo "$(GREEN)🚀 Starting indexer...$(NC)"
	COMPOSE_BAKE=true $(DC) up --build -d indexer

down-indexer:
	@echo "$(RED)🛑 Stopping and removing indexer containers...$(NC)"
	$(DC) down indexer

start-indexer:
	@echo "$(GREEN)▶️  Starting indexer services (no build)...$(NC)"
	COMPOSE_BAKE=true $(DC) up -d indexer

stop-indexer:
	@echo "$(YELLOW)⏹️  Stopping indexer services...$(NC)"
	$(DC) stop indexer

restart-indexer:
	@echo "$(BLUE)🔄 Restarting indexer services...$(NC)"
	$(DC) restart indexer

build-indexer:
	@echo "$(BLUE)🔨 Building indexer images...$(NC)"
	COMPOSE_BAKE=true $(DC) build indexer

rebuild-indexer:
	@echo "$(BLUE)🔨 Force rebuilding indexer images...$(NC)"
	COMPOSE_BAKE=true $(DC) build --no-cache indexer

logs-indexer:
	@echo "$(CYAN)📋 Viewing indexer logs...$(NC)"
	$(DC) logs -f --tail=100 indexer

status-indexer:
	@echo "$(CYAN)📊 indexer services status:$(NC)"
	$(DC) ps indexer

reset-indexer:
	@echo "$(RED)🔄 Resetting indexer environment...$(NC)"
	$(DC) down -v indexer
	COMPOSE_BAKE=true $(DC) up --build -d indexer

clean-indexer:
	@echo "$(RED)🧹 Cleaning indexer environment...$(NC)"
	$(DC) down -v --remove-orphans indexer
	docker system prune -f

up-app:
	@echo "$(GREEN)🚀 Starting app...$(NC)"
	COMPOSE_BAKE=true $(DC) up --build -d app

down-app:
	@echo "$(RED)🛑 Stopping and removing app containers...$(NC)"
	$(DC) down app

start-app:
	@echo "$(GREEN)▶️  Starting app services (no build)...$(NC)"
	COMPOSE_BAKE=true $(DC) up -d app

stop-app:
	@echo "$(YELLOW)⏹️  Stopping app services...$(NC)"
	$(DC) stop app

restart-app:
	@echo "$(BLUE)🔄 Restarting app services...$(NC)"
	$(DC) restart app

build-app:
	@echo "$(BLUE)🔨 Building app images...$(NC)"
	COMPOSE_BAKE=true $(DC) build app

rebuild-app:
	@echo "$(BLUE)🔨 Force rebuilding app images...$(NC)"
	COMPOSE_BAKE=true $(DC) build --no-cache app

logs-app:
	@echo "$(CYAN)📋 Viewing app logs...$(NC)"
	$(DC) logs -f --tail=100 app

status-app:
	@echo "$(CYAN)📊 app services status:$(NC)"
	$(DC) ps app

reset-app:
	@echo "$(RED)🔄 Resetting app environment...$(NC)"
	$(DC) down -v app
	COMPOSE_BAKE=true $(DC) up --build -d app

clean-app:
	@echo "$(RED)🧹 Cleaning app environment...$(NC)"
	$(DC) down -v --remove-orphans app
	docker system prune -f

# ============================================================================
# INDIVIDUAL SERVICE MANAGEMENT (PROD)
# ============================================================================

up-crawler-prod:
	@echo "$(GREEN)🚀 Starting crawler-prod...$(NC)"
	COMPOSE_BAKE=true $(DC_PROD) up --build -d crawler-prod

down-crawler-prod:
	@echo "$(RED)🛑 Stopping and removing crawler-prod containers...$(NC)"
	$(DC_PROD) down crawler-prod

start-crawler-prod:
	@echo "$(GREEN)▶️  Starting crawler-prod services (no build)...$(NC)"
	COMPOSE_BAKE=true $(DC_PROD) up -d crawler-prod

stop-crawler-prod:
	@echo "$(YELLOW)⏹️  Stopping crawler-prod services...$(NC)"
	$(DC_PROD) stop crawler-prod

restart-crawler-prod:
	@echo "$(BLUE)🔄 Restarting crawler-prod services...$(NC)"
	$(DC_PROD) restart crawler-prod

build-crawler-prod:
	@echo "$(BLUE)🔨 Building crawler-prod images...$(NC)"
	COMPOSE_BAKE=true $(DC_PROD) build crawler-prod

rebuild-crawler-prod:
	@echo "$(BLUE)🔨 Force rebuilding crawler-prod images...$(NC)"
	COMPOSE_BAKE=true $(DC_PROD) build --no-cache crawler-prod

logs-crawler-prod:
	@echo "$(CYAN)📋 Viewing crawler-prod logs...$(NC)"
	$(DC_PROD) logs -f --tail=100 crawler-prod

status-crawler-prod:
	@echo "$(CYAN)📊 crawler-prod services status:$(NC)"
	$(DC_PROD) ps crawler-prod

reset-crawler-prod:
	@echo "$(RED)🔄 Resetting crawler-prod environment...$(NC)"
	$(DC_PROD) down -v crawler-prod
	COMPOSE_BAKE=true $(DC_PROD) up --build -d crawler-prod

clean-crawler-prod:
	@echo "$(RED)🧹 Cleaning crawler-prod environment...$(NC)"
	$(DC_PROD) down -v --remove-orphans crawler-prod
	docker system prune -f

up-parser-prod:
	@echo "$(GREEN)🚀 Starting parser-prod...$(NC)"
	COMPOSE_BAKE=true $(DC_PROD) up --build -d parser-prod

down-parser-prod:
	@echo "$(RED)🛑 Stopping and removing parser-prod containers...$(NC)"
	$(DC_PROD) down parser-prod

start-parser-prod:
	@echo "$(GREEN)▶️  Starting parser-prod services (no build)...$(NC)"
	COMPOSE_BAKE=true $(DC_PROD) up -d parser-prod

stop-parser-prod:
	@echo "$(YELLOW)⏹️  Stopping parser-prod services...$(NC)"
	$(DC_PROD) stop parser-prod

restart-parser-prod:
	@echo "$(BLUE)🔄 Restarting parser-prod services...$(NC)"
	$(DC_PROD) restart parser-prod

build-parser-prod:
	@echo "$(BLUE)🔨 Building parser-prod images...$(NC)"
	COMPOSE_BAKE=true $(DC_PROD) build parser-prod

rebuild-parser-prod:
	@echo "$(BLUE)🔨 Force rebuilding parser-prod images...$(NC)"
	COMPOSE_BAKE=true $(DC_PROD) build --no-cache parser-prod

logs-parser-prod:
	@echo "$(CYAN)📋 Viewing parser-prod logs...$(NC)"
	$(DC_PROD) logs -f --tail=100 parser-prod

status-parser-prod:
	@echo "$(CYAN)📊 parser-prod services status:$(NC)"
	$(DC_PROD) ps parser-prod

reset-parser-prod:
	@echo "$(RED)🔄 Resetting parser-prod environment...$(NC)"
	$(DC_PROD) down -v parser-prod
	COMPOSE_BAKE=true $(DC_PROD) up --build -d parser-prod

clean-parser-prod:
	@echo "$(RED)🧹 Cleaning parser-prod environment...$(NC)"
	$(DC_PROD) down -v --remove-orphans parser-prod
	docker system prune -f

up-indexer-prod:
	@echo "$(GREEN)🚀 Starting indexer-prod...$(NC)"
	COMPOSE_BAKE=true $(DC_PROD) up --build -d indexer-prod

down-indexer-prod:
	@echo "$(RED)🛑 Stopping and removing indexer-prod containers...$(NC)"
	$(DC_PROD) down indexer-prod

start-indexer-prod:
	@echo "$(GREEN)▶️  Starting indexer-prod services (no build)...$(NC)"
	COMPOSE_BAKE=true $(DC_PROD) up -d indexer-prod

stop-indexer-prod:
	@echo "$(YELLOW)⏹️  Stopping indexer-prod services...$(NC)"
	$(DC_PROD) stop indexer-prod

restart-indexer-prod:
	@echo "$(BLUE)🔄 Restarting indexer-prod services...$(NC)"
	$(DC_PROD) restart indexer-prod

build-indexer-prod:
	@echo "$(BLUE)🔨 Building indexer-prod images...$(NC)"
	COMPOSE_BAKE=true $(DC_PROD) build indexer-prod

rebuild-indexer-prod:
	@echo "$(BLUE)🔨 Force rebuilding indexer-prod images...$(NC)"
	COMPOSE_BAKE=true $(DC_PROD) build --no-cache indexer-prod

logs-indexer-prod:
	@echo "$(CYAN)📋 Viewing indexer-prod logs...$(NC)"
	$(DC_PROD) logs -f --tail=100 indexer-prod

status-indexer-prod:
	@echo "$(CYAN)📊 indexer-prod services status:$(NC)"
	$(DC_PROD) ps indexer-prod

reset-indexer-prod:
	@echo "$(RED)🔄 Resetting indexer-prod environment...$(NC)"
	$(DC_PROD) down -v indexer-prod
	COMPOSE_BAKE=true $(DC_PROD) up --build -d indexer-prod

clean-indexer-prod:
	@echo "$(RED)🧹 Cleaning indexer-prod environment...$(NC)"
	$(DC_PROD) down -v --remove-orphans indexer-prod
	docker system prune -f

up-app-prod:
	@echo "$(GREEN)🚀 Starting app-prod...$(NC)"
	COMPOSE_BAKE=true $(DC_PROD) up --build -d app-prod

down-app-prod:
	@echo "$(RED)🛑 Stopping and removing app-prod containers...$(NC)"
	$(DC_PROD) down app-prod

start-app-prod:
	@echo "$(GREEN)▶️  Starting app-prod services (no build)...$(NC)"
	COMPOSE_BAKE=true $(DC_PROD) up -d app-prod

stop-app-prod:
	@echo "$(YELLOW)⏹️  Stopping app-prod services...$(NC)"
	$(DC_PROD) stop app-prod

restart-app-prod:
	@echo "$(BLUE)🔄 Restarting app-prod services...$(NC)"
	$(DC_PROD) restart app-prod

build-app-prod:
	@echo "$(BLUE)🔨 Building app-prod images...$(NC)"
	COMPOSE_BAKE=true $(DC_PROD) build app-prod

rebuild-app-prod:
	@echo "$(BLUE)🔨 Force rebuilding app-prod images...$(NC)"
	COMPOSE_BAKE=true $(DC_PROD) build --no-cache app-prod

logs-app-prod:
	@echo "$(CYAN)📋 Viewing app-prod logs...$(NC)"
	$(DC_PROD) logs -f --tail=100 app-prod

status-app-prod:
	@echo "$(CYAN)📊 app-prod services status:$(NC)"
	$(DC_PROD) ps app-prod

reset-app-prod:
	@echo "$(RED)🔄 Resetting app-prod environment...$(NC)"
	$(DC_PROD) down -v app-prod
	COMPOSE_BAKE=true $(DC_PROD) up --build -d app-prod

clean-app-prod:
	@echo "$(RED)🧹 Cleaning app-prod environment...$(NC)"
	$(DC_PROD) down -v --remove-orphans app-prod
	docker system prune -f

# ============================================================================
# SHELL ACCESS
# ============================================================================

exec-kafka:
	@echo "$(WHITE)🐚 Entering Kafka container...$(NC)"
	$(DC) exec kafka sh

exec-redis:
	@echo "$(WHITE)🐚 Entering Redis container...$(NC)"
	$(DC) exec redis sh

exec-crawler:
	@echo "$(WHITE)🐚 Entering crawler container...$(NC)"
	$(DC) exec crawler sh

exec-parser:
	@echo "$(WHITE)🐚 Entering parser container...$(NC)"
	$(DC) exec parser sh

exec-indexer:
	@echo "$(WHITE)🐚 Entering indexer container...$(NC)"
	$(DC) exec indexer sh

exec-app:
	@echo "$(WHITE)🐚 Entering app container...$(NC)"
	$(DC) exec app sh

exec-crawler-prod:
	@echo "$(WHITE)🐚 Entering crawler-prod container...$(NC)"
	$(DC_PROD) exec crawler-prod sh

exec-parser-prod:
	@echo "$(WHITE)🐚 Entering parser-prod container...$(NC)"
	$(DC_PROD) exec parser-prod sh

exec-indexer-prod:
	@echo "$(WHITE)🐚 Entering indexer-prod container...$(NC)"
	$(DC_PROD) exec indexer-prod sh

exec-app-prod:
	@echo "$(WHITE)🐚 Entering app-prod container...$(NC)"
	$(DC_PROD) exec app-prod sh

# ============================================================================
# KAFKA MANAGEMENT
# ============================================================================

kafka-topics:
	@echo "$(YELLOW)📊 Listing Kafka topics...$(NC)"
	$(DC) exec kafka kafka-topics --bootstrap-server localhost:9092 --list

kafka-create-topics:
	@echo "$(YELLOW)➕ Creating Kafka topics...$(NC)"
	$(DC) exec kafka kafka-topics --bootstrap-server localhost:9092 --create --if-not-exists --topic raw-html --partitions 3 --replication-factor 1
	$(DC) exec kafka kafka-topics --bootstrap-server localhost:9092 --create --if-not-exists --topic parsed-pages --partitions 3 --replication-factor 1
	$(DC) exec kafka kafka-topics --bootstrap-server localhost:9092 --create --if-not-exists --topic indexed-pages --partitions 3 --replication-factor 1
	@echo "$(GREEN)✅ Topics created successfully$(NC)"

kafka-list-topics:
	@echo "$(YELLOW)📋 Kafka topics details...$(NC)"
	$(DC) exec kafka kafka-topics --bootstrap-server localhost:9092 --describe

kafka-delete-topics:
	@echo "$(RED)🗑️  Deleting Kafka topics...$(NC)"
	$(DC) exec kafka kafka-topics --bootstrap-server localhost:9092 --delete --topic raw-html || true
	$(DC) exec kafka kafka-topics --bootstrap-server localhost:9092 --delete --topic parsed-pages || true
	$(DC) exec kafka kafka-topics --bootstrap-server localhost:9092 --delete --topic indexed-pages || true

# ============================================================================
# REDIS MANAGEMENT
# ============================================================================

redis-cli:
	@echo "$(YELLOW)🔗 Accessing Redis CLI...$(NC)"
	$(DC) exec redis redis-cli

redis-flushall:
	@echo "$(RED)🧹 Flushing all Redis data...$(NC)"
	$(DC) exec redis redis-cli FLUSHALL
	@echo "$(GREEN)✅ Redis data flushed$(NC)"

# ============================================================================
# DEPENDENCIES (DEV)
# ============================================================================

install-deps:
	@echo "$(PURPLE)📦 Installing dependencies...$(NC)"
	$(DC) exec crawler go mod tidy || true
	$(DC) exec parser pip install -r requirements.txt || true
	$(DC) exec indexer pip install -r requirements.txt || true
	$(DC) exec app npm install || true

update-deps:
	@echo "$(PURPLE)⬆️  Updating dependencies...$(NC)"
	$(DC) exec crawler go get -u ./... || true
	$(DC) exec parser pip install --upgrade -r requirements.txt || true
	$(DC) exec indexer pip install --upgrade -r requirements.txt || true
	$(DC) exec app npm update || true
