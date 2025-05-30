%global debug_package %{nil}
%define upstream_name containerd-cc
%define upstream_repo confidential-containers-containerd
%define commit_hash e55e17bb9c75834c863d422bc38b54b0056e467a

Summary: Industry-standard container runtime for confidential containers
Name: moby-%{upstream_name}
Version: 1.7.7
Release: 9%{?dist}
License: ASL 2.0
Group: Tools/Container
URL: https://www.containerd.io
Vendor: Microsoft Corporation
Distribution: Azure Linux

Source0:  https://github.com/microsoft/confidential-containers-containerd/archive/refs/tags/%{version}.tar.gz#/%{name}-%{version}.tar.gz
Source1: containerd.service
Source2: containerd.toml
Patch0: CVE-2023-47108.patch
Patch1: CVE-2023-39325.patch
Patch2: fix_cc_tests_for_golang1.21.patch
Patch3: CVE-2024-24786.patch
Patch4: CVE-2024-28180.patch
Patch5: CVE-2023-45288.patch
Patch7: CVE-2023-44487.patch
Patch8: CVE-2025-27144.patch
Patch9: CVE-2024-40635.patch

%{?systemd_requires}

BuildRequires: git
BuildRequires: golang < 1.23
BuildRequires: go-md2man
BuildRequires: make
BuildRequires: systemd-rpm-macros

Requires: moby-runc >= 1.1.0

Conflicts: moby-containerd
Conflicts: moby-engine <= 3.0.10

%description
This is the containerd runtime meant for use with confidential containers

%prep
%autosetup -p1 -n %{upstream_repo}-%{version}

%build
export BUILDTAGS="-mod=vendor"
make VERSION="%{version}" REVISION="%{commit_hash}" binaries man

%check
export BUILDTAGS="-mod=vendor"
make VERSION="%{version}" REVISION="%{commit_hash}" test

%install
make VERSION="%{version}" REVISION="%{commit_hash}" DESTDIR="%{buildroot}" PREFIX="/usr" install install-man

mkdir -p %{buildroot}/%{_unitdir}
install -D -p -m 0644 %{SOURCE1} %{buildroot}%{_unitdir}/containerd.service
install -D -p -m 0644 %{SOURCE2} %{buildroot}%{_sysconfdir}/containerd/config.toml

%post
%systemd_post containerd.service

if [ $1 -eq 1 ]; then # Package install
	systemctl enable containerd.service > /dev/null 2>&1 || :
	systemctl start containerd.service > /dev/null 2>&1 || :
fi

%preun
%systemd_preun containerd.service

%postun
%systemd_postun_with_restart containerd.service

%files
%license LICENSE NOTICE
%{_bindir}/*
%{_mandir}/*
%config(noreplace) %{_unitdir}/containerd.service
%config(noreplace) %{_sysconfdir}/containerd/config.toml

%changelog
* Wed Apr 16 2025 Manuel Huber <mahuber@microsoft.com> - 1.7.7-9
- Fix CVE-2024-40635

* Tue Apr 08 2025 Manuel Huber <mahuber@microsoft.com> - 1.7.7-8
- Fix CVE-2025-27144 with an upstream patch

* Wed Mar 05 2025 corvus-callidus <108946721+corvus-callidus@users.noreply.github.com - 1.7.7-7
- Rename patch file that addresses CVE-2023-39325
- Address CVE-2023-44487
- Fix rpmbuild warnings for changelog dates

* Fri Jan 31 2025 Kanishk Bansal <kanbansal@microsoft.com> - 1.7.7-6
- Address CVE-2024-28180, CVE-2023-45288

* Mon Nov 25 2024 Bala <balakumaran.kannan@microsoft.com> - 1.7.7-5
- Fix CVE-2024-24786 by patching

* Tue Oct 15 2024 Muhammad Falak <mwani@microsoft.com> - 1.7.7-4
- Pin golang version to <= 1.22

* Mon Apr 08 2024 Mitch Zhu <mitchzhu@microsoft.com> - 1.7.7-3
- Drop obsolete btrfs-progs-devel build dependency

* Mon Apr 01 2024 Henry Beberman <henry.beberman@microsoft.com> - 1.7.1-9
- Remove Obsoletes containerd as it was causing dnf to pick moby-containerd-cc over containerd.

* Fri Mar 08 2024 Henry Beberman <henry.beberman@microsoft.com> - 1.7.1-8
- Add OOMScoreAdjust -999 to containerd.service

* Wed Feb 21 2024 Henry Beberman <henry.beberman@microsoft.com> - 1.7.7-2
- Backport upstream patch for no-inlining seccomp and apparmor functions to fix tests.

* Tue Feb 20 2024 Mitch Zhu <mitchzhu@microsoft.com> - 1.7.7-1
- Upgrade to upstream containerd v1.7.7.

* Fri Feb 02 2024 Daniel McIlvaney <damcilva@microsoft.com> - 1.7.2-4
- Address CVE-2023-44487 by patching vendored golang.org/x/net

* Wed Dec 20 2023 Manuel Huber <mahuber@microsoft.com> - 1.7.2-3
- Set oom_score_adj of containerd to -999

* Wed Nov 22 2023 Bala <balakumaran.kannan@gmail.com> - 1.7.2-2
- Fix CVE-2023-47108 by backporting the fix made for otel-grpc-0.40.0

* Fri Nov 10 2023 Saul Paredes <saulparedes@microsoft.com> - 1.7.2-1
- Always add TargetLayerDigestLabel label to snapshots

* Mon Oct 16 2023 CBL-Mariner Servicing Account <cblmargh@microsoft.com> - 1.7.1-6
- Bump release to rebuild with go 1.20.10

* Tue Oct 10 2023 Dan Streetman <ddstreet@ieee.org> - 1.7.1-5
- Bump release to rebuild with updated version of Go.

* Mon Aug 07 2023 CBL-Mariner Servicing Account <cblmargh@microsoft.com> - 1.7.1-4
- Bump release to rebuild with go 1.19.12

* Thu Jul 13 2023 CBL-Mariner Servicing Account <cblmargh@microsoft.com> - 1.7.1-3
- Bump release to rebuild with go 1.19.11

* Thu Jun 15 2023 CBL-Mariner Servicing Account <cblmargh@microsoft.com> - 1.7.1-2
- Bump release to rebuild with go 1.19.10

* Mon May 22 2023 Dallas Delaney <dadelan@microsoft.com> - 1.7.1-1
- Fix unit test arguments for TestSnapshotterFromPodSandboxConfig

* Wed May 17 2023 Dallas Delaney <dadelan@microsoft.com> - 1.7.0-2
- Add build version dependency on golang

* Tue Apr 25 2023 Dallas Delaney <dadelan@microsoft.com> - 1.7.0-1
- Add initial spec
- License verified.
- Original version for CBL-Mariner
