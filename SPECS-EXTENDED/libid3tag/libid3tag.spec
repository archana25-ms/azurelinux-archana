Vendor:         Microsoft Corporation
Distribution:   Azure Linux
Name:           libid3tag
Version:        0.16.3
Release:        7%{?dist}
Summary:        ID3 tag manipulation library

License:        GPL-2.0-or-later
URL:            https://codeberg.org/tenacityteam/libid3tag
Source0:        https://codeberg.org/tenacityteam/libid3tag/archive/%{version}.tar.gz#/%{name}-%{version}.tar.gz
Patch0:         cmake-hook-genre.dat-and-gperf-files-generation.patch

BuildRequires:  gcc-c++
BuildRequires:  cmake
BuildRequires:  gperf
BuildRequires:  zlib-devel >= 1.1.4

%description
libid3tag is a library for reading and (eventually) writing ID3 tags,
both ID3v1 and the various versions of ID3v2.

%package        devel
Summary:        Development files for %{name}
Requires:       %{name}%{?_isa} = %{version}-%{release}

%description    devel
ID3 tag library development files.

%prep
%autosetup -p1 -n %{name}

%build
%cmake
%cmake_build

%install
%cmake_install

%ldconfig_scriptlets

%files
%doc CHANGES CREDITS README TODO
%license COPYING COPYRIGHT
%{_libdir}/libid3tag.so.0*

%files devel
%{_includedir}/id3tag.h
%{_libdir}/libid3tag.so
%{_libdir}/cmake/id3tag/
%{_libdir}/pkgconfig/id3tag.pc

%changelog
* Fri Oct 10 2024 Durga Jagadeesh Palli <v-dpalli@microsoft.com> - 0.16.3-7
- Initial Azure Linux import from Fedora 41 (license: MIT)
- License verified.

* Thu Jul 18 2024 Fedora Release Engineering <releng@fedoraproject.org> - 0.16.3-6
- Rebuilt for https://fedoraproject.org/wiki/Fedora_41_Mass_Rebuild

* Wed Apr 17 2024 Peter Lemenkov <lemenkov@gmail.com> - 0.16.3-5
- Use autorelease macro

* Thu Jan 25 2024 Fedora Release Engineering <releng@fedoraproject.org> - 0.16.3-3
- Rebuilt for https://fedoraproject.org/wiki/Fedora_40_Mass_Rebuild

* Sun Jan 21 2024 Fedora Release Engineering <releng@fedoraproject.org> - 0.16.3-2
- Rebuilt for https://fedoraproject.org/wiki/Fedora_40_Mass_Rebuild

* Thu Dec 14 2023 Leigh Scott <leigh123linux@gmail.com> - 0.16.3-1
- Update to 0.16.3

* Thu Jul 20 2023 Fedora Release Engineering <releng@fedoraproject.org> - 0.16.2-3
- Rebuilt for https://fedoraproject.org/wiki/Fedora_39_Mass_Rebuild

* Wed Aug 31 2022 Leigh Scott <leigh123linux@gmail.com> - 0.16.2-1
- Update to 0.16.2

* Thu Jul 21 2022 Fedora Release Engineering <releng@fedoraproject.org> - 0.15.1b-37
- Rebuilt for https://fedoraproject.org/wiki/Fedora_37_Mass_Rebuild

* Thu Jan 20 2022 Fedora Release Engineering <releng@fedoraproject.org> - 0.15.1b-36
- Rebuilt for https://fedoraproject.org/wiki/Fedora_36_Mass_Rebuild

* Thu Jul 22 2021 Fedora Release Engineering <releng@fedoraproject.org> - 0.15.1b-35
- Rebuilt for https://fedoraproject.org/wiki/Fedora_35_Mass_Rebuild

* Tue Jan 26 2021 Fedora Release Engineering <releng@fedoraproject.org> - 0.15.1b-34
- Rebuilt for https://fedoraproject.org/wiki/Fedora_34_Mass_Rebuild

* Tue Jul 28 2020 Fedora Release Engineering <releng@fedoraproject.org> - 0.15.1b-33
- Rebuilt for https://fedoraproject.org/wiki/Fedora_33_Mass_Rebuild

* Fri Oct 15 2021 Pawel Winogrodzki <pawelwi@microsoft.com> - 0.15.1b-33
  Initial CBL-Mariner import from Fedora 32 (license: MIT).

* Wed Jan 29 2020 Fedora Release Engineering <releng@fedoraproject.org> - 0.15.1b-32
- Rebuilt for https://fedoraproject.org/wiki/Fedora_32_Mass_Rebuild

* Thu Jul 25 2019 Fedora Release Engineering <releng@fedoraproject.org> - 0.15.1b-31
- Rebuilt for https://fedoraproject.org/wiki/Fedora_31_Mass_Rebuild

* Fri Feb 01 2019 Fedora Release Engineering <releng@fedoraproject.org> - 0.15.1b-30
- Rebuilt for https://fedoraproject.org/wiki/Fedora_30_Mass_Rebuild

* Fri Jul 13 2018 Fedora Release Engineering <releng@fedoraproject.org> - 0.15.1b-29
- Rebuilt for https://fedoraproject.org/wiki/Fedora_29_Mass_Rebuild

* Fri Mar 30 2018 David King <amigadave@amigadave.com> - 0.15.1b-28
- Add gperf patch from Debian

* Thu Mar 29 2018 David King <amigadave@amigadave.com> - 0.15.1b-27
- Add ID3v1 zero padding patch from Debian
- Add a fix for CVE-2017-11550 (#1478934)
- Add a fix for CVE-2004-2779 (#1561983)
- Use %%license, remove Group tag
- Add BuildRequires on gcc

* Wed Feb 07 2018 Fedora Release Engineering <releng@fedoraproject.org> - 0.15.1b-26
- Rebuilt for https://fedoraproject.org/wiki/Fedora_28_Mass_Rebuild

* Wed Jan 31 2018 Igor Gnatenko <ignatenkobrain@fedoraproject.org> - 0.15.1b-25
- Switch to %%ldconfig_scriptlets

* Thu Aug 03 2017 Fedora Release Engineering <releng@fedoraproject.org> - 0.15.1b-24
- Rebuilt for https://fedoraproject.org/wiki/Fedora_27_Binutils_Mass_Rebuild

* Wed Jul 26 2017 Fedora Release Engineering <releng@fedoraproject.org> - 0.15.1b-23
- Rebuilt for https://fedoraproject.org/wiki/Fedora_27_Mass_Rebuild

* Fri Feb 10 2017 Fedora Release Engineering <releng@fedoraproject.org> - 0.15.1b-22
- Rebuilt for https://fedoraproject.org/wiki/Fedora_26_Mass_Rebuild

* Thu Feb 04 2016 Fedora Release Engineering <releng@fedoraproject.org> - 0.15.1b-21
- Rebuilt for https://fedoraproject.org/wiki/Fedora_24_Mass_Rebuild

* Wed Jun 17 2015 Fedora Release Engineering <rel-eng@lists.fedoraproject.org> - 0.15.1b-20
- Rebuilt for https://fedoraproject.org/wiki/Fedora_23_Mass_Rebuild

* Sun Aug 17 2014 Fedora Release Engineering <rel-eng@lists.fedoraproject.org> - 0.15.1b-19
- Rebuilt for https://fedoraproject.org/wiki/Fedora_21_22_Mass_Rebuild

* Sat Jun 07 2014 Fedora Release Engineering <rel-eng@lists.fedoraproject.org> - 0.15.1b-18
- Rebuilt for https://fedoraproject.org/wiki/Fedora_21_Mass_Rebuild

* Sat Aug 03 2013 Fedora Release Engineering <rel-eng@lists.fedoraproject.org> - 0.15.1b-17
- Rebuilt for https://fedoraproject.org/wiki/Fedora_20_Mass_Rebuild

* Fri May 10 2013 Hans de Goede <hdegoede@redhat.com> - 0.15.1b-16
- Remove no longer needed autoreconf call, %%configure from redhat-rpm-config
  >= 9.1.0-42 updates config.guess and config.sub for new architecture support

* Mon Mar 25 2013 Hans de Goede <hdegoede@redhat.com> - 0.15.1b-15
- Run autoreconf for aarch64 support (rhbz#925768)
- Make build honor RPM_OPT_FLAGS

* Thu Feb 14 2013 Fedora Release Engineering <rel-eng@lists.fedoraproject.org> - 0.15.1b-14
- Rebuilt for https://fedoraproject.org/wiki/Fedora_19_Mass_Rebuild

* Thu Jul 19 2012 Fedora Release Engineering <rel-eng@lists.fedoraproject.org> - 0.15.1b-13
- Rebuilt for https://fedoraproject.org/wiki/Fedora_18_Mass_Rebuild

* Fri Jan 13 2012 Fedora Release Engineering <rel-eng@lists.fedoraproject.org> - 0.15.1b-12
- Rebuilt for https://fedoraproject.org/wiki/Fedora_17_Mass_Rebuild

* Tue Feb 08 2011 Fedora Release Engineering <rel-eng@lists.fedoraproject.org> - 0.15.1b-11
- Rebuilt for https://fedoraproject.org/wiki/Fedora_15_Mass_Rebuild

* Wed Jan 13 2010 Todd Zullinger <tmz@pobox.com> - 0.15.1b-10
- Rebuild to ensure n-v-r is greater than F-12 branch
- Fix Source0 URL

* Fri Jul 24 2009 Fedora Release Engineering <rel-eng@lists.fedoraproject.org> - 0.15.1b-9
- Rebuilt for https://fedoraproject.org/wiki/Fedora_12_Mass_Rebuild

* Wed Feb 25 2009 Fedora Release Engineering <rel-eng@lists.fedoraproject.org> - 0.15.1b-8
- Rebuilt for https://fedoraproject.org/wiki/Fedora_11_Mass_Rebuild

* Thu Aug 28 2008 Todd Zullinger <tmz@pobox.com> - 0.15.1b-7
- Fix %%patch incantation for new rpm

* Fri May 09 2008 Todd Zullinger <tmz@pobox.com> - 0.15.1b-6
- fix for CVE-2008-2109 (#445812)

* Tue Feb 12 2008 Todd Zullinger <tmz@pobox.com> - 0.15.1b-5
- rebuild for gcc 4.3

* Mon Aug  6 2007 Ville Skyttä <ville.skytta at iki.fi> - 0.15.1b-4
- License: GPLv2+

* Mon Aug 28 2006 Ville Skyttä <ville.skytta at iki.fi> - 0.15.1b-3
- Rebuild.

* Mon Feb 13 2006 Ville Skyttä <ville.skytta at iki.fi> - 0.15.1b-2
- Rebuild.

* Wed Nov  9 2005 Ville Skyttä <ville.skytta at iki.fi> - 0.15.1b-1
- Don't ship static libraries.
- Embed *.pc in specfile to keep it in sync with the build.
- Build with dependency tracking disabled.

* Thu May 19 2005 Ville Skyttä <ville.skytta at iki.fi> - 0.15.1-3.b
- Rebuild.

* Fri Apr  7 2005 Michael Schwendt <mschwendt[AT]users.sf.net> - 0.15.1-2.b
- rebuilt

* Wed Feb 18 2004 Ville Skyttä <ville.skytta at iki.fi> - 0:0.15.1-0.fdr.1.b
- Update to 0.15.1b.

* Wed Oct 29 2003 Ville Skytta <ville.skytta at iki.fi> - 0:0.15.0-0.fdr.2.b
- Rebuild.

* Sun Sep 28 2003 Dams <anvil[AT]livna.org> 0:0.15.0-0.fdr.1.b.0.94
- Remove comment after scriptlets

* Mon Jun 30 2003 Ville Skyttä <ville.skytta at iki.fi> - 0:0.15.0-0.fdr.1.b
- Update to 0.15.0b.
- Split separate from the old mad package to follow upstream.
- -devel requires zlib-devel and pkgconfig.

* Thu Apr 24 2003 Ville Skyttä <ville.skytta at iki.fi> - 0:0.14.2-0.fdr.3.b
- Fix missing "main" package dependencies in *-devel.
- Include patch from Debian, possibly fixes #187 comment 7, and adds
  pkgconfig files for libraries.

* Sun Apr 20 2003 Ville Skyttä <ville.skytta at iki.fi> - 0:0.14.2-0.fdr.2.b
- Split into mad, libmad, -devel, libid3tag and -devel packages (#187).
- Provide mp3-cmdline virtual package and alternative.
- Build shared library.

* Thu Feb 20 2003 Ville Skyttä <ville.skytta at iki.fi> - 0.14.2b-1.fedora.1
- First Fedora release, based on Matthias Saou's work.

* Fri Sep 27 2002 Matthias Saou <matthias.saou@est.une.marmotte.net>
- Rebuild for Red Hat Linux 8.0 (missing because of license issues).
- Spec file cleanup.

* Tue Mar 12 2002 Bill Nottingham <notting@redhat.com> 0.14.2b-3
- ship libid3tag too

* Thu Feb 21 2002 Bill Nottingham <notting@redhat.com>
- rebuild

* Mon Jan 28 2002 Bill Nottingham <notting@redhat.com>
- split libmad off into a separate package

