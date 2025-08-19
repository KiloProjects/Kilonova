---
title: About statements
---


# Introduction

There are multiple types of statements in Kilonova. To create a statement, you usually create an attachment to the
problem, with the following format: `statement-{language}[-{type}].{format}`. `-{type}` is an optional parameter, but is
useful to clarify specific types.

# Naming scheme

Examples of common attachment names include:

- `statement-ro.md` - Romanian, Markdown statement.
- `statement-en-llm.md` - English, Markdown statement, machine-translated from the original (therefore the type is
  `llm`)
- `statement-ro.pdf` - Romanian, PDF statement. These are usually the original contest PDFs and are stored for archival
  purposes only.

Markdown statements like `statement-ro.md` and `statement-en.md` can be considered the authoritative source of the
statement.

# How it relates to the API

The API interprets the attachment names and generates a curated list of statement **variants**. The variants are
enumerated based on the attachment names.

In addition, for convenience, the URL to the raw statement data is given as the `permalink` field in the API response.


# Embedding guide

!!! note

    This part mainly applies to people that wish to embed Kilonova statements in non-Kilonova-specific contexts.


Kilonova's Markdown variant has multiple plugins that make it easy to write. These are non-standard, however. Instead, the API lets you obtain an already rendered HTML version of the statement using the `renderURL` field.

However, for various elements to render correctly, you need to include the following CSS in the context (given here in Tailwind syntax, assuming that `.statement-content` points to the `div` ):

```css
.statement-content img[data-imgalign="left"] {
    @apply px-2 lg:px-4 w-full lg:w-auto;
}

.statement-content img[data-imgalign="center"] {
    @apply px-2 lg:px-4 w-full md:w-auto mx-auto;
}

.statement-content img[data-imgalign="right"] {
    @apply pl-2 lg:pl-4 float-right max-w-[50%] lg:max-w-[30%];
    clear: both;
}

.statement-content img[data-imginline] {
    display: inline;
}
```

This is how we do it on the website, but the details may, of course, vary. What matters is that the images are properly aligned/inlined.

In addition, make sure that bulleted lists can render correctly, since most CSS resets and frameworks disable this behaviour by default for consistency.

Some statements also make use of tables but skip the header (leaving it empty). To make it render more nicely, you can also add this bit of CSS: 

```css
	.statement-content thead th:empty {
		display: none;
	}
```