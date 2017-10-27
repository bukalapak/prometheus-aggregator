.PHONY: all

REGISTRY  = registry.bukalapak.io/bukalapak
DDIR      = deploy
ODIR      = $(DDIR)/_output
NOCACHE   = --no-cache
VERSION   = $(shell git show -q --format=%h)
DEFENV    = production canary sandbox
SERVICES ?= prometheus_aggregator 
ENV      ?= $(DEFENV)
ACTION   ?= replace
FILE     ?= deployment

all:
	consul compile build push deployment

test:
	govendor test -v -cover +local,^program

dep:
	govendor fetch -v +outside

compile:
	@$(foreach var, $(SERVICES), GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(ODIR)/$(var)/bin/$(var) app/$(var)/main.go;)

$(ODIR):
	mkdir -p $(ODIR)

consul: $(ODIR)
	@wget https://releases.hashicorp.com/envconsul/0.6.2/envconsul_0.6.2_linux_amd64.tgz
	@tar -xf envconsul_0.6.2_linux_amd64.tgz -C $(ODIR)/
	@rm envconsul_0.6.2_linux_amd64.tgz

build:
	@$(foreach var, $(SERVICES), docker build $(NOCACHE) -t $(REGISTRY)/prometheus_aggregator/$(var):$(VERSION) -f ./deploy/$(var)/Dockerfile .;)

push:
	@$(foreach var, $(SERVICES), docker push $(REGISTRY)/prometheus_aggregator/$(var):$(VERSION);)

deployment: $(ODIR)
ifeq ($(ENV),$(DEFENV))
	kubelize deployment -v $(VERSION) $(SERVICES)
else
	kubelize deployment -e $(ENV) -v $(VERSION) $(SERVICES)
endif

$(ENV):
	$(foreach var, $(SERVICES), kubectl $(ACTION) -f $(ODIR)/$(var)/$@/$(FILE).yml;)

setup:
	docker run --rm -it --network host -v $PWD/db:/app/db -v $PWD/.env:/app/.env registry.bukalapak.io/sre/migration:0.0.1 db:create
	docker run --rm -it --network host -v $PWD/db:/app/db -v $PWD/.env:/app/.env registry.bukalapak.io/sre/migration:0.0.1 db:migrate

migrate:
	docker run --rm -it --network host -v $PWD/db:/app/db -v $PWD/.env:/app/.env registry.bukalapak.io/sre/migration:0.0.1 db:migrate
