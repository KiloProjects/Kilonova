# Makefile for Isolate
# (c) 2015--2019 Martin Mares <mj@ucw.cz>
# (c) 2017 Bernard Blackham <bernard@blackham.com.au>

# minified to the bare essentials by Vasiluta Mihai-Alexandru <alexv@siluta.ro>

PREFIX = $(DESTDIR)/usr/local
VARPREFIX = $(DESTDIR)/var/local
CONFIGDIR = $(PREFIX)/etc
CONFIG = $(CONFIGDIR)/isolate
BINDIR = $(PREFIX)/bin
DATAROOTDIR = $(PREFIX)/share
DATADIR = $(DATAROOTDIR)
BOXDIR = $(VARPREFIX)/lib/isolate

CC=gcc
CFLAGS=-std=gnu99 -Wall -Wextra -Wno-parentheses -Wno-unused-result -Wno-missing-field-initializers -Wstrict-prototypes -Wmissing-prototypes -D_GNU_SOURCE -DCONFIG_FILE='"$(CONFIG)"'
LIBS=-lcap


isolate: isolate.o util.o rules.o cg.o config.o
	$(CC) $(LDFLAGS) -o $@ $^ $(LIBS)
	
%.o: %.c isolate.h config.h
	$(CC) $(CFLAGS) -c -o $@ $<

clean:
	rm -f *.o

install: isolate 
	install -d $(BINDIR) $(BOXDIR) $(CONFIGDIR)
	install -m 4755 isolate $(BINDIR)
	install -m 644 default.cf $(CONFIG)

.PHONY: all clean install