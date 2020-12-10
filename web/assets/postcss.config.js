const cssnano = require("cssnano")
const purgecss = require("@fullhuman/postcss-purgecss")
const glob = require("glob")

module.exports = {
  plugins: [
		require("postcss-import"),
	  	require("tailwindcss"),
		process.env.NODE_ENV === 'production' ? require("autoprefixer") : null,
		process.env.NODE_ENV === 'production' ? cssnano({preset: "default"}) : null,
	  	purgecss({
			content: glob.sync('../templ/**/*', {nodir: true}),
			defaultExtractor: content => content.match(/[\w-/:]+(?<!:)/g) || []
		})
  ]
}

