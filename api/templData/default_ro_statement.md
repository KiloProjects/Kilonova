Se dau două numere întregi $a$ și $b$. Să se calculeze suma celor două numere.

# Date de intrare

{{if eq .InputFile "stdin"}}
Pe prima linie se găsesc două numere întregi, $a$ și $b$.
{{else}}
Pe prima linie a fișierului de intrare `{{.InputFile}}` se găsesc două numere întregi, $a$ și $b$.
{{end}}

# Date de ieșire

{{if eq .OutputFile "stdout"}}
Pe prima linie se va găsi un singur număr întreg, suma celor două numere.
{{else}}
Pe prima linie a fișierului de ieșire `{{.OutputFile}}` se va găsi un singur număr întreg, suma celor două numere.
{{end}}

# Restricții și precizări

* $1 \leq a, b \leq 1 \ 000 \ 000$;
* Aceasta este o structură de enunț, poate fi schimbată cum vreți, însă vă rugăm să mențineți acest format. În viitor, va fi publicat un ghid care va oferi mai multe informații privind standarde de formatare.

# Exemplul 1

`{{.InputFile}}`
```
1 2
```

`{{.OutputFile}}`
```
3
```

## Explicație

După cum se vede, $1 + 2 = 3$.

# Exemplul 2


`{{.InputFile}}`
```
5 -1
```

`{{.OutputFile}}`
```
4
```

## Explicație

$5 + (-1) = 4$, după cum știm.

