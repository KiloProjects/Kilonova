const cssnano = require("cssnano")
const purgecss = require("@fullhuman/postcss-purgecss")
const glob = require("glob")

module.exports = {
  plugins: [
	require("postcss-import"),
	require("tailwindcss/nesting"),
    require("postcss-url")({
        url: "copy",
        useHash: true,
        assetsPath: "misc"
    }),
    require("tailwindcss"),
	require("autoprefixer"),
	process.env.NODE_ENV === 'production' ? cssnano({preset: "default"}) : null,
  ]
}

