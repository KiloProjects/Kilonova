    const rewriteRules = [
        {
            pattern: /\\textbf{([^}]*)}/g,
            replacement: "**$1**"
        },
        {
            pattern: /\\textsc{([^}]*)}/g,
            replacement: "**$1**"
        },
        {
            pattern: /\\emph{([^}]*)}/g,
            replacement: "**$1**"
        },
        {
            pattern: /\\textit{([^}]*)}/g,
            replacement: "_$1_"
        },
        {
            pattern: /\\texttt{([^}]*)}/g,
            replacement: "`$1`"
        },
        {
            pattern: /\\item (.*)/g,
            replacement: "* $1"
        },
        {
            pattern: /\\section{([^}]*)}/g,
            replacement: "# $1"
        },
        {
            pattern: /\\np{([0-9]*)}/g,
            replacement: (match, p1) => {
                const spl = p1.split('').reverse().join('')
                    .match(/.{1,3}/g).map(group => group.split('').reverse().join('')).reverse();
                console.log(match, p1, spl)
                return spl.join(' \\ ')
            }
        },
        {
            pattern: /\\restr{(.*)}{(.*)}/g,
            replacement: "| x | $1 | $2 |",
        },
    ]

    function rewriteLaTeXMarkdown(snippet) {
        for(let rule of rewriteRules) {
            snippet = snippet.replaceAll(rule.pattern, rule.replacement)
        }
        return snippet
    }
