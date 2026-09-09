Name:           deepcoolgo
Version:        0.1.0
Release:        1%{?dist}
Summary:        Live CPU telemetry on DeepCool LCD coolers

License:        MIT
URL:            https://example.com/deepcoolgo
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  golang >= 1.26
BuildRequires:  gcc
BuildRequires:  pkgconfig(libusb-1.0)
BuildRequires:  systemd-rpm-macros
Requires:       libusbx
%{?systemd_requires}

%description
DeepCoolGo reads CPU load, temperature, clock speed and power draw from the
Linux kernel and renders a 320x240 frame to the LCD on a DeepCool cooler over
USB. Every visual is produced by a theme - a Go plugin (.so) - so the display
can be restyled without touching the daemon.

%prep
%autosetup -n %{name}-%{version}

%build
export CGO_ENABLED=1
# -mod=vendor when the tarball ships vendor/ (mock/COPR, no network);
# drop it for a plain local rpmbuild that is allowed to fetch modules.
%make_build GOFLAGS="-trimpath -buildvcs=false %{?_with_vendor:-mod=vendor}"

%install
%make_install PREFIX=%{_prefix} BINDIR=%{_bindir} LIBDIR=%{_libdir} \
              SYSTEMDDIR=%{_unitdir} CONFDIR=%{_sysconfdir}/%{name}

%post
%systemd_post %{name}.service

%preun
%systemd_preun %{name}.service

%postun
%systemd_postun_with_restart %{name}.service

%files
%license LICENSE
%doc README.md
%{_bindir}/deepcoolgo
%dir %{_libdir}/deepcoolgo
%dir %{_libdir}/deepcoolgo/themes
%{_libdir}/deepcoolgo/themes/*.so
%{_unitdir}/deepcoolgo.service
%dir %{_sysconfdir}/deepcoolgo
%dir %{_sysconfdir}/deepcoolgo/DCGO
%config(noreplace) %{_sysconfdir}/deepcoolgo/DCGO/config.yml

%changelog
* Tue Sep 09 2026 3x3cutable <avbazarov2006@mail.ru> - 0.1.0-1
- Initial package
