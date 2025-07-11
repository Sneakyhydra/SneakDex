# Variables
DC=docker-compose -f docker-compose.yml
DC_PROD=docker-compose -f docker-compose.prod.yml

# Colors for output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[1;33m
BLUE=\033[0;34m
PURPLE=\033[0;35m
CYAN=\033[0;36m
WHITE=\033[1;37m
NC=\033[0m # No Color

# General Commands
.PHONY: help \
        up-dev down-dev start-dev stop-dev restart-dev build-dev rebuild-dev logs-dev status-dev reset-dev clean-dev \
        up-prod down-prod start-prod stop-prod restart-prod build-prod rebuild-prod logs-prod status-prod reset-prod clean-prod \
        start-prometheus stop-prometheus restart-prometheus logs-prometheus \
        start-prometheus-prod stop-prometheus-prod restart-prometheus-prod logs-prometheus-prod \
        start-kafka stop-kafka restart-kafka logs-kafka \
        start-redis stop-redis restart-redis logs-redis \
        start-crawler stop-crawler restart-crawler logs-crawler \
        start-parser stop-parser restart-parser logs-parser \
        start-indexer stop-indexer restart-indexer logs-indexer \
        start-app stop-app restart-app logs-app \
		start-crawler-prod stop-crawler-prod restart-crawler-prod logs-crawler-prod \
        start-parser-prod stop-parser-prod restart-parser-prod logs-parser-prod \
        start-indexer-prod stop-indexer-prod restart-indexer-prod logs-indexer-prod \
        start-app-prod stop-app-prod restart-app-prod logs-app-prod \
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
	@echo "  $(CYAN)make start-prometheus$(NC)				Start prometheus service"
	@echo "  $(CYAN)make stop-prometheus$(NC)				Stop prometheus service"
	@echo "  $(CYAN)make restart-prometheus$(NC)				Restart prometheus service"
	@echo "  $(CYAN)make logs-prometheus$(NC)				View prometheus logs"
	@echo ""
	@echo "  $(CYAN)make start-prometheus-prod$(NC)				Start prometheus-prod service"
	@echo "  $(CYAN)make stop-prometheus-prod$(NC)				Stop prometheus-prod service"
	@echo "  $(CYAN)make restart-prometheus-prod$(NC)				Restart prometheus-prod service"
	@echo "  $(CYAN)make logs-prometheus-prod$(NC)				View prometheus-prod logs"
	@echo ""
	@echo "  $(CYAN)make start-kafka$(NC)				Start Kafka service"
	@echo "  $(CYAN)make stop-kafka$(NC)				Stop Kafka service"
	@echo "  $(CYAN)make restart-kafka$(NC)				Restart Kafka service"
	@echo "  $(CYAN)make logs-kafka$(NC)				View Kafka logs"
	@echo ""
	@echo "  $(CYAN)make start-redis$(NC)				Start Redis service"
	@echo "  $(CYAN)make stop-redis$(NC)				Stop Redis service"
	@echo "  $(CYAN)make restart-redis$(NC)				Restart Redis service"
	@echo "  $(CYAN)make logs-redis$(NC)				View Redis logs"
	@echo ""
	@echo "$(GREEN)🔧 INDIVIDUAL SERVICE MANAGEMENT$(NC)"
	@echo "  $(PURPLE)make start-crawler$(NC)			Start crawler service"
	@echo "  $(PURPLE)make stop-crawler$(NC)			Stop crawler service"
	@echo "  $(PURPLE)make restart-crawler$(NC)			Restart crawler service"
	@echo "  $(PURPLE)make logs-crawler$(NC)			View crawler logs"
	@echo ""
	@echo "  $(PURPLE)make start-parser$(NC)			Start parser service"
	@echo "  $(PURPLE)make stop-parser$(NC)				Stop parser service"
	@echo "  $(PURPLE)make restart-parser$(NC)			Restart parser service"
	@echo "  $(PURPLE)make logs-parser$(NC)				View parser logs"
	@echo ""
	@echo "  $(PURPLE)make start-indexer$(NC)			Start indexer service"
	@echo "  $(PURPLE)make stop-indexer$(NC)			Stop indexer service"
	@echo "  $(PURPLE)make restart-indexer$(NC)			Restart indexer service"
	@echo "  $(PURPLE)make logs-indexer$(NC)			View indexer logs"
	@echo ""
	@echo "  $(PURPLE)make start-app$(NC)			Start app"
	@echo "  $(PURPLE)make stop-app$(NC)			Stop app"
	@echo "  $(PURPLE)make restart-app$(NC)			Restart app"
	@echo "  $(PURPLE)make logs-app$(NC)			View app logs"
	@echo ""
	@echo "$(GREEN)🔧 INDIVIDUAL SERVICE MANAGEMENT (PROD)$(NC)"
	@echo "  $(PURPLE)make start-crawler-prod$(NC)				Start crawler-prod service"
	@echo "  $(PURPLE)make stop-crawler-prod$(NC)				Stop crawler-prod service"
	@echo "  $(PURPLE)make restart-crawler-prod$(NC)				Restart crawler-prod service"
	@echo "  $(PURPLE)make logs-crawler-prod$(NC)				View crawler-prod logs"
	@echo ""
	@echo "  $(PURPLE)make start-parser-prod$(NC)				Start parser-prod service"
	@echo "  $(PURPLE)make stop-parser-prod$(NC)				Stop parser-prod service"
	@echo "  $(PURPLE)make restart-parser-prod$(NC)				Restart parser-prod service"
	@echo "  $(PURPLE)make logs-parser-prod$(NC)				View parser-prod logs"
	@echo ""
	@echo "  $(PURPLE)make start-indexer-prod$(NC)				Start indexer-prod service"
	@echo "  $(PURPLE)make stop-indexer-prod$(NC)				Stop indexer-prod service"
	@echo "  $(PURPLE)make restart-indexer-prod$(NC)				Restart indexer-prod service"
	@echo "  $(PURPLE)make logs-indexer-prod$(NC)				View indexer-prod logs"
	@echo ""
	@echo "  $(PURPLE)make start-app-prod$(NC)				Start app-prod"
	@echo "  $(PURPLE)make stop-app-prod$(NC)				Stop app-prod"
	@echo "  $(PURPLE)make restart-app-prod$(NC)			Restart app-prod"
	@echo "  $(PURPLE)make logs-app-prod$(NC)				View app-prod logs"
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

start-prometheus:
	@echo "$(CYAN)🎯 Starting prometheus service...$(NC)"
	COMPOSE_BAKE=true $(DC) up -d prometheus

stop-prometheus:
	@echo "$(CYAN)⏹️  Stopping prometheus service...$(NC)"
	$(DC) stop prometheus

restart-prometheus:
	@echo "$(CYAN)🔄 Restarting prometheus service...$(NC)"
	$(DC) restart prometheus

logs-prometheus:
	@echo "$(CYAN)📋 Viewing prometheus logs...$(NC)"
	$(DC) logs -f prometheus

start-prometheus-prod:
	@echo "$(CYAN)🎯 Starting prometheus-prod service...$(NC)"
	COMPOSE_BAKE=true $(DC_PROD) up -d prometheus-prod

stop-prometheus-prod:
	@echo "$(CYAN)⏹️  Stopping prometheus-prod service...$(NC)"
	$(DC_PROD) stop prometheus-prod

restart-prometheus-prod:
	@echo "$(CYAN)🔄 Restarting prometheus-prod service...$(NC)"
	$(DC_PROD) restart prometheus-prod

logs-prometheus-prod:
	@echo "$(CYAN)📋 Viewing prometheus-prod logs...$(NC)"
	$(DC_PROD) logs -f prometheus-prod

start-kafka:
	@echo "$(CYAN)🎯 Starting Kafka service...$(NC)"
	COMPOSE_BAKE=true $(DC) up -d kafka

stop-kafka:
	@echo "$(CYAN)⏹️  Stopping Kafka service...$(NC)"
	$(DC) stop kafka

restart-kafka:
	@echo "$(CYAN)🔄 Restarting Kafka service...$(NC)"
	$(DC) restart kafka

logs-kafka:
	@echo "$(CYAN)📋 Viewing Kafka logs...$(NC)"
	$(DC) logs -f kafka

start-redis:
	@echo "$(CYAN)🗄️  Starting Redis service...$(NC)"
	COMPOSE_BAKE=true $(DC) up -d redis

stop-redis:
	@echo "$(CYAN)⏹️  Stopping Redis service...$(NC)"
	$(DC) stop redis

restart-redis:
	@echo "$(CYAN)🔄 Restarting Redis service...$(NC)"
	$(DC) restart redis

logs-redis:
	@echo "$(CYAN)📋 Viewing Redis logs...$(NC)"
	$(DC) logs -f redis

# ============================================================================
# INDIVIDUAL SERVICE MANAGEMENT
# ============================================================================

start-crawler:
	@echo "$(PURPLE)🕷️  Starting crawler service...$(NC)"
	COMPOSE_BAKE=true $(DC) up -d crawler 

stop-crawler:
	@echo "$(PURPLE)⏹️  Stopping crawler service...$(NC)"
	$(DC) stop crawler

restart-crawler:
	@echo "$(PURPLE)🔄 Restarting crawler service...$(NC)"
	$(DC) restart crawler

logs-crawler:
	@echo "$(PURPLE)📋 Viewing crawler logs...$(NC)"
	$(DC) logs -f crawler

start-parser:
	@echo "$(PURPLE)🔍 Starting parser service...$(NC)"
	COMPOSE_BAKE=true $(DC) up -d parser

stop-parser:
	@echo "$(PURPLE)⏹️  Stopping parser service...$(NC)"
	$(DC) stop parser

restart-parser:
	@echo "$(PURPLE)🔄 Restarting parser service...$(NC)"
	$(DC) restart parser

logs-parser:
	@echo "$(PURPLE)📋 Viewing parser logs...$(NC)"
	$(DC) logs -f parser

start-indexer:
	@echo "$(PURPLE)🗂️  Starting indexer service...$(NC)"
	COMPOSE_BAKE=true $(DC) up -d indexer

stop-indexer:
	@echo "$(PURPLE)⏹️  Stopping indexer service...$(NC)"
	$(DC) stop indexer

restart-indexer:
	@echo "$(PURPLE)🔄 Restarting indexer service...$(NC)"
	$(DC) restart indexer

logs-indexer:
	@echo "$(PURPLE)📋 Viewing indexer logs...$(NC)"
	$(DC) logs -f indexer

start-app:
	@echo "$(PURPLE)🔌 Starting app...$(NC)"
	COMPOSE_BAKE=true $(DC) up -d app

stop-app:
	@echo "$(PURPLE)⏹️  Stopping app...$(NC)"
	$(DC) stop app

restart-app:
	@echo "$(PURPLE)🔄 Restarting app...$(NC)"
	$(DC) restart app

logs-app:
	@echo "$(PURPLE)📋 Viewing app logs...$(NC)"
	$(DC) logs -f app

# ============================================================================
# INDIVIDUAL SERVICE MANAGEMENT (PROD)
# ============================================================================

start-crawler-prod:
	@echo "$(PURPLE)🕷️  Starting crawler-prod service...$(NC)"
	COMPOSE_BAKE=true $(DC_PROD) up -d crawler-prod

stop-crawler-prod:
	@echo "$(PURPLE)⏹️  Stopping crawler-prod service...$(NC)"
	$(DC_PROD) stop crawler-prod

restart-crawler-prod:
	@echo "$(PURPLE)🔄 Restarting crawler-prod service...$(NC)"
	$(DC_PROD) restart crawler-prod

logs-crawler-prod:
	@echo "$(PURPLE)📋 Viewing crawler-prod logs...$(NC)"
	$(DC_PROD) logs -f crawler-prod

start-parser-prod:
	@echo "$(PURPLE)🔍 Starting parser-prod service...$(NC)"
	COMPOSE_BAKE=true $(DC_PROD) up -d parser-prod

stop-parser-prod:
	@echo "$(PURPLE)⏹️  Stopping parser-prod service...$(NC)"
	$(DC_PROD) stop parser-prod

restart-parser-prod:
	@echo "$(PURPLE)🔄 Restarting parser-prod service...$(NC)"
	$(DC_PROD) restart parser-prod

logs-parser-prod:
	@echo "$(PURPLE)📋 Viewing parser-prod logs...$(NC)"
	$(DC_PROD) logs -f parser-prod

start-indexer-prod:
	@echo "$(PURPLE)🗂️  Starting indexer-prod service...$(NC)"
	COMPOSE_BAKE=true $(DC_PROD) up -d indexer-prod

stop-indexer-prod:
	@echo "$(PURPLE)⏹️  Stopping indexer-prod service...$(NC)"
	$(DC_PROD) stop indexer-prod

restart-indexer-prod:
	@echo "$(PURPLE)🔄 Restarting indexer-prod service...$(NC)"
	$(DC_PROD) restart indexer-prod

logs-indexer-prod:
	@echo "$(PURPLE)📋 Viewing indexer-prod logs...$(NC)"
	$(DC_PROD) logs -f indexer-prod

start-app-prod:
	@echo "$(PURPLE)🔌 Starting app-prod...$(NC)"
	COMPOSE_BAKE=true $(DC_PROD) up -d app-prod

stop-app-prod:
	@echo "$(PURPLE)⏹️  Stopping app-prod...$(NC)"
	$(DC_PROD) stop app-prod

restart-app-prod:
	@echo "$(PURPLE)🔄 Restarting app-prod...$(NC)"
	$(DC_PROD) restart app-prod

logs-app-prod:
	@echo "$(PURPLE)📋 Viewing app-prod logs...$(NC)"
	$(DC_PROD) logs -f app-prod

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
	@echo "$(GREEN)✅ Topics created successfully!$(NC)"

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
	@echo "$(GREEN)✅ Redis data flushed!$(NC)"

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