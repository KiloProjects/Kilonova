// NOTE: This document is meant to be compiled only with run.sh

#set page("a4", margin: 1.5cm, numbering: "1 / 1")

#set text(size: 13pt)

#let data = csv("in.csv", row-type: dictionary).map(entry => table.cell(breakable: false)[
    #text(weight: "bold", size: 1.25em)[#entry.at("Team Name")] \
    Username: #text(font: "DejaVu Sans Mono")[#entry.at("Username")] \
    Password: #text(font: "DejaVu Sans Mono")[#entry.at("Password")]
])



#table(
    columns: 2,
    inset: 10pt,
    stroke: 2pt,
    ..data
)
