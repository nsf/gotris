gotris: gotris.8 
	8l -o gotris gotris.8
	
SOURCES=\
	gotris.go

gotris.8: $(SOURCES)
	8g $(SOURCES)

clean:
	rm -rf *.8 gotris
