// NOTE: Compiled by contestgen.PrintablePDF (kn printable-contestants, /admin/contest_gen).

#set page("a4", margin: 1.5cm, numbering: "1 / 1")

#set text(size: 13pt)

#let data = csv("in.csv", row-type: dictionary).map(entry => table.cell(breakable: false)[
    #let team = entry.at("Team Name", default: "")
    #if team.len() > 0 {[ #text(weight: "bold", size: 1.25em)[#team] \ ]}
    Username: #text(font: "DejaVu Sans Mono")[#entry.at("Username")] \
    Password: #text(font: "DejaVu Sans Mono")[#entry.at("Password")]
])

#table(
    columns: 2,
    inset: 10pt,
    stroke: 2pt,
    ..data
)
