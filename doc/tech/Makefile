all: tech.pdf

USEDFIGS=$(shell grep -h includegraph *.tex|grep -v '\.\./format' |egrep -v '^[ \t]*%.*'|sed -E 's,.*\{figs/([^}]+)}.*,\1,g'|sort -u)

TEX=$(wildcard *.tex)
PDFS=$(wildcard figs/*.pdf)
DOTS=$(shell echo figs/gv_*.dot)
ONEDOT=$(shell echo $(DOTS)|sed -e 's/[\t ].*$$//g')
SCRIPTS=$(wildcard scripts/*.sh)

SVGS=$(shell echo figs/svg_*.svg)
ONESVG=$(shell echo figs/svg_*.svg|sed -e 's/[\t ].*$$//g')

GENPDFS=$(wildcard figs/gv_*.pdf figs/svg_*.pdf)

TMPDIR:=$(shell mktemp --tmpdir -d maketech.XXXXXXX)

usedfigs:
	@echo $(USEDFIGS)|tr ' ' '\n'


tech.pdf: tech.tex $(TEX) $(PDFS) $(SCRIPTS) tech.bib src/example.rul
	@echo '#making dots, svgs and pdf'
	@! test -e $(ONEDOT) || $(MAKE) `echo $(DOTS)|sed 's/\.dot/.pdf/g'`
	@! test -e $(ONESVG) || $(MAKE) `echo $(SVGS)|sed 's/\.svg/.pdf/g'`
	@echo 'latexmk '$<
	@latexmk -pdf -latex="pdflatex -interaction=nonstopmode" -use-make $< </dev/null 2>&1|awk '/^l\.[0-9]/,/^!/{print }'| sed -E 's/^l\.([0-9]+)/'$<':\1/g' 1>&2
	@latexmk -pdf -latex="pdflatex -interaction=nonstopmode" -use-make $< </dev/null 2>&1|awk '/^l\.[0-9]/,/^!/{print }'| sed -E 's/^l\.([0-9]+)/'$<':\1/g' 1>&2

figs/gv_%.pdf: figs/gv_%.dot
	dot -Tpdf -o$@ $<

figs/svg_%.pdf: figs/svg_%.svg
	inkscape -D $< --export-filename=$@;


.PHONY: clean nuke view prep dist usedfigs

view: all
	$(MAKE) clean
	evince tech.pdf > /dev/null	2>&1

clean:
	latexmk -c
	rm -f tech.nav tech.snm tech.fdb_latexmk tech*.vrb tech.synctex* tech.lol tech.bbl
	rm -rf /tmp/maketech.* techgood.pdf

nuke: clean
	latexmk -C
	rm -f $(GENPDFS) *.bbl *.aux *.lol

prep: nuke
	$(MAKE) && cp tech.pdf $(TMPDIR) && $(MAKE) nuke
	mv $(TMPDIR)/tech.pdf .
	rmdir $(TMPDIR)

