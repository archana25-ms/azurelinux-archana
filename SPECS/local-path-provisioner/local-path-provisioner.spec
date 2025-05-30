Summary:        Provides a way for the Kubernetes users to utilize the local storage in each node
Name:           local-path-provisioner
Version:        0.0.24
Release:        5%{?dist}
License:        ASL 2.0
URL:            https://github.com/rancher/local-path-provisioner
Group:          Applications/Text
Vendor:         Microsoft Corporation
Distribution:   Azure Linux
Source0:        https://github.com/rancher/%{name}/archive/refs/tags/v%{version}.tar.gz#/%{name}-%{version}.tar.gz
#Note that the source file should be renamed to the format {name}-%%{version}.tar.gz
Patch0:         CVE-2023-45288.patch
Patch1:         CVE-2023-39325.patch
Patch2:         CVE-2023-44487.patch
Patch3:		CVE-2020-8565.patch
BuildRequires: golang

%description
Provides a way for the Kubernetes users to utilize the local storage in each node. 

%prep
%autosetup -p1

%build
export CGO_ENABLED=0
go build -mod=vendor 

%install
install -d %{buildroot}%{_bindir}
install local-path-provisioner %{buildroot}%{_bindir}/local-path-provisioner

%files
%{_bindir}/local-path-provisioner

%changelog
* Wed Mar 05 2025 Archana Shettigar <v-shettigara@microsoft.com> - 0.0.24-5
- Address CVE-2020-8565

* Tue Mar 04 2025 corvus-callidus <108946721+corvus-callidus@users.noreply.github.com> - 0.0.24-4
* Address CVE-2023-44487


* Fri Feb 14 2025 Kanishk Bansal <kanbansal@microsoft.com> - 0.0.24-3
- Address CVE-2023-45288, CVE-2023-39325

* Tue Sep 03 2024 Pawel Winogrodzki <pawelwi@microsoft.com> - 0.0.24-2
- Release bump to fix package information.

* Fri Oct 27 2023 CBL-Mariner Servicing Account <cblmargh@microsoft.com> - 0.0.24-1
- Auto-upgrade to 0.0.24 - Azure Linux 3.0 - package upgrades

* Mon Oct 16 2023 CBL-Mariner Servicing Account <cblmargh@microsoft.com> - 0.0.21-13
- Bump release to rebuild with go 1.20.10

* Tue Oct 10 2023 Dan Streetman <ddstreet@ieee.org> - 0.0.21-12
- Bump release to rebuild with updated version of Go.

* Mon Aug 07 2023 CBL-Mariner Servicing Account <cblmargh@microsoft.com> - 0.0.21-11
- Bump release to rebuild with go 1.19.12

* Thu Jul 13 2023 CBL-Mariner Servicing Account <cblmargh@microsoft.com> - 0.0.21-10
- Bump release to rebuild with go 1.19.11

* Thu Jun 15 2023 CBL-Mariner Servicing Account <cblmargh@microsoft.com> - 0.0.21-9
- Bump release to rebuild with go 1.19.10

* Wed Apr 05 2023 CBL-Mariner Servicing Account <cblmargh@microsoft.com> - 0.0.21-8
- Bump release to rebuild with go 1.19.8

* Tue Mar 28 2023 CBL-Mariner Servicing Account <cblmargh@microsoft.com> - 0.0.21-7
- Bump release to rebuild with go 1.19.7

* Wed Mar 15 2023 CBL-Mariner Servicing Account <cblmargh@microsoft.com> - 0.0.21-6
- Bump release to rebuild with go 1.19.6

* Fri Feb 03 2023 CBL-Mariner Servicing Account <cblmargh@microsoft.com> - 0.0.21-5
- Bump release to rebuild with go 1.19.5

* Wed Jan 18 2023 CBL-Mariner Servicing Account <cblmargh@microsoft.com> - 0.0.21-4
- Bump release to rebuild with go 1.19.4

* Fri Dec 16 2022 Daniel McIlvaney <damcilva@microsoft.com> - 0.0.21-3
- Bump release to rebuild with go 1.18.8 with patch for CVE-2022-41717

* Tue Nov 01 2022 Olivia Crain <oliviacrain@microsoft.com> - 0.0.21-2
- Bump release to rebuild with go 1.18.8

* Thu Jun 23 2022 Lior Lustgarten <lilustga@microsoft.com> 0.0.21-1
- Original version for CBL-Mariner
- License Verified
