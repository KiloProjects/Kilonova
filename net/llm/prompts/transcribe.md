# AI Guidelines

These are the AI guidelines to formatting PDF statements. Please know that you only have to convert the PDF -> Markdown. You do NOT need to do anything else. Keep it simple

You MUST use your vision abilities to understand the PDF files.

## General information

* For Markdown statements, LaTeX is used to render formulas, e.g. $\displaystyle \sum_{i=1}^{n} \frac{a^{b+c}}{d^e-f_i} \le \sqrt{2}$ is `$\displaystyle \sum_{i=1}^{n} \frac{a^{b+c}}{d^e-f_i} \le \sqrt{2}$ `; use `\displaystyle` for big formulas. A list of supported features can be found [here](https://katex.org/docs/supported.html)
* **Statements written in Romanian must use diacritics, even if they were not present in the source PDF**

## Formatting tips
* You can make text bold or italic using Markdown (*text*, **text**, ***text***). You can underline, strikethrough, or make coloured text using LaTeX ($\text{\underline{YES} }\color{red}\sout{NO}$). Always use `\text{}` when you intend to write text instead of formulas; $a\text{ mod }b$ instead of $a\ mod\ b$. 
	* Note that when you aren't writing a formula, or there's two formulas separated by text, you generally shouldn't make the whole thing a LaTeX block. 
		* For example, do like this: $1 \leq a_i \leq 1 \ 000$, where $1 \leq i \leq N$;
		* Instead of $1 \leq a_i \leq 1 \ 000 \text{, where } 1 \leq i \leq N$.
* You can create links using `[text](url)`. 
* To display images you must use a custom markdown extension with the syntax `~[image.png]`, where `image.png` is an attachment of the current problem. You can also change the alignment (center or right) or width (in percentages or other CSS units): `~[image.png|align=center|width=50%]`

## Styling conventions 
* Variables, equations, and numbers must be written using LaTeX and not placed inside a markdown code block ($N$ numbers instead of `N` numbers)
* An exception to the above rule is when you mention file names (you would write "`sum.in`" in markdown) or function names in interactive problems (`compare`). You should also use code blocks if you want to emphasise strings or specific characters, usually for output formatting (`ABABAAABAA`; `,`, `.`, `*` represent cells; `NO`).
* There should be a space **AFTER** each punctuation sign (`,`, `:`, etc.), and **NOT BEFORE** it. (`1, 2, ..., n` as opposed to `1,2,...,n` or `1 , 2 , ... , n`)
* Every group of three digits must be separated by a space ($10 \ 000 \ 000$ instead of $10,000,000$; `10 \ 000 \ 000` in LaTeX)  
* Replace asterisks `*` which stand for multiplication with `\cdot` ($\cdot$) or `\times` ($\times$), based on the appropriate context.


## Transcription conventions

There is a specific name and structure for the headings we use, regardless of what is inside the PDF file. When there is a single example, we do not specify the example number. 

When there are two or more examples, we must specify in each heading the example number. 

When the input is through the terminal, the file names have to be `stdin` / `stdout`. Otherwise, use the given problem slug (say, 'martisoare') and it has to be something like `martisoare.in` / `martisoare.out`.

When there is no specific language given in a codeblock, do NOT put a language (like `text`), just leave the \`\`\` line alone. Use a language tag ONLY if it's a snippet of code.

Even though in the PDF files examples are usually given as tables with 2 or 3 columns (input, output, possible explanation), you have to transcribe them using a structure like so:

```md
# Example 1

`stdin`
```
1 2
```

`stdout`
```
3
```

## Explanation

The sum of 1 and 2 is 3

```


- Romanian headings:
    - `Cerință`
    - `Date de intrare`
    - `Date de ieșire`
    - `Restricții și precizări`
    - `Exemplu` / `Exemplul <num>`
        - `Explicație`
- English headings:
    - `Task`
    - `Input Data`
    - `Output Data`
    - `Cosntraints and clarifications`
    - `Example` / `Example <num>`
        - `Explanation`



# Example statement

The following is an example statement, to give a better sense of the structure you should follow.

```md
Gică și Lică lucrează la o fabrică de jucării, în schimburi diferite. Anul acesta patronul fabricii a hotărât să confecționeze și mărțișoare. Mărțișoarele gata confecționate sunt puse în cutii numerotate consecutiv. Cutiile sunt aranjate în ordinea strict crescătoare și consecutivă a numerelor de pe acestea. Gică trebuie să ia, în ordine, fiecare cutie, să lege la fiecare mărțișor câte un șnur alb-roșu și apoi să le pună la loc în cutie.

În fiecare schimb, Gică scrie pe o tablă magnetică, utilizând cifre magnetice, în ordine strict crescătoare, numerele cutiilor pentru care a legat șnururi la mărțișoare. Când se termină schimbul lui Gică, Lică, care lucrează în schimbul următor, vine și ambalează cutiile cu numerele de pe tablă și le trimite la magazine. Totul merge ca pe roate, până într-o zi, când, două cifre de pe tablă se demagnetizează și cad, rămânând două locuri goale. Lică observă acest lucru, le ia de jos și le pune la întâmplare pe tablă, în cele două locuri goale. Singurul lucru de care ține cont este acela că cifra $0$ nu poate fi prima cifră a unui număr.

~[martisoare.png|align=right]

# Cerință

Scrieți un program care să citească numerele naturale $N$ (reprezentând numărul de numere scrise pe tablă) și $c_1$, $c_2$, $\dots$, $c_N$ (reprezentând numerele scrise, în ordine, pe tablă, după ce Lică a completat cele două locuri goale cu cifrele căzute) și care să determine:

* cele două cifre care au fost schimbate între ele, dacă, după ce au completat locurile goale, acestea au schimbat șirul numerelor scrise de Gică;
* numărul maxim scris pe tablă de Gică.

# Date de intrare

Fișierul de intrare `martisoare.in` conține pe prima linie numărul natural $N$ reprezentând numărul de numere de pe tablă. A doua linie a fișierului conține, în ordine, cele $N$ numere $c_1$, $c_2$, $\dots$, $c_N$, separate prin câte un spațiu, reprezentând, în ordine, numerele existente pe tablă, după ce Lică a completat cele două locuri libere cu cifrele căzute.

# Date de ieșire

Fișierul de ieșire `martisoare.out` va conține pe prima linie două cifre, în ordine crescătoare, separate printr-un spațiu, reprezentând cele două cifre care au fost schimbate între ele sau `0 0` în cazul în care cele două cifre magnetice căzute, după ce au fost puse înapoi pe tablă, nu au schimbat șirul numerelor scrise de Gică. A doua linie va conține un număr reprezentând numărul maxim din secvența de numere consecutive scrisă de Gică pe tablă.

# Restricții și precizări

* $4 \leq N \leq 100 \ 000$;
* $1 \leq c_i \leq 100 \ 000$;
* $N$, $c_1$, $c_2$, $\dots$, $c_N$ sunt numere naturale;
* cele două cifre care cad de pe tablă pot proveni din același număr;
* Pentru rezolvarea cerinței a) se acordă $60\%$ din punctaj, iar pentru cerința b) se acordă $40\%$ din punctaj.

# Exemplul 1

`martisoare.in`
```
5
65 22 27 28 29
```

`martisoare.out`
```
2 6
29
```

## Explicație

Gică a scris pe tablă, în ordine, numerele: $25$, $26$, $27$, $28$, $29$

Au fost schimbate între ele cifra $2$ din primul număr și cifra $6$ din al doilea număr. Cel mai mare număr scris de Gică pe tablă este $29$.

# Exemplul 2


`martisoare.in`
```
4
95 96 97 89
```

`martisoare.out`
```
8 9
98
```

## Explicație

Gică a scris pe tablă, în ordine, numerele: $95$, $96$, $97$, $98$

Au fost schimbate între ele cifrele ultimului număr. Cel mai mare număr scris de Gică pe tablă este $98$.

# Exemplul 3

`martisoare.in`
```
5
35 36 37 38 39
```

`martisoare.out`
```
0 0
39
```

## Explicație

Gică a scris pe tablă, în ordine, numerele: $35$, $36$, $37$, $38$, $39$

Șirul numerelor nu a fost schimbat, cel mai mare număr fiind $39$.
```