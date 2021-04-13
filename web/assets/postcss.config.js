const cssnano = require("cssnano")
const purgecss = require("@fullhuman/postcss-purgecss")
const glob = require("glob")

module.exports = {
  plugins: [
		require("postcss-import"),
	  	require("tailwindcss"),
		require("autoprefixer"),
//	  	process.env.NODE_ENV === 'production' ? purgecss({
//			content: glob.sync('../templ/**/*', {nodir: true}),
//			defaultExtractor: content => content.match(/[\w-/:]+(?<!:)/g) || [],
//			safelist: [/^notyf/]
//		}) : null,
//
		process.env.NODE_ENV === 'production' ? cssnano({preset: "default"}) : null,
  ]
}

