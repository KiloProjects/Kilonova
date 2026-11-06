# Spec Delta

## Purpose

Defines how lists of problems are shown and searched — the problem search page, the latest-problems block, tag pages and a user's solved/attempted lists — including filtering, pagination and shareable URLs.

## ADDED Requirements

### Requirement: Problem tables render on the server with server pagination
The latest-problems block on the home page, the problem list on a tag page, and a profile's solved and attempted problem lists SHALL be rendered in the page response, showing for each problem its name and id, publication badge for editors, score for logged-in viewers, solved/attempted counts, and source or tags where the listing shows them. Moving between pages SHALL load the requested page from the server and SHALL be addressable by URL; the two lists on a profile SHALL paginate independently.

#### Scenario: Second page of solved problems
- **WHEN** a visitor opens page 2 of a user's solved problems
- **THEN** the solved list shows page 2, the attempted list is unchanged, and copying the URL reproduces that state

#### Scenario: Tag page with tags column
- **WHEN** a viewer toggles "show tags" on a tag page
- **THEN** the tags column appears without a request to the server

### Requirement: Problem search is URL-driven and debounced
The problem search page SHALL accept its whole query (text, author, published state, tag groups, problem list including sublists, statement language, ordering and direction, page) as URL parameters and render the matching results in the page response. Changing a filter SHALL update the results and the URL without a full page navigation; typing in the text box SHALL wait for a short pause before searching, and a slower earlier response SHALL never replace the results of a later query. Back/forward navigation SHALL restore earlier result pages. The tag-group picker SHALL keep working as today.

#### Scenario: Typing quickly
- **WHEN** a user types "graph" one letter at a time
- **THEN** results for "graph" are shown, not results for any shorter prefix, and at most a few searches are issued

#### Scenario: Shared link
- **WHEN** a user opens a copied search URL with tags and ordering set
- **THEN** the page loads with those filters selected and the matching results already rendered

#### Scenario: Deep list filter
- **WHEN** the query filters by a problem list
- **THEN** the page names that list without an extra lookup by the browser
