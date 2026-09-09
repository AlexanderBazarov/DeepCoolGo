# DeepCoolGo - build & install
#
#   make              build binary + theme plugins into ./build
#   make dev          + install the bundled themes into ~/.config/DCGO/Themes
#                       and seed ~/.config/DCGO/config.yml (no root)
#   sudo make install install binary, plugins, systemd unit and default config
#   sudo make enable  daemon-reload + systemctl enable --now
#   sudo make uninstall
#   make rpm / deb    build a package into ./build (needs nfpm)
#   make dist         source tarball deepcoolgo-$(VERSION).tar.gz (for rpmbuild)
#
# Overridable: PREFIX (default /usr/local), DESTDIR, BINDIR, LIBDIR, SYSTEMDDIR,
#              CONFDIR, GO, GOFLAGS, VERSION, XDG_CONFIG_HOME (default ~/.config).
# Requires: go toolchain, gcc, libusb-1.0 development headers (CGO).

BINARY     := deepcoolgo
VERSION    ?= 0.1.0
PREFIX     ?= /usr/local
DESTDIR    ?=
BINDIR     ?= $(PREFIX)/bin
LIBDIR     ?= $(PREFIX)/lib
THEMESDIR  := $(LIBDIR)/$(BINARY)/themes
SYSTEMDDIR ?= /usr/lib/systemd/system
CONFDIR    ?= /etc/$(BINARY)

# per-user location used by "make dev" (matches Go's os.UserConfigDir)
XDG_CONFIG_HOME ?= $(HOME)/.config
USER_CONFDIR    := $(XDG_CONFIG_HOME)/DCGO
USER_THEMESDIR  := $(USER_CONFDIR)/Themes

GO         ?= go
GOFLAGS    ?=
GOARCH     := $(shell $(GO) env GOARCH)
export CGO_ENABLED = 1

BUILDDIR    := build
PKGROOT     := $(BUILDDIR)/pkgroot
GO_SRCS     := $(shell find . -name '*.go' -not -path './plugins/*') go.mod go.sum
PLUGIN_DIRS := $(notdir $(patsubst %/,%,$(wildcard plugins/*/)))
PLUGIN_SOS  := $(addprefix $(BUILDDIR)/themes/,$(addsuffix .so,$(PLUGIN_DIRS)))

.PHONY: all build plugins dev test install uninstall enable disable clean \
        pkgroot rpm deb dist

all: build plugins

build: $(BUILDDIR)/$(BINARY)

$(BUILDDIR)/$(BINARY): $(GO_SRCS)
	@mkdir -p $(BUILDDIR)
	$(GO) build $(GOFLAGS) -o $@ .

plugins: $(PLUGIN_SOS)

$(BUILDDIR)/themes/%.so: plugins/%/main.go $(GO_SRCS)
	@mkdir -p $(BUILDDIR)/themes
	$(GO) build $(GOFLAGS) -buildmode=plugin -o $@ ./plugins/$*

dev: all
	@mkdir -p $(USER_THEMESDIR)
	install -m644 $(BUILDDIR)/themes/*.so -t $(USER_THEMESDIR)
	@if [ -f "$(USER_CONFDIR)/config.yml" ]; then \
	    echo "keeping existing $(USER_CONFDIR)/config.yml"; \
	else \
	    printf 'refresh_rate: 5\ntheme: aether.so\nthemes_path: %s\n' "$(USER_THEMESDIR)" \
	        > $(USER_CONFDIR)/config.yml; \
	    echo "wrote $(USER_CONFDIR)/config.yml"; \
	fi
	@echo
	@echo "themes installed to $(USER_THEMESDIR)"
	@echo "run: sudo ./$(BUILDDIR)/$(BINARY)"

install: all
	install -Dm755 $(BUILDDIR)/$(BINARY) $(DESTDIR)$(BINDIR)/$(BINARY)
	@mkdir -p $(DESTDIR)$(THEMESDIR)
	install -m644 $(BUILDDIR)/themes/*.so -t $(DESTDIR)$(THEMESDIR)
	@mkdir -p $(DESTDIR)$(SYSTEMDDIR)
	sed -e 's|@BINDIR@|$(BINDIR)|g' -e 's|@CONFDIR@|$(CONFDIR)|g' \
	    packaging/$(BINARY).service.in > $(DESTDIR)$(SYSTEMDDIR)/$(BINARY).service
	chmod 644 $(DESTDIR)$(SYSTEMDDIR)/$(BINARY).service
	@if [ -f "$(DESTDIR)$(CONFDIR)/DCGO/config.yml" ]; then \
	    echo "keeping existing $(CONFDIR)/DCGO/config.yml"; \
	else \
	    mkdir -p $(DESTDIR)$(CONFDIR)/DCGO; \
	    sed -e 's|@THEMESDIR@|$(THEMESDIR)|g' packaging/config.yml.in \
	        > $(DESTDIR)$(CONFDIR)/DCGO/config.yml; \
	    echo "installed default $(CONFDIR)/DCGO/config.yml"; \
	fi
	@echo
	@echo "Next: sudo systemctl daemon-reload && sudo systemctl enable --now $(BINARY)"
	@echo "  (or just: sudo make enable)"

test:
	$(GO) test ./...

# staged install tree consumed by nfpm (rpm/deb)
pkgroot: all
	rm -rf $(PKGROOT)
	$(MAKE) install DESTDIR=$(PKGROOT) \
	    PREFIX=/usr BINDIR=/usr/bin LIBDIR=/usr/lib \
	    SYSTEMDDIR=/usr/lib/systemd/system CONFDIR=/etc/$(BINARY)

rpm: pkgroot
	VERSION=$(VERSION) GOARCH=$(GOARCH) nfpm package -f packaging/nfpm.yaml -p rpm -t $(BUILDDIR)

deb: pkgroot
	VERSION=$(VERSION) GOARCH=$(GOARCH) nfpm package -f packaging/nfpm.yaml -p deb -t $(BUILDDIR)

dist:
	git archive --format=tar.gz --prefix=$(BINARY)-$(VERSION)/ \
	    -o $(BINARY)-$(VERSION).tar.gz HEAD 2>/dev/null || \
	tar czf $(BINARY)-$(VERSION).tar.gz \
	    --transform 's,^\./,$(BINARY)-$(VERSION)/,' \
	    --exclude=./.git --exclude=./$(BUILDDIR) --exclude='./*.tar.gz' .

enable:
	systemctl daemon-reload
	systemctl enable --now $(BINARY).service

disable:
	-systemctl disable --now $(BINARY).service

uninstall: disable
	rm -f $(DESTDIR)$(BINDIR)/$(BINARY)
	rm -rf $(DESTDIR)$(LIBDIR)/$(BINARY)
	rm -f $(DESTDIR)$(SYSTEMDDIR)/$(BINARY).service
	systemctl daemon-reload || true
	@echo "config left at $(CONFDIR) - remove it by hand if you want it gone"

clean:
	rm -rf $(BUILDDIR)
