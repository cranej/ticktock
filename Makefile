.POSIX:

ifndef PREFIX
  PREFIX = /usr/local
endif
ifndef MANPREFIX
  MANPREFIX = $(PREFIX)/share/man
endif

install:
	mkdir -p $(DESTDIR)$(PREFIX)/bin
	go build -o $(DESTDIR)$(PREFIX)/bin/
	cp dtick $(DESTDIR)$(PREFIX)/bin/
	mkdir -p $(DESTDIR)$(MANPREFIX)/man1
	cp -f ticktock.1 $(DESTDIR)$(MANPREFIX)/man1/ticktock.1
	chmod 644 $(DESTDIR)$(MANPREFIX)/man1/ticktock.1

uninstall:
	rm -f $(DESTDIR)$(PREFIX)/bin/ticktock $(DESTDIR)$(PREFIX)/bin/dtick
	rm -f $(DESTDIR)$(MANPREFIX)/man1/ticktock.1

.PHONY: install uninstall
