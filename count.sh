#!/usr/bin/bash

while read key; do
	cnt=$(rg -l "[\"\`']$key[\"\`']" web/ | wc -l)
	#cnt=$(rg -l $key web/ | wc -l)
	echo "$cnt $key"
done < <(grep '^\[.*\]' translations.toml | sed 's/\[\(.*\)\]/\1/g')
