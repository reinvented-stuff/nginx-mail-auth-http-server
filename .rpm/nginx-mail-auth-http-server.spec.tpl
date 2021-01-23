Name:           nginx-mail-auth-http-server
Version:        __VERSION__
Release:        1%{?dist}
Summary:        A reinvented server for Nginx mail_auth_http module

License:        MIT
URL:            https://reinvented-stuff.com/nginx-mail-auth-http-server
Source0:        __SOURCE_TARGZ_FILENAME__


%description
nginx-mail-auth-http-server provides a simple way to authorise
your mail server clients and direct the connections to a correct
mail backend using Nginx as a reverse proxy.

%prep
%setup -q


%build
make %{?_smp_mflags} build


%install
rm -rf $RPM_BUILD_ROOT
%make_install


%files
%attr(755, root, root) /usr/bin/nginx-mail-auth-http-server
%attr(644, root, root) /usr/share/doc/nginx-mail-auth-http-server-__VERSION__/README.md
%doc



%changelog
